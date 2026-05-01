# Production Readiness Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the current UPay Pro branch from “internal demo / active secondary development” to a cleaner, more consistent, and easier-to-ship state.

**Architecture:** Keep the existing Go backend + Vue 3 admin split, but run a focused cleanup pass over the product surface and release assets. Prioritize consistency, correctness, and operational clarity before tackling deeper architectural refactors.

**Tech Stack:** Go, Gin, SQLite, Redis, Vue 3, Vite, TypeScript, Arco Design, Docker Compose

---

### File Structure

- Modify: `frontend/src/views/dashboard/dashboard-view.vue`
- Modify: `frontend/src/views/orders/orders-view.vue`
- Modify: `frontend/src/views/login/login-view.vue`
- Modify: `frontend/src/views/settings/settings-view.vue`
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/styles/global.css`
- Modify: `static/pay.html`
- Modify: `README.md`
- Modify: `DEEP_AUDIT_ISSUES.md`
- Modify: `AUDIT_AND_FIX_REPORT.md`
- Modify: `web/web.go`
- Modify: `web/function.go`
- Modify: `db/rdb/rdb.go`
- Modify: `cron/cron.go`

### Task 1: Align visible product copy and amount/unit wording

**Files:**
- Modify: `frontend/src/views/dashboard/dashboard-view.vue`
- Modify: `frontend/src/views/orders/orders-view.vue`
- Modify: `frontend/src/views/login/login-view.vue`
- Modify: `frontend/src/settings/settings-view.vue`
- Modify: `static/pay.html`
- Modify: `README.md`

- [ ] Audit the current UI for mixed `¥` / `USD` / `USDT` wording and list the screens that still disagree
- [ ] Replace outdated or misleading labels with a single approved wording set
- [ ] Rebuild the frontend and verify the admin output updates cleanly
- [ ] Rebuild Docker if any server-rendered page copy changes

### Task 2: Reconcile docs with the current product direction

**Files:**
- Modify: `README.md`
- Modify: `DEEP_AUDIT_ISSUES.md`
- Modify: `AUDIT_AND_FIX_REPORT.md`

- [ ] Remove or update stale statements that no longer match the current product
- [ ] Separate “already fixed”, “partially mitigated”, and “still open” issues
- [ ] Add the current local Docker workflow and current default admin credentials where appropriate
- [ ] Re-read the edited docs and remove contradictions

### Task 3: Run an end-to-end regression pass on the critical payment flow

**Files:**
- Modify only if a bug is found during verification

- [ ] Verify login, dashboard load, order list load, wallet add/edit, payment page open, order status refresh, expired order handling, and settings save
- [ ] Record any failing step with the exact page, action, and observed result
- [ ] Fix only confirmed regressions discovered during the pass
- [ ] Re-run the affected validation commands

### Task 4: Triage remaining production blockers

**Files:**
- Modify: `web/web.go`
- Modify: `web/function.go`
- Modify: `db/rdb/rdb.go`
- Modify: `cron/cron.go`

- [ ] Re-check the previously identified high-risk backend issues against the current code
- [ ] Mark each one as fixed, partially fixed, or still open with evidence
- [ ] Pick the single highest-risk unresolved backend item and implement the smallest safe improvement
- [ ] Verify the touched backend path with focused tests

### Task 5: Final verification and release summary

**Files:**
- Modify docs only if verification reveals missing notes

- [ ] Run `npm run build` in `frontend/`
- [ ] Run `go test ./web`
- [ ] Rebuild `docker compose up -d --build app`
- [ ] Summarize current production readiness with explicit “ready”, “needs monitoring”, and “not ready” items
