# Adapter Validation Checklist

Use this checklist when onboarding a real robot adapter to the SAI AUROSY platform. Complete all items before considering the adapter pilot-ready.

**Automated validation:** Run `go run ./cmd/validation` to validate scenarios against the Simulated Robot Harness (requires NATS). Use `-contract-check` to verify adapter contract compliance (telemetry fields, robot_id consistency) after each scenario. Use `-output-contract` to generate the contract JSON for tooling or vendor onboarding.

---

## 1. Identity and Registration

- [ ] Robot is registered in Fleet Registry with correct `robot_id` (vendor prefix: `x1-`, `go2-`, `ros-`, etc.)
- [ ] `robot_id` is consistent in telemetry, command subscription, and registry
- [ ] Tenant ID and optional Edge ID are set correctly

---

## 2. Command Support Verified

- [ ] `navigate_to` — Executes and updates `target_position`, `distance_to_target` in telemetry
- [ ] `safe_stop` — Stops movement immediately; `current_task` reflects idle
- [ ] `release_control` — Operator can take over; platform automation does not override
- [ ] `walk_mode` — Robot enters walking mode
- [ ] `stand_mode` — Robot enters standing pose
- [ ] `speak` (if speech capability) — TTS audio plays or text is sent correctly

---

## 3. Telemetry Mapping Verified

- [ ] `robot_id` — Present and correct
- [ ] `timestamp` — ISO8601, UTC
- [ ] `online` — Accurately reflects connection state
- [ ] `position` — Format `"x,y,z"` when available
- [ ] `target_position` — Set when navigating
- [ ] `distance_to_target` — Updates during navigation; decreases as robot approaches
- [ ] `current_task` — Reflects mode: `idle`, `stand`, `walk`
- [ ] `actuator_status` — One of: `enabled`, `disabled`, `error`, `calibration`

---

## 4. Capability Mapping Verified

- [ ] Declared capabilities match actual robot behavior
- [ ] `navigation` → `navigate_to` works and arrival is detectable
- [ ] `speech` → `speak` or audio output works
- [ ] `safe_stop` → Emergency stop works
- [ ] `release_control` → Manual override path exists

---

## 5. Safety Behavior Verified

- [ ] `safe_stop` is handled with highest priority
- [ ] `safe_stop` is never silently ignored
- [ ] Operator control overrides automation when `release_control` is used
- [ ] Safety commands override non-safety commands

---

## 6. Execution Engine Compatibility Verified

- [ ] Navigation arrival is detected when `distance_to_target` < 1.0 m
- [ ] Robot offline (`online: false`) causes navigation task to fail cleanly
- [ ] Navigation timeout (no arrival within 60–120 s) triggers failure path
- [ ] Return-to-base flow completes and robot ends in IDLE state

---

## 7. Mall Assistant Scenario Compatibility Verified

- [ ] Full flow: start scenario → visitor request → navigate → arrive → speak → return → IDLE
- [ ] Unknown store: no navigation task created; graceful completion
- [ ] Events: `visitor_interaction_started`, `mall_store_resolved`, `navigation_started`, `navigation_completed`, `visitor_interaction_finished`
- [ ] Return-to-base: `robot_returning_to_base`, `robot_idle` events

---

## 8. Failure Scenarios Verified

- [ ] Robot offline before task → task fails cleanly
- [ ] Robot offline mid-route → execution engine transitions to error path
- [ ] Safe stop mid-route → movement stops; lifecycle recorded
- [ ] Navigation timeout → failure path triggered

---

## 9. Timing and Performance

- [ ] Telemetry published at least every 2 seconds when active
- [ ] Command effect visible in telemetry within 2 seconds
- [ ] No stale telemetry gaps > 5 seconds when robot is online

---

## Sign-off

| Role | Name | Date |
|------|------|------|
| Adapter Developer | | |
| Platform QA | | |
| Pilot Lead | | |

---

## Related Documents

- [Adapter Readiness Contract](adapter-readiness-contract.md)
- [Validation Layer](../../architecture/validation-layer.md)
