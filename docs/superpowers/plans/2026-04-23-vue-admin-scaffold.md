# Vue Admin Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a standalone `frontend` admin scaffold for UPay Pro using Vue 3, Vite, TypeScript, Vue Router, Pinia, and Arco Design Vue.

**Architecture:** Keep the existing Go backend unchanged and add a separate SPA in `frontend/`. The scaffold should include a reusable admin layout, route-level pages for the current backend domains, and a small shared UI foundation so later API integration is straightforward.

**Tech Stack:** Vue 3, Vite, TypeScript, Vue Router, Pinia, Arco Design Vue

---

### File Structure

- Create: `frontend/`
- Create: `frontend/src/app/`
- Create: `frontend/src/components/`
- Create: `frontend/src/layouts/`
- Create: `frontend/src/router/`
- Create: `frontend/src/stores/`
- Create: `frontend/src/styles/`
- Create: `frontend/src/types/`
- Create: `frontend/src/views/login/`
- Create: `frontend/src/views/dashboard/`
- Create: `frontend/src/views/orders/`
- Create: `frontend/src/views/wallets/`
- Create: `frontend/src/views/settings/`
- Create: `frontend/src/views/apikeys/`

### Task 1: Scaffold the project

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/src/main.ts`

- [ ] Initialize a Vue 3 + Vite + TypeScript app in `frontend/`
- [ ] Install `vue-router`, `pinia`, and `@arco-design/web-vue`
- [ ] Verify the generated app starts with the default Vite command

### Task 2: Add application shell

**Files:**
- Create: `frontend/src/App.vue`
- Create: `frontend/src/layouts/admin-layout.vue`
- Create: `frontend/src/router/index.ts`
- Create: `frontend/src/stores/app.ts`
- Create: `frontend/src/styles/global.css`

- [ ] Replace the default starter page with a routed app shell
- [ ] Add a sidebar + header admin layout using Arco components
- [ ] Add a small app store for menu collapse and page title state

### Task 3: Add starter views

**Files:**
- Create: `frontend/src/views/login/login-view.vue`
- Create: `frontend/src/views/dashboard/dashboard-view.vue`
- Create: `frontend/src/views/orders/orders-view.vue`
- Create: `frontend/src/views/wallets/wallets-view.vue`
- Create: `frontend/src/views/settings/settings-view.vue`
- Create: `frontend/src/views/apikeys/apikeys-view.vue`
- Create: `frontend/src/components/page-header-card.vue`

- [ ] Create six route pages that match the current backend domains
- [ ] Use consistent placeholder data so the scaffold already looks like an admin system
- [ ] Keep each view isolated so real API wiring can be added page by page later

### Task 4: Verify and document

**Files:**
- Modify: `frontend/README.md` or `frontend/package.json`

- [ ] Run a production build to verify the scaffold compiles
- [ ] Run a type-aware validation command if available
- [ ] Summarize the dev command and the next recommended integration step
