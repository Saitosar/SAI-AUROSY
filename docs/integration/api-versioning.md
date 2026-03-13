# API Versioning and Deprecation Policy

This document defines how the SAI AUROSY Control Plane API is versioned and how deprecated versions are retired.

## Versioning Scheme

- **URL path versioning:** All API endpoints are under a version prefix: `/v1`, `/v2`, etc. (When behind a proxy that adds `/api`, the full path is `/api/v1`.)
- **No header-based versioning:** Clients select a version by URL path only.
- **Stability guarantee:** A version remains stable until explicitly deprecated. Non-breaking changes stay within the same version.

## When to Introduce a New Version (/v2)

A new major version (e.g. `/v2`) is introduced **only for breaking changes**. Examples:

| Breaking change | Example |
|-----------------|---------|
| Removing or renaming request/response fields | `robot_id` renamed to `robotId` |
| Changing HTTP status codes or error formats | 404 response body schema changed |
| Removing endpoints | `GET /zones` removed |
| Changing authentication/authorization semantics | API key format changed, new required scopes |
| Changing pagination or filtering behavior | `limit` default or max changed |

## Non-Breaking Changes (Stay in v1)

The following do **not** require a new version:

- Adding new optional request fields
- Adding new optional response fields
- Adding new endpoints
- Adding new query parameters (optional)
- Adding new event types to webhooks
- Bug fixes that do not change documented behavior

## Deprecation Lifecycle

When a version is superseded by a newer one, it follows this lifecycle:

| Phase | Duration | Actions |
|-------|----------|---------|
| **Announcement** | At least 6 months before sunset | Changelog, docs update, `Deprecation` and `Sunset` headers added to responses |
| **Deprecated** | Minimum 6 months | Old version still works; response headers warn clients |
| **Sunset** | After grace period | Old version removed; requests return 410 Gone or redirect to migration guide |

### Response Headers During Deprecation

When a version is deprecated, all responses include:

- `Deprecation: true` — Indicates the API version is deprecated
- `Sunset: <RFC 3339 date>` — Date when the version will be removed (e.g. `Sunset: 2026-12-31T00:00:00Z`)

Clients should monitor these headers and plan migration before the sunset date.

### Migration Guide

When v2 is introduced, a migration guide will be published at `docs/integration/v1-to-v2-migration.md` (or equivalent), covering:

- Summary of breaking changes
- Endpoint mapping (v1 → v2)
- Request/response schema changes
- Authentication changes, if any

### Communication

- **Changelog and release notes:** Deprecation announced in the release that introduces the replacement version
- **Documentation:** Deprecation notice and sunset date in API Reference and Integration Guide
- **API key holders:** Email or in-app notification to known integrations when applicable

## Current Versions

| Version | Status | Notes |
|---------|--------|-------|
| v1 | **Current** | Stable. No deprecation planned. |

## Implementation Notes

When v2 is introduced:

1. Add `v2 := r.PathPrefix("/v2").Subrouter()` in the API server
2. Register v2 handlers with new implementations
3. Wrap v1 subrouter with deprecation middleware that sets `Deprecation: true` and `Sunset: <date>`
4. Provide versioned OpenAPI specs (`/openapi/v1.json`, `/openapi/v2.json`)
