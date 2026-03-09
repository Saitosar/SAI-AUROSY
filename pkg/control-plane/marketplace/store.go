package marketplace

import (
	"context"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
)

// Category represents a scenario category.
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// ScenarioWithRating extends Scenario with rating info.
type ScenarioWithRating struct {
	scenarios.Scenario
	Author       string    `json:"author,omitempty"`
	CategoryID   string    `json:"category_id,omitempty"`
	CategoryName string    `json:"category_name,omitempty"`
	Version      string    `json:"version,omitempty"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
	AvgRating    float64   `json:"avg_rating"`
	RatingCount  int       `json:"rating_count"`
}

// ListOptions filters marketplace scenarios.
type ListOptions struct {
	Category string // filter by category slug or id
	Search   string // search in name, description
	Sort     string // "rating", "newest" (default)
}

// Store is the marketplace store interface.
type Store interface {
	ListCategories(ctx context.Context) ([]Category, error)
	ListScenarios(ctx context.Context, opts ListOptions) ([]ScenarioWithRating, error)
	GetScenario(ctx context.Context, id string) (*ScenarioWithRating, error)
	RateScenario(ctx context.Context, scenarioID, tenantID string, rating int) error
}
