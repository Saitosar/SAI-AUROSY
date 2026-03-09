package tasks

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// SQLStore is a persistent task store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed task store.
// driver: "sqlite" or "pgx" (PostgreSQL); db: existing connection (migrations must already be run).
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

// Create adds a new task.
func (s *SQLStore) Create(t *Task) error {
	now := time.Now()
	t.UpdatedAt = now
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	payload := ""
	if len(t.Payload) > 0 {
		payload = string(t.Payload)
	}
	var completedAt interface{}
	if t.CompletedAt != nil {
		completedAt = t.CompletedAt
	}
	_, err := s.db.Exec(s.ph("INSERT INTO tasks (id, robot_id, type, scenario_id, payload, status, created_at, updated_at, completed_at, operator_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"),
		t.ID, t.RobotID, t.Type, nullIfEmpty(t.ScenarioID), payload, string(t.Status), t.CreatedAt, t.UpdatedAt, completedAt, nullIfEmpty(t.OperatorID))
	return err
}

// Get returns a task by ID.
func (s *SQLStore) Get(id string) (*Task, error) {
	var t Task
	var payload []byte
	var scenarioID, operatorID sql.NullString
	var completedAt sql.NullTime
	err := s.db.QueryRow(s.ph("SELECT id, robot_id, type, scenario_id, payload, status, created_at, updated_at, completed_at, operator_id FROM tasks WHERE id=?"),
		id,
	).Scan(&t.ID, &t.RobotID, &t.Type, &scenarioID, &payload, &t.Status, &t.CreatedAt, &t.UpdatedAt, &completedAt, &operatorID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(payload) > 0 {
		t.Payload = json.RawMessage(payload)
	}
	if scenarioID.Valid {
		t.ScenarioID = scenarioID.String
	}
	if operatorID.Valid {
		t.OperatorID = operatorID.String
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return &t, nil
}

// List returns tasks matching filters.
func (s *SQLStore) List(filters ListFilters) ([]Task, error) {
	q := "SELECT id, robot_id, type, scenario_id, payload, status, created_at, updated_at, completed_at, operator_id FROM tasks WHERE 1=1"
	args := []interface{}{}
	if filters.RobotID != "" {
		q += " AND robot_id=?"
		args = append(args, filters.RobotID)
	}
	if filters.Status != "" {
		q += " AND status=?"
		args = append(args, string(filters.Status))
	}
	q += " ORDER BY created_at DESC"
	rows, err := s.db.Query(s.ph(q), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task
		var payload []byte
		var scenarioID, operatorID sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.RobotID, &t.Type, &scenarioID, &payload, &t.Status, &t.CreatedAt, &t.UpdatedAt, &completedAt, &operatorID); err != nil {
			continue
		}
		if len(payload) > 0 {
			t.Payload = json.RawMessage(payload)
		}
		if scenarioID.Valid {
			t.ScenarioID = scenarioID.String
		}
		if operatorID.Valid {
			t.OperatorID = operatorID.String
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateStatus updates task status.
func (s *SQLStore) UpdateStatus(id string, status Status) error {
	_, err := s.db.Exec(s.ph("UPDATE tasks SET status=?, updated_at=? WHERE id=?"), string(status), time.Now(), id)
	return err
}

// UpdateStatusAndCompletedAt updates status and sets completed_at.
func (s *SQLStore) UpdateStatusAndCompletedAt(id string, status Status, completedAt time.Time) error {
	_, err := s.db.Exec(s.ph("UPDATE tasks SET status=?, updated_at=?, completed_at=? WHERE id=?"), string(status), time.Now(), completedAt, id)
	return err
}

// HasRunningForRobot returns true if the robot has a task in running status.
func (s *SQLStore) HasRunningForRobot(robotID string) (bool, error) {
	var count int
	err := s.db.QueryRow(s.ph("SELECT COUNT(*) FROM tasks WHERE robot_id=? AND status=?"), robotID, string(StatusRunning)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
