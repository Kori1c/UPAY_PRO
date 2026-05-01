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
.metric-card {
  padding: 14px 18px;
}

.metric-card__label {
  color: var(--text-secondary);
  font-size: 12px;
}

.metric-card__value {
  margin: 6px 0 8px;
  font-size: 26px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: -0.02em;
}

.metric-card__delta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.metric-card__dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--accent);
  box-shadow: none;
}

.metric-card__dot[data-tone='success'] {
  background: var(--success);
}

.metric-card__dot[data-tone='warning'] {
  background: var(--warning);
}

.metric-card__dot[data-tone='danger'] {
  background: var(--danger);
}

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
