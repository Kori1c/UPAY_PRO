const API_BASE = import.meta.env.DEV ? '/__api' : ''

function toRequestUrl(url: string) {
  if (/^https?:\/\//.test(url)) {
    return url
  }
  return `${API_BASE}${url}`
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const defaultOptions: RequestInit = {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  }

  const response = await fetch(toRequestUrl(url), defaultOptions)

  if (response.status === 401) {
    // Unauthorized, redirect to login if not already there
    if (!window.location.pathname.endsWith('/login')) {
      window.location.href = '/login'
    }
    throw new Error('Unauthorized')
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Network response was not ok' }))
    throw new Error(error.message || 'Request failed')
  }

  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) {
    return {} as T
  }

  return response.json()
}

export const api = {
  get: <T>(url: string) => request<T>(url, { method: 'GET' }),
  post: <T>(url: string, data?: any) =>
    request<T>(url, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  put: <T>(url: string, data?: any) =>
    request<T>(url, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: <T>(url: string) => request<T>(url, { method: 'DELETE' }),
}
