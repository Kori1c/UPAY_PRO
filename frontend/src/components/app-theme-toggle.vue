<script setup lang="ts">
import { computed } from 'vue'
import { IconMoonFill, IconSunFill } from '@arco-design/web-vue/es/icon'

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
        <icon-sun-fill v-if="isDark" />
        <icon-moon-fill v-else />
      </template>
    </a-button>
  </div>
</template>
