<script setup lang="ts">
import { computed } from 'vue'

type IconName =
  | 'apps'
  | 'copy'
  | 'dashboard'
  | 'info-circle'
  | 'launch'
  | 'lock'
  | 'moon-fill'
  | 'poweroff'
  | 'question-circle'
  | 'right'
  | 'safe'
  | 'save'
  | 'storage'
  | 'sun-fill'
  | 'user'

const props = withDefaults(
  defineProps<{
    name: IconName
  }>(),
  {},
)

const iconMap: Record<IconName, string> = {
  apps: '<rect x="5" y="5" width="6.5" height="6.5" rx="1.5" /><rect x="12.5" y="5" width="6.5" height="6.5" rx="1.5" /><rect x="5" y="12.5" width="6.5" height="6.5" rx="1.5" /><rect x="12.5" y="12.5" width="6.5" height="6.5" rx="1.5" />',
  copy: '<rect x="9" y="7" width="9" height="11" rx="2" /><path d="M7 15H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v1" />',
  dashboard: '<path d="M4 13a8 8 0 1 1 16 0" /><path d="M12 13 16.5 8.5" /><circle cx="12" cy="13" r="1.2" fill="currentColor" stroke="none" />',
  'info-circle': '<circle cx="12" cy="12" r="9" /><path d="M12 10v5" /><circle cx="12" cy="7.5" r="1" fill="currentColor" stroke="none" />',
  launch: '<path d="M13 5h6v6" /><path d="M19 5 11 13" /><path d="M10 7H7a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-3" />',
  lock: '<rect x="5" y="10" width="14" height="10" rx="2.5" /><path d="M8 10V7.8A4 4 0 0 1 12 4a4 4 0 0 1 4 3.8V10" /><path d="M12 14v2.5" />',
  'moon-fill': '<path d="M14.5 3.5a8.5 8.5 0 1 0 6 14.5 8.2 8.2 0 0 1-2.8.5A8.5 8.5 0 0 1 9.2 10c0-2.4 1-4.5 2.7-6a8.4 8.4 0 0 0 2.6-.5Z" fill="currentColor" stroke="none" />',
  poweroff: '<path d="M12 3v8" /><path d="M7.1 5.6A7 7 0 1 0 16.9 5.6" />',
  'question-circle': '<circle cx="12" cy="12" r="9" /><path d="M9.8 9.5a2.6 2.6 0 1 1 4.7 1.5c-.5.7-1.2 1-1.8 1.5-.5.4-.7.8-.7 1.5" /><circle cx="12" cy="16.8" r="1" fill="currentColor" stroke="none" />',
  right: '<path d="m9 6 6 6-6 6" />',
  safe: '<path d="M12 3.5 5.5 6v5.8c0 4 2.7 6.8 6.5 8.7 3.8-1.9 6.5-4.7 6.5-8.7V6L12 3.5Z" /><path d="m9.4 12 1.7 1.8 3.6-3.8" />',
  save: '<path d="M5 5.5h11.5L19 8v10.5A1.5 1.5 0 0 1 17.5 20h-11A1.5 1.5 0 0 1 5 18.5Z" /><path d="M8 5.5v4h6v-4" /><rect x="8" y="13" width="8" height="5" rx="1" />',
  storage: '<ellipse cx="12" cy="6" rx="6.5" ry="2.5" /><path d="M5.5 6v4c0 1.4 2.9 2.5 6.5 2.5s6.5-1.1 6.5-2.5V6" /><path d="M5.5 10v4c0 1.4 2.9 2.5 6.5 2.5s6.5-1.1 6.5-2.5v-4" />',
  'sun-fill': '<circle cx="12" cy="12" r="4" fill="currentColor" stroke="none" /><path d="M12 2.5v2.2M12 19.3v2.2M21.5 12h-2.2M4.7 12H2.5M18.7 5.3l-1.6 1.6M6.9 17.1l-1.6 1.6M18.7 18.7l-1.6-1.6M6.9 6.9 5.3 5.3" />',
  user: '<circle cx="12" cy="8" r="3.2" /><path d="M5.5 18a6.5 6.5 0 0 1 13 0" />',
}

const svgMarkup = computed(() => iconMap[props.name])
const isFilled = computed(() => props.name === 'moon-fill' || props.name === 'sun-fill')
</script>

<template>
  <svg
    viewBox="0 0 24 24"
    width="1em"
    height="1em"
    aria-hidden="true"
    focusable="false"
    :fill="isFilled ? 'currentColor' : 'none'"
    :stroke="isFilled ? 'currentColor' : 'currentColor'"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    v-html="svgMarkup"
  />
</template>
