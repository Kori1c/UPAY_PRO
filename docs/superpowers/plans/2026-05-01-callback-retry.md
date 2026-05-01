# Callback Retry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add normalized callback status fields, last callback failure reason storage, and a manual callback retry action to the admin order list.

**Architecture:** Normalize callback state in backend order-list serializers so the frontend renders server-owned status. Store the latest callback failure reason on each order row. Expose a focused retry endpoint that re-triggers asynchronous callback processing for eligible paid orders.

**Tech Stack:** Go, Gin, Gorm, Vue 3, TypeScript, Arco Design Vue

---

### Task 1: Add backend callback-state tests

**Files:**
- Modify: `web/function_test.go`

- [ ] Add a failing test for callback-state normalization covering `not_applicable`, `pending`, `confirmed`, and `failed`.
- [ ] Run: `go test ./web -run TestBuildOrderListItemCallbackState -count=1`
- [ ] Confirm the test fails before implementation.

### Task 2: Implement backend callback-state serialization

**Files:**
- Create: `web/order_callbacks.go`
- Modify: `web/web.go`

- [ ] Add a focused serializer for order-list rows with:
  - `callback_state`
  - `callback_state_label`
  - `callback_message`
  - `can_retry_callback`
- [ ] Switch `/admin/api/orders` to return serialized rows instead of raw `sdb.Orders`.
- [ ] Re-run: `go test ./web -run TestBuildOrderListItemCallbackState -count=1`

### Task 3: Add callback retry endpoint tests

**Files:**
- Modify: `web/function_test.go`

- [ ] Add a failing test that verifies retry is allowed for paid orders with notify URL and denied for unpaid/confirmed/no-notify orders.
- [ ] Run: `go test ./web -run TestRetryOrderCallback -count=1`
- [ ] Confirm the test fails before implementation.

### Task 4: Implement retry endpoint and callback failure persistence

**Files:**
- Create: `web/order_callbacks.go`
- Modify: `db/sdb/sdb.go`
- Modify: `cron/cron.go`
- Modify: `web/web.go`

- [ ] Add `CallbackMessage` to the order model.
- [ ] Update callback processing so failures persist the latest error and success clears it.
- [ ] Add `POST /admin/api/orders/:id/retry-callback`.
- [ ] Re-run: `go test ./web -run TestRetryOrderCallback -count=1`

### Task 5: Update frontend order types and UI

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/views/orders/orders-view.vue`

- [ ] Add normalized callback fields to the order type.
- [ ] Replace frontend callback-state guessing with backend fields.
- [ ] Add callback message column and retry button.
- [ ] Run: `cd frontend && npm test`
- [ ] Run: `cd frontend && npm run build`

### Task 6: Final verification

**Files:**
- No code changes expected

- [ ] Run: `go test ./...`
- [ ] Run: `cd frontend && npm test`
- [ ] Run: `cd frontend && npm run build`
- [ ] Review `git diff --stat` for scope correctness
