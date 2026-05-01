<script setup lang="ts">
defineProps<{
  label: string
  value: string
  hint: string
  tone?: 'success' | 'warning' | 'danger' | 'info'
  clickable?: boolean
}>()

defineEmits(['click'])
</script>

<template>
  <article 
    class="surface-card metric-card" 
    :class="{ 'metric-card--clickable': clickable }"
    @click="clickable ? $emit('click') : undefined"
  >
    <span class="metric-card__label">{{ label }}</span>
    <div class="metric-card__value">{{ value }}</div>
    <div class="metric-card__delta">
      <span class="metric-card__dot" :data-tone="tone ?? 'info'" />
      {{ hint }}
    </div>
  </article>
</template>

<style scoped>
.metric-card--clickable {
  cursor: pointer;
  transition: all 0.2s ease;
}
.metric-card--clickable:hover {
  transform: translateY(-2px);
  border-color: var(--accent);
  box-shadow: var(--shadow-soft);
}
</style>
