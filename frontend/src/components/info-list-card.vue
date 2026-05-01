<script setup lang="ts">
type InfoListItem = {
  title: string
  description: string
  badge: string
  tone?: 'green' | 'blue' | 'gold' | 'red'
}

defineProps<{
  title: string
  items: InfoListItem[]
}>()
</script>

<template>
  <section class="surface-card section-card">
    <header class="section-card__header">
      <h3 class="section-card__title">{{ title }}</h3>
      <slot name="header" />
    </header>
    <div class="section-card__body list-block">
      <div v-for="item in items" :key="item.title" class="list-row">
        <div>
          <strong>{{ item.title }}</strong>
          <span>{{ item.description }}</span>
        </div>
        <a-tag :color="item.tone ?? 'blue'" bordered>{{ item.badge }}</a-tag>
      </div>
    </div>
  </section>
</template>

<style scoped src="./styles/section-card.css"></style>

<style scoped>
.list-block {
  display: grid;
  gap: 12px;
}

.list-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 15px 16px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-md);
  background: var(--surface-tertiary);
}

.list-row > div:first-child {
  min-width: 0;
}

.list-row strong,
.list-row span {
  display: block;
}

.list-row strong {
  font-size: 14px;
}

.list-row span {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 12px;
}
</style>
