package registry

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs pending migrations. driver is "sqlite" or "postgres" for placeholder format.
func Migrate(db *sql.DB, driver string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var ups []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		ups = append(ups, strings.TrimSuffix(e.Name(), ".up.sql"))
	}
	sort.Strings(ups)
	for _, name := range ups {
		if applied[name] {
			continue
		}
		sql, err := migrationsFS.ReadFile("migrations/" + name + ".up.sql")
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		insertSQL := "INSERT INTO schema_migrations (version) VALUES (?)"
		if driver == "postgres" {
			insertSQL = "INSERT INTO schema_migrations (version) VALUES ($1)"
		}
		if _, err := db.Exec(insertSQL, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY
		)
	`)
	return err
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		m[v] = true
	}
	return m, rows.Err()
}
