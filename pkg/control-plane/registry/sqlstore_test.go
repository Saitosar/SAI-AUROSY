package registry

import (
	"testing"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

func TestSQLStore_SQLiteInMemory(t *testing.T) {
	s, err := NewSQLStore("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("NewSQLStore: %v", err)
	}
	defer s.Close()

	r := &hal.Robot{
		ID:              "sql-r1",
		Vendor:          "agibot",
		Model:           "X1",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
	}
	s.Add(r)

	got := s.Get("sql-r1")
	if got == nil {
		t.Fatal("Get: expected robot, got nil")
	}
	if got.ID != "sql-r1" || got.Vendor != "agibot" {
		t.Errorf("Get: got %+v", got)
	}

	list := s.List()
	if len(list) != 1 {
		t.Errorf("List: expected 1, got %d", len(list))
	}

	if !s.Delete("sql-r1") {
		t.Error("Delete: expected true")
	}
	if s.Get("sql-r1") != nil {
		t.Error("Delete: robot should be gone")
	}
}

func TestSQLStore_AddUpdate(t *testing.T) {
	s, err := NewSQLStore("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("NewSQLStore: %v", err)
	}
	defer s.Close()

	r := &hal.Robot{
		ID:              "upd-r1",
		Vendor:          "agibot",
		Model:           "X1",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	s.Add(r)
	r.Model = "X2"
	s.Add(r)

	got := s.Get("upd-r1")
	if got == nil || got.Model != "X2" {
		t.Errorf("Add update: expected Model X2, got %v", got)
	}
}
