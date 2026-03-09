package marketplace

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SQLStore is a marketplace store backed by SQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new marketplace SQL store.
func NewSQLStore(db *sql.DB, driver string) *SQLStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLStore{db: db, driver: driver}
}

func (s *SQLStore) ph(q string) string {
	if s.driver != "pgx" {
		return q
	}
	n := 1
	var b strings.Builder
	for _, r := range q {
		if r == '?' {
			b.WriteString("$")
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ListCategories returns all categories.
func (s *SQLStore) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, slug, description FROM scenario_categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		var desc sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &desc); err != nil {
			continue
		}
		if desc.Valid {
			c.Description = desc.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListScenarios returns published scenarios with ratings.
func (s *SQLStore) ListScenarios(ctx context.Context, opts ListOptions) ([]ScenarioWithRating, error) {
	q := `
		SELECT s.id, s.name, s.description, s.steps, s.required_capabilities,
		       COALESCE(s.author,''), COALESCE(s.category_id,''), COALESCE(c.name,''), COALESCE(s.version,''), s.published_at,
		       COALESCE(avg(r.rating), 0), COUNT(r.id)
		FROM scenarios s
		LEFT JOIN scenario_categories c ON s.category_id = c.id
		LEFT JOIN scenario_ratings r ON s.id = r.scenario_id
		WHERE s.published_at IS NOT NULL
	`
	args := []interface{}{}
	if opts.Category != "" {
		q += " AND (s.category_id = ? OR c.slug = ?)"
		args = append(args, opts.Category, opts.Category)
	}
	if opts.Search != "" {
		q += " AND (s.name LIKE ? OR s.description LIKE ?)"
		pat := "%" + opts.Search + "%"
		args = append(args, pat, pat)
	}
	q += " GROUP BY s.id"
	switch opts.Sort {
	case "rating":
		q += " ORDER BY avg(r.rating) DESC, s.published_at DESC"
	case "newest":
		q += " ORDER BY s.published_at DESC"
	default:
		q += " ORDER BY s.published_at DESC"
	}

	rows, err := s.db.QueryContext(ctx, s.ph(q), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanScenarioRows(rows)
}

// GetScenario returns a scenario by ID with rating.
func (s *SQLStore) GetScenario(ctx context.Context, id string) (*ScenarioWithRating, error) {
	q := `
		SELECT s.id, s.name, s.description, s.steps, s.required_capabilities,
		       COALESCE(s.author,''), COALESCE(s.category_id,''), COALESCE(c.name,''), COALESCE(s.version,''), s.published_at,
		       COALESCE(avg(r.rating), 0), COUNT(r.id)
		FROM scenarios s
		LEFT JOIN scenario_categories c ON s.category_id = c.id
		LEFT JOIN scenario_ratings r ON s.id = r.scenario_id
		WHERE s.id = ? AND s.published_at IS NOT NULL
		GROUP BY s.id
	`
	var swr ScenarioWithRating
	var stepsJSON, capsJSON string
	var desc sql.NullString
	var author, catID, catName, version string
	var publishedAt sql.NullTime
	var avgRating float64
	var ratingCount int
	err := s.db.QueryRowContext(ctx, s.ph(q), id).
		Scan(&swr.ID, &swr.Name, &desc, &stepsJSON, &capsJSON, &author, &catID, &catName, &version, &publishedAt, &avgRating, &ratingCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		swr.Description = desc.String
	}
	_ = json.Unmarshal([]byte(stepsJSON), &swr.Steps)
	_ = json.Unmarshal([]byte(capsJSON), &swr.RequiredCapabilities)
	swr.Author = author
	swr.CategoryID = catID
	swr.CategoryName = catName
	swr.Version = version
	if publishedAt.Valid {
		swr.PublishedAt = publishedAt.Time
	}
	swr.AvgRating = avgRating
	swr.RatingCount = ratingCount
	return &swr, nil
}

// RateScenario adds or updates a rating.
func (s *SQLStore) RateScenario(ctx context.Context, scenarioID, tenantID string, rating int) error {
	if rating < 1 || rating > 5 {
		return nil // invalid, ignore
	}
	id := "rating-" + uuid.New().String()
	now := time.Now()
	_, err := s.db.ExecContext(ctx, s.ph(`
		INSERT INTO scenario_ratings (id, scenario_id, tenant_id, rating, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (scenario_id, tenant_id) DO UPDATE SET rating = excluded.rating
	`), id, scenarioID, tenantID, rating, now)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) scanScenarioRows(rows *sql.Rows) ([]ScenarioWithRating, error) {
	var out []ScenarioWithRating
	for rows.Next() {
		var swr ScenarioWithRating
		var stepsJSON, capsJSON string
		var desc sql.NullString
		var author, catID, catName, version string
		var publishedAt sql.NullTime
		var avgRating float64
		var ratingCount int
		if err := rows.Scan(&swr.ID, &swr.Name, &desc, &stepsJSON, &capsJSON, &author, &catID, &catName, &version, &publishedAt, &avgRating, &ratingCount); err != nil {
			continue
		}
		if desc.Valid {
			swr.Description = desc.String
		}
		_ = json.Unmarshal([]byte(stepsJSON), &swr.Steps)
		_ = json.Unmarshal([]byte(capsJSON), &swr.RequiredCapabilities)
		swr.Author = author
		swr.CategoryID = catID
		swr.CategoryName = catName
		swr.Version = version
		if publishedAt.Valid {
			swr.PublishedAt = publishedAt.Time
		}
		swr.AvgRating = avgRating
		swr.RatingCount = ratingCount
		out = append(out, swr)
	}
	return out, rows.Err()
}
