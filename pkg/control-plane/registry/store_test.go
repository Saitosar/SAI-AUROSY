package registry

import (
	"testing"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

func TestMemoryStore_AddGet(t *testing.T) {
	s := NewMemoryStore()
	r := &hal.Robot{
		ID:              "r1",
		Vendor:          "agibot",
		Model:           "X1",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
	}
	s.Add(r)
	got := s.Get("r1")
	if got == nil {
		t.Fatal("Get: expected robot, got nil")
	}
	if got.ID != "r1" || got.Vendor != "agibot" {
		t.Errorf("Get: got %+v", got)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()
	got := s.Get("nonexistent")
	if got != nil {
		t.Errorf("Get: expected nil, got %+v", got)
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()
	s.Add(&hal.Robot{ID: "r1", Vendor: "a", Model: "m", AdapterEndpoint: "e", TenantID: "t", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	s.Add(&hal.Robot{ID: "r2", Vendor: "b", Model: "m", AdapterEndpoint: "e", TenantID: "t", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	list := s.List()
	if len(list) != 2 {
		t.Errorf("List: expected 2, got %d", len(list))
	}
}

func TestMemoryStore_ListByTenant(t *testing.T) {
	s := NewMemoryStore()
	s.Add(&hal.Robot{ID: "r1", Vendor: "a", Model: "m", AdapterEndpoint: "e", TenantID: "t1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	s.Add(&hal.Robot{ID: "r2", Vendor: "b", Model: "m", AdapterEndpoint: "e", TenantID: "t1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	s.Add(&hal.Robot{ID: "r3", Vendor: "c", Model: "m", AdapterEndpoint: "e", TenantID: "t2", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if got := s.ListByTenant(""); len(got) != 3 {
		t.Errorf("ListByTenant(''): expected 3, got %d", len(got))
	}
	if got := s.ListByTenant("t1"); len(got) != 2 {
		t.Errorf("ListByTenant('t1'): expected 2, got %d", len(got))
	}
	if got := s.ListByTenant("t2"); len(got) != 1 {
		t.Errorf("ListByTenant('t2'): expected 1, got %d", len(got))
	}
	if got := s.ListByTenant("t3"); len(got) != 0 {
		t.Errorf("ListByTenant('t3'): expected 0, got %d", len(got))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	s.Add(&hal.Robot{ID: "r1", Vendor: "a", Model: "m", AdapterEndpoint: "e", TenantID: "t", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if !s.Delete("r1") {
		t.Error("Delete: expected true")
	}
	if s.Get("r1") != nil {
		t.Error("Delete: robot should be gone")
	}
	if s.Delete("r1") {
		t.Error("Delete: second delete should return false")
	}
}
