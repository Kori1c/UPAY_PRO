/// <reference types="node" />

import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const globalCss = readFileSync(resolve(__dirname, './global.css'), 'utf8')
const loginViewSource = readFileSync(resolve(__dirname, '../views/login/login-view.vue'), 'utf8')
const settingsViewSource = readFileSync(resolve(__dirname, '../views/settings/settings-view.vue'), 'utf8')
const walletsViewSource = readFileSync(resolve(__dirname, '../views/wallets/wallets-view.vue'), 'utf8')
const metricCardSource = readFileSync(resolve(__dirname, '../components/metric-stat-card.vue'), 'utf8')
const infoListCardSource = readFileSync(resolve(__dirname, '../components/info-list-card.vue'), 'utf8')
const pageSectionCardSource = readFileSync(resolve(__dirname, '../components/page-section-card.vue'), 'utf8')
const formSectionCardSource = readFileSync(resolve(__dirname, '../components/form-section-card.vue'), 'utf8')

const deadSelectors = [
  '.admin-header__user',
  '.admin-mobile-trigger',
  '.page-toolbar__select',
  '.panel-grid',
  '.wallet-card__status',
  '.wallet-card__footer',
  '.wallet-card__metrics',
  '.wallet-card__metric',
  '.wallet-card__value',
  '.wallet-card__value--text',
  '.wallet-card__hint',
  '.wallet-card__extra',
  '.form-actions',
  '.token-chip',
  '.theme-toggle__button .arco-icon',
  '.admin-menu .arco-menu-item .arco-icon',
  '.admin-menu .arco-menu-selected .arco-icon',
  '.admin-mobile-bottom-nav__item .arco-icon',
  '.admin-mobile-bottom-nav__item--active .arco-icon',
] as const

const responsiveSnippetOccurrences = {
  '.admin-shell__sider {\n    display: none;\n  }': 0,
  '.admin-header__left {\n    min-width: 0;\n  }': 1,
  '.page-toolbar__search {\n    width: 100%;\n  }': 1,
  '.section-card {\n    padding: 18px;\n  }': 0,
  '.login-form-panel {\n    justify-content: center;': 0,
  '.theme-toggle {\n    padding: 2px;\n  }': 0,
  '.wallet-card__topbar,\n  .wallet-card__body {\n    grid-template-columns: 1fr;': 0,
  '.wallet-card__topbar,\n  .wallet-card__body {\n    grid-template-columns: 1fr;\n    flex-direction: column;': 0,
} as const

function countOccurrences(source: string, snippet: string) {
  return source.split(snippet).length - 1
}

const loginOnlySelectors = [
  '.login-shell',
  '.login-form-panel',
  '.login-form-card',
  '.login-submit',
  '.login-passkey',
  '.login-passkey-only',
] as const

const settingsOnlySelectors = [
  '.settings-grid',
  '.settings-warning-card',
] as const

const walletOnlySelectors = [
  '.wallet-grid',
  '.wallet-card',
  '.wallet-mini-card',
  '.wallet-mini-card__header',
  '.wallet-mini-card__title',
  '.wallet-card__topbar',
  '.wallet-card__chips',
  '.wallet-card__rate-inline',
  '.wallet-card__label',
  '.wallet-card__body',
  '.wallet-card__address-block',
  '.wallet-card__actions',
] as const

const metricOnlySelectors = [
  '.metric-card',
  '.metric-card__label',
  '.metric-card__value',
  '.metric-card__delta',
  '.metric-card__dot',
] as const

const infoListOnlySelectors = [
  '.list-block',
  '.list-row',
] as const

const sectionCardOnlySelectors = [
  '.section-card',
  '.section-card__header',
  '.section-card__title',
  '.section-card__description',
  '.section-card__body',
] as const

describe('global style hygiene', () => {
  it('does not keep dead global selectors that are no longer used by the app shell', () => {
    const leftovers = deadSelectors.filter((selector) => globalCss.includes(selector))
    expect(leftovers).toEqual([])
  })

  it('does not keep redundant responsive snippets after layout consolidation', () => {
    const mismatches = Object.entries(responsiveSnippetOccurrences).filter(
      ([snippet, expectedCount]) => countOccurrences(globalCss, snippet) !== expectedCount,
    )
    expect(mismatches).toEqual([])
  })

  it('keeps login-only styles out of the global stylesheet', () => {
    const leakedSelectors = loginOnlySelectors.filter((selector) => globalCss.includes(selector))

    expect(leakedSelectors).toEqual([])
    expect(loginViewSource).toContain('<style scoped>')
  })

  it('keeps settings-only styles out of the global stylesheet', () => {
    const leakedSelectors = settingsOnlySelectors.filter((selector) => globalCss.includes(selector))

    expect(leakedSelectors).toEqual([])
    expect(settingsViewSource).toContain('<style scoped>')
  })

  it('keeps wallet-only styles out of the global stylesheet', () => {
    const leakedSelectors = walletOnlySelectors.filter((selector) => globalCss.includes(selector))

    expect(leakedSelectors).toEqual([])
    expect(walletsViewSource).toContain('<style>')
  })

  it('keeps metric card styles out of the global stylesheet', () => {
    const leakedSelectors = metricOnlySelectors.filter((selector) => globalCss.includes(selector))

    expect(leakedSelectors).toEqual([])
    expect(metricCardSource).toContain('<style scoped>')
  })

  it('keeps info list styles out of the global stylesheet', () => {
    const leakedSelectors = infoListOnlySelectors.filter((selector) => globalCss.includes(selector))

    expect(leakedSelectors).toEqual([])
    expect(infoListCardSource).toContain('<style scoped>')
  })

  it('keeps shared section card styles out of the global stylesheet', () => {
    const leakedSelectors = sectionCardOnlySelectors.filter((selector) => globalCss.includes(selector))

    expect(leakedSelectors).toEqual([])
    expect(pageSectionCardSource).toContain('section-card.css')
    expect(formSectionCardSource).toContain('section-card.css')
    expect(infoListCardSource).toContain('section-card.css')
  })
})
