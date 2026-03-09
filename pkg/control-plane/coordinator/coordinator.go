package coordinator

// Coordinator provides zone-based coordination for multi-robot tasks.
// Uses FIFO: first robot to acquire a zone gets it.
type Coordinator struct {
	store *Store
}

// NewCoordinator creates a coordinator with default zones A, B, C.
func NewCoordinator() *Coordinator {
	c := &Coordinator{store: NewStore()}
	c.store.RegisterZone(&Zone{ID: "A", Name: "Zone A"})
	c.store.RegisterZone(&Zone{ID: "B", Name: "Zone B"})
	c.store.RegisterZone(&Zone{ID: "C", Name: "Zone C"})
	return c
}

// NewCoordinatorWithStore creates a coordinator with a custom store (e.g. for testing).
func NewCoordinatorWithStore(store *Store) *Coordinator {
	return &Coordinator{store: store}
}

// AcquireZone attempts to acquire exclusive access to a zone.
func (c *Coordinator) AcquireZone(robotID, zoneID string) bool {
	return c.store.AcquireZone(robotID, zoneID)
}

// ReleaseZone releases a zone lock.
func (c *Coordinator) ReleaseZone(robotID, zoneID string) bool {
	return c.store.ReleaseZone(robotID, zoneID)
}

// ReleaseAllForRobot releases all zone locks held by a robot.
func (c *Coordinator) ReleaseAllForRobot(robotID string) {
	c.store.ReleaseAllForRobot(robotID)
}

// IsZoneAvailable returns true if the zone is not occupied.
func (c *Coordinator) IsZoneAvailable(zoneID string) bool {
	return c.store.IsZoneAvailable(zoneID)
}

// GetZoneStatus returns the status of a zone.
func (c *Coordinator) GetZoneStatus(zoneID string) *ZoneStatus {
	return c.store.GetZoneStatus(zoneID)
}

// ListZones returns all registered zones.
func (c *Coordinator) ListZones() []Zone {
	return c.store.ListZones()
}

// ListZoneStatuses returns status for all zones.
func (c *Coordinator) ListZoneStatuses() []ZoneStatus {
	return c.store.ListZoneStatuses()
}
