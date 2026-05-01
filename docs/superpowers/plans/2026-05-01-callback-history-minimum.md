# Callback History Minimum Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal callback history flow that records each callback attempt and lets the admin inspect per-order callback events without widening the main order table.

**Architecture:** Persist callback attempts in a dedicated `CallbackEvent` table instead of overloading the order row. Keep the order row as the summary surface (`callback_state`, `callback_message`, `last_callback_at`) and expose a focused history endpoint consumed by a lightweight orders-page modal.

**Tech Stack:** Go, Gin, Gorm, Vue 3, Vite, Arco Design Vue, Vitest

---

### Task 1: Lock callback history serialization with backend tests

**Files:**
- Modify: `web/function_test.go`
- Test: `web/function_test.go`

- [ ] Add failing tests for callback history serialization and listing.
- [ ] Verify the tests fail before production code changes.

### Task 2: Add callback event persistence model and helpers

**Files:**
- Modify: `db/sdb/sdb.go`
- Modify: `cron/cron.go`
- Create: `web/callback_events.go`

- [ ] Add `CallbackEvent` model and migrate it.
- [ ] Persist an event for each callback success/failure/manual retry trigger.
- [ ] Keep order summary fields updated alongside event writes.

### Task 3: Expose callback history API

**Files:**
- Create: `web/callback_events.go`
- Modify: `web/web.go`
- Modify: `web/function_test.go`

- [ ] Add `GET /admin/api/orders/:id/callback-events`.
- [ ] Return normalized items ordered newest first.
- [ ] Cover not-found and happy-path cases with tests.

### Task 4: Add lightweight history UI in orders page

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/views/orders/orders-view.vue`

- [ ] Add callback event types and API method.
- [ ] Add a compact “回调历史” action and modal.
- [ ] Keep existing table width stable.

### Task 5: Verify end to end

**Files:**
- Modify: none

- [ ] Run targeted Go tests for callback history.
- [ ] Run `go test ./...`.
- [ ] Run `cd frontend && npm test`.
- [ ] Run `cd frontend && npm run build`.
