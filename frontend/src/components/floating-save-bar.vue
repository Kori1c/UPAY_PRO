<script setup lang="ts">
import AppIcon from './icons/app-icon.vue'

defineProps<{
  show: boolean
  loading?: boolean
}>()

defineEmits(['save'])
</script>

<template>
  <transition name="capsule-fade">
    <div v-if="show" class="save-capsule">
      <div class="save-capsule__inner">
        <div class="save-capsule__hint">
          <app-icon name="info-circle" class="hint-icon" />
          <span>配置有改动，请及时保存</span>
        </div>
        <a-button 
          type="primary" 
          size="medium" 
          :loading="loading" 
          class="save-btn"
          @click="$emit('save')"
        >
          <template #icon><app-icon name="save" /></template>
          立即保存
        </a-button>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.save-capsule {
  position: fixed;
  bottom: 40px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2000;
  width: auto;
  pointer-events: none;
}

.save-capsule__inner {
  pointer-events: auto;
  background: var(--body-background);
  border: 1px solid var(--border-strong);
  box-shadow: 
    0 4px 6px -1px rgba(0, 0, 0, 0.1),
    0 20px 25px -5px rgba(0, 0, 0, 0.15),
    0 10px 10px -5px rgba(0, 0, 0, 0.04);
  padding: 8px 10px 8px 20px;
  border-radius: 99px;
  display: flex;
  align-items: center;
  gap: 24px;
  backdrop-filter: blur(12px);
}

.save-capsule__hint {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
}

.hint-icon {
  color: var(--accent);
  font-size: 16px;
}

.save-btn {
  height: 38px !important;
  border-radius: 99px !important;
  padding: 0 20px !important;
  font-weight: 600 !important;
  background: var(--accent) !important;
  border: none !important;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.2) !important;
}

.save-btn:hover {
  background: var(--accent-strong) !important;
  transform: translateY(-1px);
}

/* Transition */
.capsule-fade-enter-active,
.capsule-fade-leave-active {
  transition: all 0.5s cubic-bezier(0.19, 1, 0.22, 1);
}

.capsule-fade-enter-from,
.capsule-fade-leave-to {
  opacity: 0;
  transform: translate(-50%, 30px) scale(0.95);
}

@media (max-width: 768px) {
  .save-capsule {
    right: 16px;
    bottom: calc(96px + env(safe-area-inset-bottom));
    left: 16px;
    width: auto;
    transform: none;
  }

  .save-capsule__inner {
    justify-content: space-between;
    gap: 10px;
    padding: 8px 8px 8px 14px;
  }

  .save-capsule__hint {
    min-width: 0;
    font-size: 13px;
  }

  .save-capsule__hint span {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .save-btn {
    flex: 0 0 auto;
    height: 36px !important;
    padding: 0 16px !important;
  }

  .capsule-fade-enter-from,
  .capsule-fade-leave-to {
    transform: translateY(20px) scale(0.98);
  }
}
</style>
