import { createRouter, createWebHistory } from 'vue-router'

import { routeDefinitions } from './navigation'

export const router = createRouter({
  history: createWebHistory(),
  routes: routeDefinitions,
  scrollBehavior() {
    return { top: 0 }
  },
})
