package tenants

import "errors"

var ErrNotFound = errors.New("tenant not found")

// Store is the tenant store interface.
type Store interface {
	List() ([]Tenant, error)
	Get(id string) (*Tenant, error)
	Create(t *Tenant) error
	Update(t *Tenant) error
	Delete(id string) error
}
