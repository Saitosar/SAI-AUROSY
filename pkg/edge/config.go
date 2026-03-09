package edge

import (
	"os"
	"strconv"
	"strings"
)

// Config holds edge agent configuration.
type Config struct {
	EdgeID    string   // Edge node identifier
	NATSURL   string   // Local NATS URL
	CloudURL  string   // Cloud Control Plane API base URL (e.g. https://api.example.com)
	RobotIDs  []string // Robot IDs managed by this edge (for heartbeat)
	Heartbeat int      // Heartbeat interval in seconds
}

// LoadConfig loads configuration from environment.
func LoadConfig() *Config {
	robotIDs := strings.Split(os.Getenv("EDGE_ROBOT_IDS"), ",")
	trimmed := make([]string, 0, len(robotIDs))
	for _, id := range robotIDs {
		if s := strings.TrimSpace(id); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	heartbeat := 10
	if h := os.Getenv("EDGE_HEARTBEAT_SEC"); h != "" {
		if n, err := parseInt(h); err == nil && n > 0 {
			heartbeat = n
		}
	}
	return &Config{
		EdgeID:    getEnv("EDGE_ID", "edge-001"),
		NATSURL:   getEnv("NATS_URL", "nats://localhost:4222"),
		CloudURL:  getEnv("CLOUD_URL", "http://localhost:8080"),
		RobotIDs:  trimmed,
		Heartbeat: heartbeat,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}
