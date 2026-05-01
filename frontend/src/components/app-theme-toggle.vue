<script setup lang="ts">
import { computed } from 'vue'

import AppIcon from './icons/app-icon.vue'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

const isDark = computed(() => appStore.resolvedTheme === 'dark')

function handleToggle() {
  appStore.toggleThemePreference()
}
</script>

<template>
  <div class="theme-toggle" :data-system="appStore.themePreference === 'system'">
    <a-button class="theme-toggle__button" shape="circle" size="large" @click="handleToggle">
      <template #icon>
        <app-icon v-if="isDark" name="sun-fill" />
        <app-icon v-else name="moon-fill" />
      </template>
    </a-button>
  </div>
</template>
