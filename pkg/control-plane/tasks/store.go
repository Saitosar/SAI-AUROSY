package tasks

import (
	"sync"
	"time"
)

// ListFilters filters tasks for List.
type ListFilters struct {
	RobotID string
	Status  Status
}

// Store is the task store interface.
type Store interface {
	Create(t *Task) error
	Get(id string) (*Task, error)
	List(filters ListFilters) ([]Task, error)
	UpdateStatus(id string, status Status) error
	UpdateStatusAndCompletedAt(id string, status Status, completedAt time.Time) error
	HasRunningForRobot(robotID string) (bool, error)
}

// MemoryStore is an in-memory task store.
type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

// NewMemoryStore creates a new in-memory task store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]*Task),
	}
}

// Create adds a new task.
func (s *MemoryStore) Create(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	t.UpdatedAt = now
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	s.tasks[t.ID] = t
	return nil
}

// Get returns a task by ID.
func (s *MemoryStore) Get(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

// List returns tasks matching filters.
func (s *MemoryStore) List(filters ListFilters) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Task
	for _, t := range s.tasks {
		if filters.RobotID != "" && t.RobotID != filters.RobotID {
			continue
		}
		if filters.Status != "" && t.Status != filters.Status {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}

// UpdateStatus updates task status.
func (s *MemoryStore) UpdateStatus(id string, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return nil
}

// UpdateStatusAndCompletedAt updates status and sets completed_at.
func (s *MemoryStore) UpdateStatusAndCompletedAt(id string, status Status, completedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	t.CompletedAt = &completedAt
	return nil
}

// HasRunningForRobot returns true if the robot has a task in running status.
func (s *MemoryStore) HasRunningForRobot(robotID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		if t.RobotID == robotID && t.Status == StatusRunning {
			return true, nil
		}
	}
	return false, nil
}
