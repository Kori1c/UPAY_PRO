import { defineStore } from 'pinia'
import {
  DEFAULT_THEME_PREFERENCE,
  THEME_STORAGE_KEY,
  type ThemePreference,
  type ThemeValue,
  resolveThemePreference,
} from '../app/theme'

export const useAppStore = defineStore('app', {
  state: () => ({
    pageTitle: '仪表盘',
    themePreference: DEFAULT_THEME_PREFERENCE as ThemePreference,
    resolvedTheme: 'light' as ThemeValue,
  }),
  actions: {
    setPageTitle(title: string) {
      this.pageTitle = title
    },
    hydrateThemePreference() {
      this.themePreference = resolveThemePreference(
        window.localStorage.getItem(THEME_STORAGE_KEY),
      )
    },
    setThemePreference(preference: ThemePreference) {
      this.themePreference = preference
      window.localStorage.setItem(THEME_STORAGE_KEY, preference)
    },
    toggleThemePreference() {
      const nextPreference: ThemePreference =
        this.resolvedTheme === 'dark' ? 'light' : 'dark'

      this.setThemePreference(nextPreference)
    },
    setResolvedTheme(theme: ThemeValue) {
      this.resolvedTheme = theme
    },
  },
})
