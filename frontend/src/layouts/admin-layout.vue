<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import AppThemeToggle from '../components/app-theme-toggle.vue'
import AppIcon from '../components/icons/app-icon.vue'
import { adminNavigationItems } from '../router/navigation'
import { useAppStore } from '../stores/app'

import { adminApi } from '../api'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const menuWrapRef = ref<HTMLElement | null>(null)
const activePillStyle = ref({
  opacity: 0,
  transform: 'translateY(0px)',
  height: '48px',
  width: '0px',
  left: '0px',
})

const selectedKeys = computed(() => {
  const menuKey = route.meta.menuKey
  return menuKey ? [menuKey] : []
})

const selectedIndex = computed(() => {
  const currentKey = selectedKeys.value[0]
  const index = adminNavigationItems.findIndex((item) => item.key === currentKey)
  return index >= 0 ? index : 0
})

function updateActivePill() {
  nextTick(() => {
    const menuWrapEl = menuWrapRef.value
    if (!menuWrapEl) return

    const selectedItem = menuWrapEl.querySelector('.arco-menu-selected') as HTMLElement | null
    if (!selectedItem) {
      activePillStyle.value = {
        opacity: 0,
        transform: 'translateY(0px)',
        height: '48px',
        width: '0px',
        left: '0px',
      }
      return
    }

    activePillStyle.value = {
      opacity: 1,
      transform: `translateY(${selectedItem.offsetTop}px)`,
      height: `${selectedItem.offsetHeight}px`,
      width: `${selectedItem.offsetWidth}px`,
      left: `${selectedItem.offsetLeft}px`,
    }
  })
}

watch(
  () => route.meta.title,
  (title) => {
    if (typeof title === 'string' && title.length > 0) {
      appStore.setPageTitle(title)
      document.title = `${title} | UPay Pro Admin`
    }
  },
  {
    immediate: true,
  },
)

watch(selectedIndex, () => {
  updateActivePill()
})

watch(
  () => route.fullPath,
  () => {
    updateActivePill()
  },
)

onMounted(() => {
  updateActivePill()
  window.addEventListener('resize', updateActivePill)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateActivePill)
})

async function handleMenuClick(path: string) {
  if (route.path === path) return
  await router.push(path)
}

async function handleLogout() {
  try {
    await adminApi.logout()
  } catch (error) {
    console.error('Logout request failed', error)
  } finally {
    // Clear cookie (optional as backend does it, but good for local state)
    document.cookie = "token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;"
    router.push('/login')
  }
}
</script>

<template>
  <a-layout class="admin-shell">
    <a-layout-sider
      :width="250"
      class="admin-shell__sider"
    >
      <div class="admin-shell__sider-inner">
        <div class="admin-shell__sider-main">
          <div class="admin-brand">
            <div class="admin-brand__mark">
              <span class="admin-brand__coin" aria-hidden="true">
                <svg viewBox="0 0 48 48" role="img" focusable="false">
                  <circle cx="24" cy="24" r="22" />
                  <path
                    d="M14 14.5h20v5.1h-7.1v4.1c5.4.3 9.3 1.5 9.3 3s-3.9 2.7-9.3 3v8.8h-5.8v-8.8c-5.4-.3-9.3-1.5-9.3-3s3.9-2.7 9.3-3v-4.1H14v-5.1Zm10 11.3c-4.5 0-8.1.5-8.1 1s3.6 1 8.1 1 8.1-.5 8.1-1-3.6-1-8.1-1Z"
                  />
                </svg>
              </span>
              <span class="admin-brand__wordmark">
                <strong>UPAY PRO</strong>
                <span>USDT PAYMENT</span>
              </span>
            </div>
          </div>

          <div ref="menuWrapRef" class="admin-menu-wrap">
            <a-menu :selected-keys="selectedKeys" auto-open class="admin-menu">
              <div
                class="admin-menu__active-pill"
                :style="activePillStyle"
              />
              <a-menu-item
                v-for="item in adminNavigationItems"
                :key="item.key"
                @click="handleMenuClick(item.path)"
              >
                <template #icon>
                  <app-icon :name="item.icon" />
                </template>
                {{ item.label }}
              </a-menu-item>
            </a-menu>
          </div>
        </div>

        <div class="admin-sider-footer">
          <a-button long type="outline" status="danger" @click="handleLogout">
            <template #icon>
              <app-icon name="poweroff" />
            </template>
            退出登录
          </a-button>
        </div>
      </div>
    </a-layout-sider>

    <a-layout class="admin-shell__main">
      <a-layout-header class="admin-header">
        <div class="admin-header__left">
          <div>
            <span class="admin-header__label">Control Center</span>
            <strong>{{ appStore.pageTitle }}</strong>
          </div>
        </div>

        <div class="admin-header__right">
          <app-theme-toggle />
        </div>
      </a-layout-header>

      <a-layout-content class="admin-content">
        <router-view />
      </a-layout-content>
    </a-layout>

    <nav
      class="admin-mobile-bottom-nav"
      aria-label="移动端导航"
      :style="{ '--mobile-nav-index': String(selectedIndex) }"
    >
      <div class="admin-mobile-bottom-nav__rail">
        <div class="admin-mobile-bottom-nav__pill" />
        <RouterLink
          v-for="item in adminNavigationItems"
          :key="item.key"
          :to="item.path"
          class="admin-mobile-bottom-nav__item"
          :class="{ 'admin-mobile-bottom-nav__item--active': selectedKeys[0] === item.key }"
        >
          <app-icon :name="item.icon" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </div>
    </nav>
  </a-layout>
</template>
