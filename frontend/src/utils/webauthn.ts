type BufferLikeOption = ArrayBuffer | Uint8Array | string

type PublicKeyRequestOptionsFromApi = Omit<PublicKeyCredentialRequestOptions, 'challenge' | 'allowCredentials'> & {
  challenge: BufferLikeOption
  allowCredentials?: Array<Omit<PublicKeyCredentialDescriptor, 'id'> & { id: BufferLikeOption }>
}

type PublicKeyCreationOptionsFromApi = Omit<PublicKeyCredentialCreationOptions, 'challenge' | 'user' | 'excludeCredentials'> & {
  challenge: BufferLikeOption
  user: Omit<PublicKeyCredentialUserEntity, 'id'> & { id: BufferLikeOption }
  excludeCredentials?: Array<Omit<PublicKeyCredentialDescriptor, 'id'> & { id: BufferLikeOption }>
}

export interface SerializedPublicKeyCredential {
  id: string
  rawId: string
  type: PublicKeyCredentialType
  authenticatorAttachment?: AuthenticatorAttachment | null
  clientExtensionResults: AuthenticationExtensionsClientOutputs
  response: Record<string, unknown>
}

function toUint8Array(value: BufferLikeOption): Uint8Array {
  if (value instanceof Uint8Array) {
    return value
  }
  if (value instanceof ArrayBuffer) {
    return new Uint8Array(value)
  }
  return decodeBase64Url(value)
}

function toArrayBuffer(value: BufferLikeOption): ArrayBuffer {
  const bytes = toUint8Array(value)
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

export function decodeBase64Url(value: string): Uint8Array {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  const binary = atob(padded)
  return Uint8Array.from(binary, (char) => char.charCodeAt(0))
}

export function encodeBase64Url(value: ArrayBuffer | Uint8Array | null): string {
  if (!value) {
    return ''
  }

  const bytes = value instanceof Uint8Array ? value : new Uint8Array(value)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })

  return btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/g, '')
}

export function supportsPasskey(): boolean {
  return getPasskeyUnavailableReason() === ''
}

export function getPasskeyUnavailableReason(): string {
  if (typeof window === 'undefined') {
    return '当前环境不支持 Passkey'
  }

  if (window.location.hostname === '127.0.0.1' || window.location.hostname === '::1') {
    return '本地测试 Passkey 请使用 localhost 访问，不要使用 127.0.0.1'
  }

  if (!window.isSecureContext) {
    return 'Passkey 需要 HTTPS 或 localhost 安全环境'
  }

  if (
    typeof window.PublicKeyCredential === 'undefined'
    || typeof navigator.credentials?.create !== 'function'
    || typeof navigator.credentials?.get !== 'function'
  ) {
    return '当前浏览器不支持 Passkey'
  }

  return ''
}

export function normalizeRequestOptions(
  options: PublicKeyRequestOptionsFromApi,
): PublicKeyCredentialRequestOptions {
  return {
    ...options,
    challenge: toArrayBuffer(options.challenge),
    allowCredentials: options.allowCredentials?.map((credential) => ({
      ...credential,
      id: toArrayBuffer(credential.id),
    })),
  }
}

export function normalizeCreationOptions(
  options: PublicKeyCreationOptionsFromApi,
): PublicKeyCredentialCreationOptions {
  return {
    ...options,
    challenge: toArrayBuffer(options.challenge),
    user: {
      ...options.user,
      id: toArrayBuffer(options.user.id),
    },
    excludeCredentials: options.excludeCredentials?.map((credential) => ({
      ...credential,
      id: toArrayBuffer(credential.id),
    })),
  }
}

export async function createRegistrationCredential(
  options: PublicKeyCreationOptionsFromApi,
): Promise<PublicKeyCredential> {
  const credential = await navigator.credentials.create({
    publicKey: normalizeCreationOptions(options),
  })

  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error('Passkey 注册已取消')
  }

  return credential
}

export async function getAssertionCredential(
  options: PublicKeyRequestOptionsFromApi,
): Promise<PublicKeyCredential> {
  const credential = await navigator.credentials.get({
    publicKey: normalizeRequestOptions(options),
  })

  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error('Passkey 登录已取消')
  }

  return credential
}

export function serializeRegistrationCredential(
  credential: PublicKeyCredential,
): SerializedPublicKeyCredential {
  const response = credential.response as AuthenticatorAttestationResponse

  return {
    id: credential.id,
    rawId: encodeBase64Url(credential.rawId),
    type: credential.type as PublicKeyCredentialType,
    authenticatorAttachment: credential.authenticatorAttachment as AuthenticatorAttachment | null,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      clientDataJSON: encodeBase64Url(response.clientDataJSON),
      attestationObject: encodeBase64Url(response.attestationObject),
      transports: typeof response.getTransports === 'function' ? response.getTransports() : [],
    },
  }
}

export function serializeAssertionCredential(
  credential: PublicKeyCredential,
): SerializedPublicKeyCredential {
  const response = credential.response as AuthenticatorAssertionResponse

  return {
    id: credential.id,
    rawId: encodeBase64Url(credential.rawId),
    type: credential.type as PublicKeyCredentialType,
    authenticatorAttachment: credential.authenticatorAttachment as AuthenticatorAttachment | null,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      authenticatorData: encodeBase64Url(response.authenticatorData),
      clientDataJSON: encodeBase64Url(response.clientDataJSON),
      signature: encodeBase64Url(response.signature),
      userHandle: encodeBase64Url(response.userHandle),
    },
  }
}
