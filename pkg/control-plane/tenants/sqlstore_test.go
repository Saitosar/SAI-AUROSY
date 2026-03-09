package tenants

import (
	"testing"

	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
)

func TestSQLStore_ListGet(t *testing.T) {
	reg, err := registry.NewSQLStore("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("NewSQLStore: %v", err)
	}
	defer reg.Close()
	s := NewSQLStore(reg.DB(), "sqlite")

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 1 {
		t.Errorf("List: expected at least 1 (default), got %d", len(list))
	}

	got, err := s.Get("default")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get default: expected tenant, got nil")
	}
	if got.ID != "default" || got.Name != "Default" {
		t.Errorf("Get default: got %+v", got)
	}
}

func TestSQLStore_CreateUpdateDelete(t *testing.T) {
	reg, err := registry.NewSQLStore("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("NewSQLStore: %v", err)
	}
	defer reg.Close()
	s := NewSQLStore(reg.DB(), "sqlite")

	t1 := &Tenant{ID: "t1", Name: "Tenant 1"}
	if err := s.Create(t1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("t1")
	if err != nil || got == nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.Name != "Tenant 1" {
		t.Errorf("Get: expected Name Tenant 1, got %s", got.Name)
	}

	t1.Name = "Tenant 1 Updated"
	if err := s.Update(t1); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get("t1")
	if got.Name != "Tenant 1 Updated" {
		t.Errorf("Get after Update: expected Tenant 1 Updated, got %s", got.Name)
	}

	if err := s.Delete("t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ = s.Get("t1")
	if got != nil {
		t.Error("Get after Delete: expected nil")
	}
}
