import { describe, expect, it } from 'vitest'

import {
  decodeBase64Url,
  encodeBase64Url,
  normalizeCreationOptions,
  normalizeRequestOptions,
} from './webauthn'

describe('webauthn helpers', () => {
  it('round-trips base64url encoded buffers', () => {
    const input = new Uint8Array([1, 2, 3, 250]).buffer
    const encoded = encodeBase64Url(input)

    expect(Array.from(decodeBase64Url(encoded))).toEqual([1, 2, 3, 250])
  })

  it('converts request challenge and allowCredentials ids into buffers', () => {
    const options = normalizeRequestOptions({
      challenge: 'AQID',
      allowCredentials: [{ id: 'BAUG', type: 'public-key' }],
    } as any)

    expect(options.challenge).toBeInstanceOf(ArrayBuffer)
    expect(options.allowCredentials?.[0]?.id).toBeInstanceOf(ArrayBuffer)
  })

  it('converts creation challenge, user id, and excluded credentials into buffers', () => {
    const options = normalizeCreationOptions({
      challenge: 'AQID',
      user: { id: 'BAUG', name: 'admin', displayName: 'admin' },
      excludeCredentials: [{ id: 'BwgJ', type: 'public-key' }],
    } as any)

    expect(options.challenge).toBeInstanceOf(ArrayBuffer)
    expect(options.user.id).toBeInstanceOf(ArrayBuffer)
    expect(options.excludeCredentials?.[0]?.id).toBeInstanceOf(ArrayBuffer)
  })
})
