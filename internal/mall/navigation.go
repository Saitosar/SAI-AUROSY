package mall

import (
	"container/heap"
	"math"
)

// Graph is an adjacency list representation of the navigation graph.
type Graph struct {
	adj map[string]map[string]float64
}

// NewGraph builds a graph from mall map nodes and edges.
// Bidirectional edges add both From→To and To→From.
func NewGraph(nodes []NavNode, edges []NavEdge) *Graph {
	adj := make(map[string]map[string]float64)
	for _, n := range nodes {
		adj[n.ID] = make(map[string]float64)
	}
	for _, e := range edges {
		if adj[e.From] == nil {
			adj[e.From] = make(map[string]float64)
		}
		adj[e.From][e.To] = e.Distance
		if e.Bidirectional {
			if adj[e.To] == nil {
				adj[e.To] = make(map[string]float64)
			}
			adj[e.To][e.From] = e.Distance
		}
	}
	return &Graph{adj: adj}
}

// ShortestPath returns the shortest path from fromID to toID using Dijkstra.
// Returns node IDs in order, total distance, and error if no path exists.
func (g *Graph) ShortestPath(fromID, toID string) ([]string, float64, error) {
	if g.adj[fromID] == nil || g.adj[toID] == nil {
		return nil, 0, ErrPathNotFound
	}
	if fromID == toID {
		return []string{fromID}, 0, nil
	}

	dist := make(map[string]float64)
	prev := make(map[string]string)
	for id := range g.adj {
		dist[id] = math.Inf(1)
	}
	dist[fromID] = 0

	pq := make(priorityQueue, 0, len(g.adj))
	heap.Push(&pq, &item{id: fromID, dist: 0})
	seen := make(map[string]bool)

	for len(pq) > 0 {
		u := heap.Pop(&pq).(*item)
		if u.id == toID {
			break
		}
		if seen[u.id] {
			continue
		}
		seen[u.id] = true

		for v, w := range g.adj[u.id] {
			if seen[v] {
				continue
			}
			alt := dist[u.id] + w
			if alt < dist[v] {
				dist[v] = alt
				prev[v] = u.id
				heap.Push(&pq, &item{id: v, dist: alt})
			}
		}
	}

	if math.IsInf(dist[toID], 1) {
		return nil, 0, ErrPathNotFound
	}

	path := make([]string, 0)
	for at := toID; at != ""; at = prev[at] {
		path = append([]string{at}, path...)
	}
	return path, dist[toID], nil
}

type item struct {
	id   string
	dist float64
}

type priorityQueue []*item

func (pq priorityQueue) Len() int            { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *priorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*item))
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}
