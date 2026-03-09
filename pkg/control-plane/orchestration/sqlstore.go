package orchestration

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// SQLRunStore is a persistent workflow run store.
type SQLRunStore struct {
	db     *sql.DB
	driver string
}

// NewSQLRunStore creates a SQL-backed run store.
// Migrations (000005) must already be applied to db.
func NewSQLRunStore(db *sql.DB, driver string) *SQLRunStore {
	return &SQLRunStore{db: db, driver: driver}
}

func (s *SQLRunStore) ph(q string) string {
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

// Create adds a workflow run.
func (s *SQLRunStore) Create(run *WorkflowRun) error {
	now := time.Now()
	run.UpdatedAt = now
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	_, err := s.db.Exec(s.ph("INSERT INTO workflow_runs (id, workflow_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"),
		run.ID, run.WorkflowID, string(run.Status), run.CreatedAt, run.UpdatedAt)
	return err
}

// List returns all workflow runs.
func (s *SQLRunStore) List() ([]WorkflowRun, error) {
	rows, err := s.db.Query("SELECT id, workflow_id, status, created_at, updated_at FROM workflow_runs ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkflowRun
	for rows.Next() {
		var run WorkflowRun
		if err := rows.Scan(&run.ID, &run.WorkflowID, &run.Status, &run.CreatedAt, &run.UpdatedAt); err != nil {
			continue
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

// Get returns a workflow run by ID.
func (s *SQLRunStore) Get(id string) (*WorkflowRun, error) {
	var run WorkflowRun
	err := s.db.QueryRow(s.ph("SELECT id, workflow_id, status, created_at, updated_at FROM workflow_runs WHERE id=?"),
		id,
	).Scan(&run.ID, &run.WorkflowID, &run.Status, &run.CreatedAt, &run.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(s.ph("SELECT task_id, step_index FROM workflow_run_tasks WHERE workflow_run_id=? ORDER BY step_index"), id)
	if err != nil {
		return &run, nil
	}
	defer rows.Close()
	for rows.Next() {
		var t WorkflowRunTask
		if err := rows.Scan(&t.TaskID, &t.StepIndex); err != nil {
			continue
		}
		run.Tasks = append(run.Tasks, t)
	}
	return &run, rows.Err()
}

// UpdateStatus updates the run status.
func (s *SQLRunStore) UpdateStatus(id string, status WorkflowRunStatus) error {
	_, err := s.db.Exec(s.ph("UPDATE workflow_runs SET status=?, updated_at=? WHERE id=?"), string(status), time.Now(), id)
	return err
}

// AddTask adds a task to a workflow run.
func (s *SQLRunStore) AddTask(runID, taskID string, stepIndex int) error {
	_, err := s.db.Exec(s.ph("INSERT INTO workflow_run_tasks (workflow_run_id, task_id, step_index) VALUES (?, ?, ?)"), runID, taskID, stepIndex)
	return err
}
