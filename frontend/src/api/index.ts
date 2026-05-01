import { api } from './request'

export interface Stats {
  userCount: number
  successOrderCount: number
  pendingOrderCount: number
  walletCount: number
  todayAmount: number
  yesterdayAmount: number
  totalAmount: number
  todayOrderCount: number
  currentMonthSuccessOrderCount: number
}

export interface Order {
  id: number
  CreatedAt: string
  trade_id: string
  order_id: string
  amount: number
  actual_amount: number
  type: string
  token: string
  status: number
  callback_num: number
  call_back_confirm: number
}

type OrderApiRecord = Order & {
  ID?: number
  TradeId?: string
  OrderId?: string
  Amount?: number
  ActualAmount?: number
  Type?: string
  Token?: string
  Status?: number
  CallbackNum?: number
  CallBackConfirm?: number
}

export interface OrdersResponse {
  orders: Order[]
  total: number
  page: number
  limit: number
}

export interface Wallet {
  id?: number
  currency: string
  token: string
  status: number
  rate: number
  AutoRate: boolean
}

type WalletApiRecord = Wallet & {
  ID?: number
  Currency?: string
  Token?: string
  Status?: number
  Rate?: number
}

export interface Setting {
  AppUrl: string
  SecretKey: string
  Httpport: number
  Tgbotkey: string
  Tgchatid: string
  Barkkey: string
  Redishost: string
  Redisport: number
  Redispasswd: string
  Redisdb: number
  AppName: string
  CustomerServiceContact: string
  ExpirationDate: number
}

export interface ApiKey {
  Tronscan: string
  Trongrid: string
  Etherscan: string
}

export interface AdminAccount {
  id: number
  username: string
}

export interface LoginAuthConfig {
  passwordLoginEnabled: boolean
  passkeySupported: boolean
}

export interface PasskeyItem {
  id: number
  credentialId: string
  deviceLabel: string
  transports: string[]
  createdAt: string
  lastUsedAt?: string | null
}

export interface PasskeySettings {
  passwordLoginEnabled: boolean
  passkeys: PasskeyItem[]
}

export interface PasskeyBeginResponse {
  challengeId: string
  publicKey: any
}

export interface PasskeyCeremonyPayload {
  challengeId: string
  credential: any
}

function normalizeWallet(wallet: WalletApiRecord): Wallet {
  return {
    id: wallet.id ?? wallet.ID,
    currency: wallet.currency ?? wallet.Currency ?? '',
    token: wallet.token ?? wallet.Token ?? '',
    status: wallet.status ?? wallet.Status ?? 1,
    rate: wallet.rate ?? wallet.Rate ?? 0,
    AutoRate: wallet.AutoRate ?? false,
  }
}

function normalizeOrder(order: OrderApiRecord): Order {
  return {
    id: order.id ?? order.ID ?? 0,
    CreatedAt: order.CreatedAt ?? '',
    trade_id: order.trade_id ?? order.TradeId ?? '',
    order_id: order.order_id ?? order.OrderId ?? '',
    amount: order.amount ?? order.Amount ?? 0,
    actual_amount: order.actual_amount ?? order.ActualAmount ?? 0,
    type: order.type ?? order.Type ?? '',
    token: order.token ?? order.Token ?? '',
    status: order.status ?? order.Status ?? 0,
    callback_num: order.callback_num ?? order.CallbackNum ?? 0,
    call_back_confirm: order.call_back_confirm ?? order.CallBackConfirm ?? 0,
  }
}

export const adminApi = {
  // Stats
  getStats: () => api.get<{ code: number; data: Stats }>('/admin/api/stats'),

  // Orders
  getOrders: (params: { page: number; limit: number; search?: string; status?: string }) => {
    const query = new URLSearchParams({
      page: params.page.toString(),
      limit: params.limit.toString(),
    })
    if (params.search) query.append('search', params.search)
    if (params.status) query.append('status', params.status)
    return api
      .get<{ code: number; data: { orders: OrderApiRecord[]; total: number; page: number; limit: number } }>(
        `/admin/api/orders?${query.toString()}`,
      )
      .then((res) => ({
        ...res,
        data: {
          ...res.data,
          orders: Array.isArray(res.data?.orders) ? res.data.orders.map(normalizeOrder) : [],
        },
      }))
  },

  // Account
  getAccount: () => api.get<{ code: number; data: AdminAccount }>('/admin/api/account'),
  updateAccount: (payload: { username: string; password?: string }) =>
    api.post<{ code: number; message: string; relogin?: boolean }>('/admin/api/account', payload),
  manualCompleteOrder: (orderId: string) =>
    api.post<{ code: number; message: string }>('/admin/api/manual-complete-order', { order_id: orderId }),

  // Wallets
  getWallets: async () => {
    const res = await api.get<{ code: number; data: WalletApiRecord[] }>('/admin/api/wallets')
    return {
      ...res,
      data: Array.isArray(res.data) ? res.data.map(normalizeWallet) : [],
    }
  },
  addWallet: async (wallet: Wallet) => {
    const res = await api.post<{ code: number; message: string; data?: WalletApiRecord }>('/admin/api/wallets', wallet)
    return {
      ...res,
      data: res.data ? normalizeWallet(res.data) : undefined,
    }
  },
  updateWallet: (id: number, wallet: Wallet) =>
    api.put<{ code: number; message: string }>(`/admin/api/wallets/${id}`, wallet),
  deleteWallet: (id: number) => api.delete<{ code: number; message: string }>(`/admin/api/wallets/${id}`),

  // Settings
  getSettings: () => api.get<{ code: number; data: Setting }>('/admin/api/settings'),
  getSecretKey: () => api.get<{ code: number; data: { SecretKey: string } }>('/admin/api/settings/secret-key'),
  saveSettings: (settings: Partial<Setting>) =>
    api.post<{ code: number; message: string }>('/admin/api/settings', settings),

  // API Keys
  getApiKeys: () => api.get<{ code: number; data: ApiKey }>('/admin/api/apikeys'),
  saveApiKeys: (keys: Partial<ApiKey>) => api.post<{ code: number; message: string }>('/admin/api/apikeys', keys),

  // Auth
  getLoginAuthConfig: () => api.get<{ code: number; data: LoginAuthConfig }>('/login/auth-config'),
  login: (data: any) => api.post<{ code: number; message: string }>('/login', data),
  beginPasskeyLogin: () =>
    api.post<{ code: number; data: PasskeyBeginResponse }>('/login/passkey/options', {}),
  finishPasskeyLogin: (payload: PasskeyCeremonyPayload) =>
    api.post<{ code: number; message: string }>('/login/passkey/verify', payload),
  logout: () => api.post<{ code: number; message: string }>('/admin/logout'),

  // Passkeys
  getPasskeys: () => api.get<{ code: number; data: PasskeySettings }>('/admin/api/passkeys'),
  beginPasskeyRegistration: () =>
    api.post<{ code: number; data: PasskeyBeginResponse }>('/admin/api/passkeys/register/options', {}),
  finishPasskeyRegistration: (payload: PasskeyCeremonyPayload) =>
    api.post<{ code: number; data: PasskeyItem }>('/admin/api/passkeys/register/verify', payload),
  setPasswordLoginEnabled: (enabled: boolean) =>
    api.post<{ code: number; message: string }>('/admin/api/passkeys/password-login', { enabled }),
  deletePasskey: (id: number) => api.delete<{ code: number; message: string }>(`/admin/api/passkeys/${id}`),
}
