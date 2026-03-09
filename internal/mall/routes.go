package mall

import (
	"context"
)

// RoutePlanner calculates routes between nodes in a mall.
type RoutePlanner struct {
	repo Repository
}

// NewRoutePlanner creates a route planner.
func NewRoutePlanner(repo Repository) *RoutePlanner {
	return &RoutePlanner{repo: repo}
}

// CalculateRoute returns the shortest path from fromNodeID to toNodeID as ordered NavNodes.
func (p *RoutePlanner) CalculateRoute(ctx context.Context, mallID, fromNodeID, toNodeID string) ([]NavNode, float64, error) {
	m, err := p.repo.GetMap(ctx, mallID)
	if err != nil {
		return nil, 0, err
	}
	nodeByID := make(map[string]NavNode)
	for _, n := range m.Nodes {
		nodeByID[n.ID] = n
	}
	if _, ok := nodeByID[fromNodeID]; !ok {
		return nil, 0, ErrPathNotFound
	}
	if _, ok := nodeByID[toNodeID]; !ok {
		return nil, 0, ErrPathNotFound
	}
	g := NewGraph(m.Nodes, m.Edges)
	pathIDs, dist, err := g.ShortestPath(fromNodeID, toNodeID)
	if err != nil {
		return nil, 0, err
	}
	route := make([]NavNode, len(pathIDs))
	for i, id := range pathIDs {
		route[i] = nodeByID[id]
	}
	return route, dist, nil
}
