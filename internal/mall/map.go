package mall

import "fmt"

// MallMap represents a mall digital twin with floors, navigation graph, and store locations.
type MallMap struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Floors     []Floor          `json:"floors"`
	Nodes      []NavNode        `json:"nodes"`
	Edges      []NavEdge        `json:"edges"`
	Stores     []StoreLocation  `json:"stores"`
	BasePoint  string           `json:"base_point"`
}

// Floor represents a floor level in the mall.
type Floor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// NavNode represents a key point in the mall navigation graph.
// Type: store, entrance, standby, elevator, escalator, info_desk, junction
type NavNode struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	FloorID     string       `json:"floor_id"`
	Zone        string       `json:"zone"`
	Coordinates Coordinates  `json:"coordinates"`
	Type        string       `json:"type"`
}

// NavEdge represents a walkable route between two nodes.
type NavEdge struct {
	From          string  `json:"from"`
	To            string  `json:"to"`
	Distance      float64 `json:"distance"`
	Bidirectional bool    `json:"bidirectional"`
}

// Coordinates holds X, Y (and optionally Z) position.
type Coordinates struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// String returns coordinates as "x,y,0" for adapter compatibility.
func (c Coordinates) String() string {
	return fmt.Sprintf("%.2f,%.2f,0", c.X, c.Y)
}

// StoreLocation maps a store to its navigation node.
type StoreLocation struct {
	StoreName string `json:"store_name"`
	FloorID   string `json:"floor_id"`
	Zone      string `json:"zone"`
	NodeID    string `json:"node_id"`
}
