# Phase 3.4 — Robot Application Marketplace

Phase 3.4 implements the Robot Application Marketplace: catalog of published scenarios with categories, ratings, and discovery.

## Overview

| Area | Changes |
|------|---------|
| **Schema** | `scenario_categories`, `scenario_ratings`, extended `scenarios` (author, category_id, version, published_at) |
| **API** | `/v1/marketplace/categories`, `/v1/marketplace/scenarios`, `POST .../rate` |
| **Console** | Marketplace section: catalog, filters, ratings, "Use" button |

## Schema

### scenario_categories

| Column | Type |
|--------|------|
| id | TEXT PK |
| name | TEXT |
| slug | TEXT UNIQUE |
| description | TEXT |

Seed: mobility, safety, inspection.

### scenario_ratings

| Column | Type |
|--------|------|
| id | TEXT PK |
| scenario_id | TEXT |
| tenant_id | TEXT |
| rating | INTEGER 1-5 |
| created_at | TIMESTAMP |
| UNIQUE(scenario_id, tenant_id) | |

One rating per tenant per scenario.

### scenarios (extended)

- `author` — e.g. "platform"
- `category_id` — FK to scenario_categories
- `version` — e.g. "1.0"
- `published_at` — when set, scenario appears in marketplace

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/marketplace/categories` | List categories |
| GET | `/v1/marketplace/scenarios` | List published scenarios; query: `category`, `search`, `sort=rating|newest` |
| GET | `/v1/marketplace/scenarios/{id}` | Get scenario with avg rating, rating count |
| POST | `/v1/marketplace/scenarios/{id}/rate` | Submit rating (1-5); body: `{"rating": 1-5}` |

## Publish Flow

Admin creates/updates scenario via `/v1/scenarios`. To publish, set `published_at` (e.g. via migration or future publish endpoint). Built-in scenarios (standby, patrol, navigation) are marked published in migration `000018_marketplace.up.sql`.

## Operator Console

New section **Каталог приложений** (Marketplace):

- Grid of cards: name, description, category, author, rating (stars), "Использовать" (Use)
- Filters: category, search
- Sort: newest, rating
- Click "Use" → opens Create Task modal with scenario pre-selected
- Click stars to rate (1-5)

## Links

- [Phase 3.3 Developer Platform](phase-3.3-developer-platform.md)
- [Phase 2.2 Task Engine](phase-2.2-task-engine.md)
- [Roadmap](../product/roadmap.md)
