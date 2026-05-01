import { describe, expect, it } from 'vitest'

import {
  DEFAULT_THEME_PREFERENCE,
  resolveThemePreference,
  resolveThemeValue,
} from './theme'

describe('theme helpers', () => {
  it('falls back to the default preference for unknown values', () => {
    expect(resolveThemePreference('unexpected')).toBe(DEFAULT_THEME_PREFERENCE)
    expect(resolveThemePreference(null)).toBe(DEFAULT_THEME_PREFERENCE)
  })

  it('maps system preference to the current system theme', () => {
    expect(resolveThemeValue('system', 'dark')).toBe('dark')
    expect(resolveThemeValue('system', 'light')).toBe('light')
  })

  it('returns explicit theme values unchanged', () => {
    expect(resolveThemeValue('light', 'dark')).toBe('light')
    expect(resolveThemeValue('dark', 'light')).toBe('dark')
  })
})
