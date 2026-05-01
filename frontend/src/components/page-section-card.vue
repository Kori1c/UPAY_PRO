<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
}>()
</script>

<template>
  <section class="surface-card section-card">
    <header v-if="title || description || $slots.header" class="section-card__header">
      <div v-if="title || description">
        <h3 v-if="title" class="section-card__title">{{ title }}</h3>
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
