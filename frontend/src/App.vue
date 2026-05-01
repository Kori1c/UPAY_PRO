<script setup lang="ts">
import { onBeforeUnmount, onMounted, watch } from 'vue'

import {
  applyThemeToDocument,
  resolveSystemTheme,
  resolveThemeValue,
} from './app/theme'
import { useAppStore } from './stores/app'

const appStore = useAppStore()
let mediaQuery: MediaQueryList | null = null

function syncTheme() {
  const systemTheme = resolveSystemTheme(mediaQuery?.matches)
  const resolvedTheme = resolveThemeValue(appStore.themePreference, systemTheme)
  appStore.setResolvedTheme(resolvedTheme)
  applyThemeToDocument(resolvedTheme)
}

function handleThemeChange() {
  syncTheme()
}

watch(
  () => appStore.themePreference,
  () => {
    syncTheme()
  },
)

onMounted(() => {
  appStore.hydrateThemePreference()
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', handleThemeChange)
  syncTheme()
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', handleThemeChange)
})
</script>

<template>
  <router-view v-slot="{ Component }">
    <transition name="fade" mode="out-in">
      <component :is="Component" />
    </transition>
  </router-view>
</template>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
