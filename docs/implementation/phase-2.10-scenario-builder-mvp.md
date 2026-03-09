# Phase 2.10 — Scenario Builder MVP

Phase 2.10 replaces the raw JSON/comma-separated scenario editor with a structured builder: step list, capability selection, and client-side validation.

## Goal

Provide a simple, form-based scenario editor instead of manual JSON editing. Drag-and-drop and visual editor are explicitly deferred to a later phase.

## Features

### 1. Structured Step List

- **Add step** button appends a new step.
- **Each step card**:
  - **Command** dropdown: `stand_mode`, `walk_mode`, `cmd_vel`, `release_control`, `zero_mode`, `safe_stop`.
  - **Payload** (only for `cmd_vel`): three number inputs — `linear_x`, `linear_y`, `angular_z` (m/s, rad/s). Otherwise `null`.
  - **Duration (sec)** input: integer; `0` = instant, `-1` = from task payload.
- **Remove step** button per card.
- Steps rendered in order (no reorder in MVP).

### 2. Capability Multi-Select

- **Known capabilities** (hardcoded in frontend): `walk`, `stand`, `safe_stop`, `release_control`, `cmd_vel`, `zero_mode`, `patrol`, `navigation`.
- UI: checkbox chips. At least one capability required for validation.

### 3. Client-Side Validation

Before submit:

- **Required**: `id`, `name`, at least one step, at least one capability.
- **Step validation**: each step has non-empty `command`; `duration_sec` is integer; for `cmd_vel`, payload fields are numbers.
- Inline error messages.

## Files Changed

| File | Change |
|------|--------|
| `pkg/operator-console/src/App.jsx` | Refactor `ScenarioModal`: add `StepEditor`, `CapabilitySelector`, replace JSON/csv inputs, add validation |

## Data Model (Unchanged)

- **ScenarioStep**: `{ command, payload, duration_sec }`
- **Scenario**: `{ id, name, description, steps, required_capabilities }`
- API: `POST /v1/scenarios`, `PUT /v1/scenarios/:id` — same request/response format.

## Out of Scope (Deferred)

- Drag-and-drop step reordering
- Visual / flowchart editor
- API endpoint for capabilities (hardcoded in frontend for MVP)

## Related

- [Phase 2.2 Task Engine](phase-2.2-task-engine.md) — scenario catalog, task execution
- [Phase 2.9 Operator Console UX](phase-2.9-operator-console-ux.md) — read-only mode, toasts
- [Robot Adapter Contract](../adapters/robot-adapter-contract.md) — commands and capabilities
