import { describe, expect, it } from 'vitest'

import { adminNavigationItems, routeDefinitions } from './navigation'

describe('navigation scaffold', () => {
  it('defines the login route and five admin routes', () => {
    const routeNames = routeDefinitions.flatMap((route) => {
      const parentName = typeof route.name === 'string' ? [route.name] : []
      const childNames =
        route.children?.flatMap((child) =>
          typeof child.name === 'string' ? [child.name] : [],
        ) ?? []

      return [...parentName, ...childNames]
    })

    expect(routeNames).toContain('login')
    expect(adminNavigationItems).toHaveLength(5)
    expect(routeNames).toEqual([
      'login',
      'dashboard',
      'orders',
      'wallets',
      'settings',
      'apikeys',
    ])
  })
})
