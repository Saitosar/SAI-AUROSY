package scenarios

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("scenario not found")

// Store is the scenario store interface.
type Store interface {
	List(ctx context.Context) ([]Scenario, error)
	Get(ctx context.Context, id string) (*Scenario, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Scenario, error)
	GetByTenant(ctx context.Context, id, tenantID string) (*Scenario, error)
	Create(ctx context.Context, s *Scenario) error
	Update(ctx context.Context, s *Scenario) error
	Delete(ctx context.Context, id string) error
}
