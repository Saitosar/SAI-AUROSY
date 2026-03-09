package coordinator

import (
	"sync"
	"time"
)

// Store manages zone locks in memory.
type Store struct {
	mu    sync.RWMutex
	zones map[string]*Zone
	locks map[string]*ZoneLock // zone_id -> lock
}

// NewStore creates a new coordinator store.
func NewStore() *Store {
	return &Store{
		zones: make(map[string]*Zone),
		locks: make(map[string]*ZoneLock),
	}
}

// RegisterZone adds or updates a zone.
func (s *Store) RegisterZone(z *Zone) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zones[z.ID] = z
}

// ListZones returns all registered zones.
func (s *Store) ListZones() []Zone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Zone, 0, len(s.zones))
	for _, z := range s.zones {
		out = append(out, *z)
	}
	return out
}

// AcquireZone attempts to acquire exclusive access to a zone for a robot.
// Returns true if acquired, false if zone is already occupied.
func (s *Store) AcquireZone(robotID, zoneID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, occupied := s.locks[zoneID]; occupied {
		return false
	}
	s.locks[zoneID] = &ZoneLock{
		ZoneID:   zoneID,
		RobotID:  robotID,
		Acquired: time.Now(),
	}
	return true
}

// ReleaseZone releases a zone lock held by a robot.
// Returns true if the lock was held and released.
func (s *Store) ReleaseZone(robotID, zoneID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, ok := s.locks[zoneID]
	if !ok || lock.RobotID != robotID {
		return false
	}
	delete(s.locks, zoneID)
	return true
}

// ReleaseAllForRobot releases all zone locks held by a robot.
func (s *Store) ReleaseAllForRobot(robotID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for zoneID, lock := range s.locks {
		if lock.RobotID == robotID {
			delete(s.locks, zoneID)
		}
	}
}

// IsZoneAvailable returns true if the zone is not occupied.
func (s *Store) IsZoneAvailable(zoneID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, occupied := s.locks[zoneID]
	return !occupied
}

// GetZoneStatus returns the status of a zone.
func (s *Store) GetZoneStatus(zoneID string) *ZoneStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lock, occupied := s.locks[zoneID]
	st := &ZoneStatus{ZoneID: zoneID, Occupied: occupied}
	if occupied {
		st.RobotID = &lock.RobotID
	}
	return st
}

// ListZoneStatuses returns status for all registered zones.
func (s *Store) ListZoneStatuses() []ZoneStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ZoneStatus
	for zoneID := range s.zones {
		lock, occupied := s.locks[zoneID]
		st := ZoneStatus{ZoneID: zoneID, Occupied: occupied}
		if occupied {
			st.RobotID = &lock.RobotID
		}
		out = append(out, st)
	}
	return out
}
