export type ThemePreference = 'system' | 'light' | 'dark'
export type ThemeValue = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'upay-theme-preference'
export const DEFAULT_THEME_PREFERENCE: ThemePreference = 'system'

export function resolveThemePreference(
  value: string | null | undefined,
): ThemePreference {
  if (value === 'light' || value === 'dark' || value === 'system') {
    return value
  }

  return DEFAULT_THEME_PREFERENCE
}

export function resolveSystemTheme(
  matchesDark: boolean | undefined,
): ThemeValue {
  return matchesDark ? 'dark' : 'light'
}

export function resolveThemeValue(
  preference: ThemePreference,
  systemTheme: ThemeValue,
): ThemeValue {
  if (preference === 'system') {
    return systemTheme
  }

  return preference
}

export function applyThemeToDocument(theme: ThemeValue) {
  document.documentElement.dataset.theme = theme
  document.body.setAttribute('arco-theme', theme)
}
