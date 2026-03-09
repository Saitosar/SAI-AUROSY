package tenants

import (
	"testing"
)

func TestMemoryStore_ListGet(t *testing.T) {
	s := NewMemoryStore()
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List: expected 1 (default), got %d", len(list))
	}
	got, err := s.Get("default")
	if err != nil || got == nil {
		t.Fatalf("Get default: %v", err)
	}
	if got.ID != "default" {
		t.Errorf("Get: got %+v", got)
	}
}

func TestMemoryStore_CreateUpdateDelete(t *testing.T) {
	s := NewMemoryStore()
	t1 := &Tenant{ID: "t1", Name: "Tenant 1"}
	if err := s.Create(t1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	list, _ := s.List()
	if len(list) != 2 {
		t.Errorf("List after Create: expected 2, got %d", len(list))
	}
	got, _ := s.Get("t1")
	if got == nil || got.Name != "Tenant 1" {
		t.Errorf("Get: got %+v", got)
	}
	t1.Name = "Updated"
	if err := s.Update(t1); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get("t1")
	if got.Name != "Updated" {
		t.Errorf("Get after Update: got %s", got.Name)
	}
	if err := s.Delete("t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ = s.Get("t1")
	if got != nil {
		t.Error("Get after Delete: expected nil")
	}
}
