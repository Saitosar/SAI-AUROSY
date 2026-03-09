package edges

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/pkg/hal"
)

// SQLStore is a SQL-backed edge store.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL edge store.
func NewSQLStore(db *sql.DB, driver string) *SQLStore {
	return &SQLStore{db: db, driver: driver}
}

func (s *SQLStore) ph(q string) string {
	if s.driver != "pgx" {
		return q
	}
	var b strings.Builder
	n := 1
	for _, r := range q {
		if r == '?' {
			b.WriteString("$")
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// UpsertEdge creates or updates an edge record (on heartbeat).
func (s *SQLStore) UpsertEdge(ctx context.Context, e *Edge) error {
	now := time.Now()
	e.UpdatedAt = now
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	_, err := s.db.ExecContext(ctx, s.ph(`
		INSERT INTO edges (id, last_heartbeat, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET last_heartbeat=?, updated_at=?
	`), e.ID, e.LastHeartbeat, e.ConfigJSON, e.CreatedAt, e.UpdatedAt, e.LastHeartbeat, e.UpdatedAt)
	return err
}

// GetEdge returns an edge by ID.
func (s *SQLStore) GetEdge(ctx context.Context, id string) (*Edge, error) {
	var e Edge
	err := s.db.QueryRowContext(ctx, s.ph(`
		SELECT id, last_heartbeat, config_json, created_at, updated_at FROM edges WHERE id=?
	`), id).Scan(&e.ID, &e.LastHeartbeat, &e.ConfigJSON, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListEdges returns all edges.
func (s *SQLStore) ListEdges(ctx context.Context) ([]Edge, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, last_heartbeat, config_json, created_at, updated_at FROM edges ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.ID, &e.LastHeartbeat, &e.ConfigJSON, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EnqueueCommand adds a command to the edge's queue.
func (s *SQLStore) EnqueueCommand(ctx context.Context, edgeID, robotID string, cmd *hal.Command) error {
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	id := uuid.New().String()
	now := time.Now()
	_, err = s.db.ExecContext(ctx, s.ph(`
		INSERT INTO edge_commands (id, edge_id, robot_id, command_json, created_at)
		VALUES (?, ?, ?, ?, ?)
	`), id, edgeID, robotID, string(cmdJSON), now)
	return err
}

// FetchAndAckPendingCommands returns pending commands for the edge and marks them as acked.
func (s *SQLStore) FetchAndAckPendingCommands(ctx context.Context, edgeID string) ([]hal.Command, error) {
	rows, err := s.db.QueryContext(ctx, s.ph(`
		SELECT id, command_json FROM edge_commands
		WHERE edge_id=? AND acked_at IS NULL
		ORDER BY created_at
	`), edgeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	var cmds []hal.Command
	for rows.Next() {
		var id, cmdJSON string
		if err := rows.Scan(&id, &cmdJSON); err != nil {
			return nil, err
		}
		var c hal.Command
		if err := json.Unmarshal([]byte(cmdJSON), &c); err != nil {
			continue
		}
		ids = append(ids, id)
		cmds = append(cmds, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now()
	for _, id := range ids {
		_, _ = s.db.ExecContext(ctx, s.ph(`UPDATE edge_commands SET acked_at=? WHERE id=?`), now, id)
	}

	return cmds, nil
}
