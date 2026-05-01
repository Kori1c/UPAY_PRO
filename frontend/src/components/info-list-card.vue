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
