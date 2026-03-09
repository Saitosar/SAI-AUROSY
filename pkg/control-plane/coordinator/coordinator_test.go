package coordinator

import (
	"testing"
)

func TestNewCoordinator_HasDefaultZones(t *testing.T) {
	c := NewCoordinator()
	zones := c.ListZones()
	if len(zones) != 3 {
		t.Errorf("expected 3 default zones, got %d", len(zones))
	}
	ids := make(map[string]bool)
	for _, z := range zones {
		ids[z.ID] = true
	}
	for _, id := range []string{"A", "B", "C"} {
		if !ids[id] {
			t.Errorf("expected zone %s in default coordinator", id)
		}
	}
}

func TestAcquireZone_SucceedsWhenFree(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "X", Name: "Zone X"})
	c := NewCoordinatorWithStore(store)

	if !c.AcquireZone("robot-1", "X") {
		t.Error("AcquireZone should succeed when zone is free")
	}
	if c.IsZoneAvailable("X") {
		t.Error("zone should not be available after acquire")
	}
	st := c.GetZoneStatus("X")
	if st == nil || !st.Occupied || st.RobotID == nil || *st.RobotID != "robot-1" {
		t.Errorf("GetZoneStatus: expected occupied by robot-1, got %+v", st)
	}
}

func TestAcquireZone_FailsWhenOccupied(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "Y", Name: "Zone Y"})
	c := NewCoordinatorWithStore(store)

	if !c.AcquireZone("robot-1", "Y") {
		t.Fatal("first AcquireZone should succeed")
	}
	if c.AcquireZone("robot-2", "Y") {
		t.Error("second AcquireZone should fail when zone occupied")
	}
}

func TestReleaseZone_ReleasesLock(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "Z", Name: "Zone Z"})
	c := NewCoordinatorWithStore(store)

	c.AcquireZone("robot-1", "Z")
	if !c.ReleaseZone("robot-1", "Z") {
		t.Error("ReleaseZone should succeed when robot holds lock")
	}
	if !c.IsZoneAvailable("Z") {
		t.Error("zone should be available after release")
	}
}

func TestReleaseZone_FailsWhenWrongRobot(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "W", Name: "Zone W"})
	c := NewCoordinatorWithStore(store)

	c.AcquireZone("robot-1", "W")
	if c.ReleaseZone("robot-2", "W") {
		t.Error("ReleaseZone by wrong robot should fail")
	}
	if c.IsZoneAvailable("W") {
		t.Error("zone should still be occupied")
	}
}

func TestReleaseAllForRobot_ReleasesAllZones(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "P", Name: "P"})
	store.RegisterZone(&Zone{ID: "Q", Name: "Q"})
	c := NewCoordinatorWithStore(store)

	c.AcquireZone("r1", "P")
	c.AcquireZone("r1", "Q")
	c.ReleaseAllForRobot("r1")
	if !c.IsZoneAvailable("P") || !c.IsZoneAvailable("Q") {
		t.Error("ReleaseAllForRobot should release all zones held by robot")
	}
}

func TestListZoneStatuses_ReturnsAllZones(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "M", Name: "M"})
	store.RegisterZone(&Zone{ID: "N", Name: "N"})
	c := NewCoordinatorWithStore(store)

	c.AcquireZone("r1", "M")
	statuses := c.ListZoneStatuses()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 zone statuses, got %d", len(statuses))
	}
	for _, st := range statuses {
		if st.ZoneID == "M" && !st.Occupied {
			t.Error("zone M should be occupied")
		}
		if st.ZoneID == "N" && st.Occupied {
			t.Error("zone N should be available")
		}
	}
}

func TestNewCoordinatorWithStore_CustomZones(t *testing.T) {
	store := NewStore()
	store.RegisterZone(&Zone{ID: "Custom1", Name: "Custom Zone 1"})
	c := NewCoordinatorWithStore(store)

	zones := c.ListZones()
	if len(zones) != 1 || zones[0].ID != "Custom1" {
		t.Errorf("expected 1 custom zone, got %+v", zones)
	}
}
