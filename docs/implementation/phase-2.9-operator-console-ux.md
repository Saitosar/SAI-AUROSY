# Phase 2.9 — Operator Console UX Improvements

Phase 2.9 adds three UX improvements to the Operator Console: read-only mode for monitoring, command history in robot cards, and toast notifications for key events.

## Features

### 1. Read-Only Mode (Viewer Role)

A view-only mode for operators who need to monitor the fleet without sending commands.

- **Backend:** New role `viewer` in RBAC. GET endpoints allow `operator`, `administrator`, or `viewer`. POST/PUT/DELETE require `operator` or `administrator`.
- **Endpoint:** `GET /v1/me` returns `{ roles: string[], tenant_id?: string }` for the current user.
- **Frontend:** Operator Console fetches `/me` on load. When the user has only the `viewer` role (no `operator` or `administrator`), all write actions are hidden: command buttons, Safe Stop, Create Task, Cancel, Run Workflow, Create/Edit/Delete Scenario, Use Scenario.
- **Indicator:** A "Только чтение" badge appears in the header when in read-only mode.

### 2. Command History in Robot Card

Last commands per robot from the audit log, shown in a collapsible section.

- **Backend:** Uses existing `GET /v1/audit?robot_id={id}&action=command&limit=10`.
- **Frontend:** Each robot card has an "История команд" section. When expanded, fetches audit entries and displays: `actor: command — relative time` (e.g. "console: Safe Stop — 2 мин назад").

### 3. Toast Notifications

Toast alerts for `safe_stop`, `robot_online`, and `task_completed`.

- **Backend:** Event broadcaster (`pkg/control-plane/events/broadcaster.go`) distributes events to SSE clients. Wired to task runner (OnTaskCompleted), robot online watcher, and sendCommand (safe_stop).
- **Endpoint:** `GET /v1/events/stream` — SSE stream. Events: `robot_online`, `task_completed`, `safe_stop`. Format: `event: <type>`, `data: { event, timestamp, data }`.
- **Frontend:** Operator Console subscribes via EventSource. Toasts appear top-right, auto-dismiss after 5 seconds. Colors: safe_stop (red), robot_online (green), task_completed (blue).

## Files Changed

| File | Change |
|------|--------|
| `pkg/control-plane/auth/rbac.go` | Add `RoleViewer` |
| `pkg/control-plane/api/server.go` | Add `/v1/me`, `/v1/events/stream`, split routes, event broadcaster |
| `pkg/control-plane/events/broadcaster.go` | **New** — SSE event broadcaster |
| `cmd/control-plane/main.go` | Wire event broadcaster |
| `pkg/operator-console/src/App.jsx` | readOnly mode, command history, Toast, EventSource |
| `docs/integration/api-reference.md` | Document `/me`, `/events/stream`, roles |

## API Keys with Viewer Role

Create an API key with role `viewer` for read-only access:

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <admin-key>" \
  -d '{"name":"monitor","roles":"viewer","tenant_id":"default"}'
```

Use the returned key in the Operator Console (e.g. `VITE_API_KEY=sk-...`) to get read-only mode.
