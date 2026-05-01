import type { RouteRecordRaw } from 'vue-router'
import {
  IconApps,
  IconDashboard,
  IconLock,
  IconSafe,
  IconStorage,
} from '@arco-design/web-vue/es/icon'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    requiresAuth?: boolean
    menuKey?: string
  }
}

export type NavigationItem = {
  key: string
  label: string
  path: string
  icon: unknown
}

export const adminNavigationItems: NavigationItem[] = [
  {
    key: 'dashboard',
    label: '仪表盘',
    path: '/dashboard',
    icon: IconDashboard,
  },
  {
    key: 'orders',
    label: '订单管理',
    path: '/orders',
    icon: IconApps,
  },
  {
    key: 'wallets',
    label: '钱包管理',
    path: '/wallets',
    icon: IconStorage,
  },
  {
    key: 'settings',
    label: '系统设置',
    path: '/settings',
    icon: IconSafe,
  },
  {
    key: 'apikeys',
    label: '密钥',
    path: '/apikeys',
    icon: IconLock,
  },
]

export const routeDefinitions: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/login/login-view.vue'),
    meta: {
      title: '登录',
    },
  },
  {
    path: '/',
    component: () => import('../layouts/admin-layout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('../views/dashboard/dashboard-view.vue'),
        meta: {
          title: '仪表盘',
          requiresAuth: true,
          menuKey: 'dashboard',
        },
      },
      {
        path: 'orders',
        name: 'orders',
        component: () => import('../views/orders/orders-view.vue'),
        meta: {
          title: '订单管理',
          requiresAuth: true,
          menuKey: 'orders',
        },
      },
      {
        path: 'wallets',
        name: 'wallets',
        component: () => import('../views/wallets/wallets-view.vue'),
        meta: {
          title: '钱包管理',
          requiresAuth: true,
          menuKey: 'wallets',
        },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('../views/settings/settings-view.vue'),
        meta: {
          title: '系统设置',
          requiresAuth: true,
          menuKey: 'settings',
        },
      },
      {
        path: 'apikeys',
        name: 'apikeys',
        component: () => import('../views/apikeys/apikeys-view.vue'),
        meta: {
          title: 'API Keys',
          requiresAuth: true,
          menuKey: 'apikeys',
        },
      },
    ],
  },
]
