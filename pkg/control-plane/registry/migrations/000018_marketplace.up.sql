-- Scenario categories for marketplace
CREATE TABLE IF NOT EXISTS scenario_categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT
);

-- Seed default categories
INSERT INTO scenario_categories (id, name, slug, description) VALUES
('mobility', 'Mobility', 'mobility', 'Walking, navigation, patrol'),
('safety', 'Safety', 'safety', 'Safe stop, standby, release control'),
('inspection', 'Inspection', 'inspection', 'Surveillance, monitoring')
ON CONFLICT (id) DO NOTHING;

-- Extend scenarios for marketplace
ALTER TABLE scenarios ADD COLUMN author TEXT DEFAULT 'platform';
ALTER TABLE scenarios ADD COLUMN category_id TEXT;
ALTER TABLE scenarios ADD COLUMN version TEXT DEFAULT '1.0';
ALTER TABLE scenarios ADD COLUMN published_at TIMESTAMP;

-- Mark built-in scenarios as published
UPDATE scenarios SET published_at = COALESCE(created_at, CURRENT_TIMESTAMP), author = 'platform', version = '1.0', category_id = 'mobility' WHERE id IN ('patrol', 'navigation');
UPDATE scenarios SET published_at = COALESCE(created_at, CURRENT_TIMESTAMP), author = 'platform', version = '1.0', category_id = 'safety' WHERE id = 'standby';

-- Scenario ratings (1-5 stars)
CREATE TABLE IF NOT EXISTS scenario_ratings (
    id TEXT PRIMARY KEY,
    scenario_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMP NOT NULL,
    UNIQUE(scenario_id, tenant_id)
);
