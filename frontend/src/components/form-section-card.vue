<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <section class="surface-card section-card">
    <header class="section-card__header">
      <div>
        <h3 class="section-card__title">{{ title }}</h3>
        <p v-if="description" class="section-card__description">{{ description }}</p>
      </div>
      <slot name="header" />
    </header>
    <div class="section-card__body">
      <slot />
    </div>
  </section>
</template>

<style scoped src="./styles/section-card.css"></style>
