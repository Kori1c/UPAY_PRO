# Theme And Login Refinement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a global light/dark/system theme system, unify repeated dashboard shell components, and rebuild the login page to match the provided visual design.

**Architecture:** Keep routing unchanged but insert a theme layer shared by all pages. Replace repeated layout fragments with small reusable Vue components so later API integration lands on a consistent visual system instead of one-off page markup.

**Tech Stack:** Vue 3, Vite, TypeScript, Pinia, Vue Router, Arco Design Vue, Vitest

---

### File Structure

- Create: `frontend/src/app/theme.ts`
- Create: `frontend/src/components/app-theme-toggle.vue`
- Create: `frontend/src/components/page-section-card.vue`
- Create: `frontend/src/components/metric-stat-card.vue`
- Create: `frontend/src/components/info-list-card.vue`
- Create: `frontend/src/components/form-section-card.vue`
- Create: `frontend/src/app/theme.spec.ts`
- Modify: `frontend/src/stores/app.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/layouts/admin-layout.vue`
- Modify: `frontend/src/styles/global.css`
- Modify: `frontend/src/views/login/login-view.vue`
- Modify: `frontend/src/views/dashboard/dashboard-view.vue`
- Modify: `frontend/src/views/wallets/wallets-view.vue`
- Modify: `frontend/src/views/settings/settings-view.vue`
- Modify: `frontend/src/views/apikeys/apikeys-view.vue`

### Task 1: Add theme utilities and tests

**Files:**
- Create: `frontend/src/app/theme.ts`
- Test: `frontend/src/app/theme.spec.ts`

- [ ] Write a failing test for theme preference parsing and resolved theme selection
- [ ] Run the test to confirm it fails because the helper does not exist yet
- [ ] Implement the smallest helper set needed for `system`, `light`, and `dark`
- [ ] Re-run the theme test and confirm it passes

### Task 2: Wire theme into the app shell

**Files:**
- Modify: `frontend/src/stores/app.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/layouts/admin-layout.vue`
- Create: `frontend/src/components/app-theme-toggle.vue`

- [ ] Extend the app store with theme preference state and persistence
- [ ] Sync the resolved theme to DOM attributes for app CSS and Arco dark mode
- [ ] Add a visible theme switcher in the admin header

### Task 3: Unify repeated page components

**Files:**
- Create: `frontend/src/components/page-section-card.vue`
- Create: `frontend/src/components/metric-stat-card.vue`
- Create: `frontend/src/components/info-list-card.vue`
- Create: `frontend/src/components/form-section-card.vue`
- Modify: `frontend/src/views/dashboard/dashboard-view.vue`
- Modify: `frontend/src/views/wallets/wallets-view.vue`
- Modify: `frontend/src/views/settings/settings-view.vue`
- Modify: `frontend/src/views/apikeys/apikeys-view.vue`

- [ ] Replace repeated `glass-panel` usage with explicit shared components
- [ ] Keep each component focused on one repeated page pattern
- [ ] Preserve current content but make styling and spacing consistent

### Task 4: Rebuild the login page from the provided design

**Files:**
- Modify: `frontend/src/views/login/login-view.vue`
- Modify: `frontend/src/styles/global.css`

- [ ] Recreate the two-column composition and card hierarchy from the reference
- [ ] Build a reusable CSS illustration rather than a disposable one-off bitmap dependency
- [ ] Keep the page responsive and visually coherent in both light and dark themes

### Task 5: Verify

**Files:**
- Modify: `frontend/package.json` only if scripts need adjustment

- [ ] Run the focused theme test
- [ ] Run the existing route test
- [ ] Run the production build
- [ ] Summarize any remaining visual polish or integration follow-ups
