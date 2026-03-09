package orchestration

import (
	"sync"
	"time"
)

// RunStore stores workflow runs.
type RunStore interface {
	Create(run *WorkflowRun) error
	Get(id string) (*WorkflowRun, error)
	List() ([]WorkflowRun, error)
	UpdateStatus(id string, status WorkflowRunStatus) error
	AddTask(runID, taskID string, stepIndex int) error
}

// MemoryRunStore is an in-memory workflow run store.
type MemoryRunStore struct {
	mu   sync.RWMutex
	runs map[string]*WorkflowRun
}

// NewMemoryRunStore creates a new in-memory run store.
func NewMemoryRunStore() *MemoryRunStore {
	return &MemoryRunStore{runs: make(map[string]*WorkflowRun)}
}

// Create adds a workflow run.
func (s *MemoryRunStore) Create(run *WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	run.UpdatedAt = now
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.Tasks = make([]WorkflowRunTask, 0)
	cp := *run
	s.runs[run.ID] = &cp
	return nil
}

// List returns all workflow runs.
func (s *MemoryRunStore) List() ([]WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]WorkflowRun, 0, len(s.runs))
	for _, r := range s.runs {
		cp := *r
		cp.Tasks = make([]WorkflowRunTask, len(r.Tasks))
		copy(cp.Tasks, r.Tasks)
		out = append(out, cp)
	}
	return out, nil
}

// Get returns a workflow run by ID.
func (s *MemoryRunStore) Get(id string) (*WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	cp := *r
	cp.Tasks = make([]WorkflowRunTask, len(r.Tasks))
	copy(cp.Tasks, r.Tasks)
	return &cp, nil
}

// UpdateStatus updates the run status.
func (s *MemoryRunStore) UpdateStatus(id string, status WorkflowRunStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.runs[id]; ok {
		r.Status = status
		r.UpdatedAt = time.Now()
	}
	return nil
}

// AddTask adds a task to a workflow run.
func (s *MemoryRunStore) AddTask(runID, taskID string, stepIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.runs[runID]; ok {
		r.Tasks = append(r.Tasks, WorkflowRunTask{TaskID: taskID, StepIndex: stepIndex})
		r.UpdatedAt = time.Now()
	}
	return nil
}
