package mall

import (
	"context"
	"fmt"
	"strings"
)

// Service provides mall map and route operations.
type Service struct {
	repo    Repository
	planner *RoutePlanner
}

// NewService creates a mall service.
func NewService(repo Repository) *Service {
	return &Service{
		repo:    repo,
		planner: NewRoutePlanner(repo),
	}
}

// GetMallMap returns the full mall map.
func (s *Service) GetMallMap(ctx context.Context, mallID string) (*MallMap, error) {
	return s.repo.GetMap(ctx, mallID)
}

// FindStoreNode looks up a store by name and returns its NavNode.
// Uses case-insensitive fuzzy matching on StoreLocation.StoreName.
func (s *Service) FindStoreNode(ctx context.Context, mallID, storeName string) (*NavNode, error) {
	m, err := s.repo.GetMap(ctx, mallID)
	if err != nil {
		return nil, err
	}
	key := strings.ToLower(strings.TrimSpace(storeName))
	nodeByID := make(map[string]NavNode)
	for _, n := range m.Nodes {
		nodeByID[n.ID] = n
	}
	for _, sl := range m.Stores {
		if strings.EqualFold(sl.StoreName, storeName) {
			if n, ok := nodeByID[sl.NodeID]; ok {
				return &n, nil
			}
			return nil, ErrStoreNotFound
		}
	}
	for _, sl := range m.Stores {
		if strings.Contains(strings.ToLower(sl.StoreName), key) || strings.Contains(key, strings.ToLower(sl.StoreName)) {
			if n, ok := nodeByID[sl.NodeID]; ok {
				return &n, nil
			}
		}
	}
	return nil, ErrStoreNotFound
}

// GetBasePoint returns the standby/base node for the mall.
func (s *Service) GetBasePoint(ctx context.Context, mallID string) (*NavNode, error) {
	m, err := s.repo.GetMap(ctx, mallID)
	if err != nil {
		return nil, err
	}
	if m.BasePoint == "" {
		return nil, fmt.Errorf("mall %s has no base point", mallID)
	}
	for i := range m.Nodes {
		if m.Nodes[i].ID == m.BasePoint {
			return &m.Nodes[i], nil
		}
	}
	return nil, ErrPathNotFound
}

// CalculateRoute returns the shortest path between two nodes.
func (s *Service) CalculateRoute(ctx context.Context, mallID, fromNodeID, toNodeID string) ([]NavNode, float64, error) {
	return s.planner.CalculateRoute(ctx, mallID, fromNodeID, toNodeID)
}

// ListStores returns all store locations in the mall.
func (s *Service) ListStores(ctx context.Context, mallID string) ([]StoreLocation, error) {
	m, err := s.repo.GetMap(ctx, mallID)
	if err != nil {
		return nil, err
	}
	return m.Stores, nil
}
