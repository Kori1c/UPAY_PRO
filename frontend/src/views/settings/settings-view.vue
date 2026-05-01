<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconCopy, IconQuestionCircle, IconSafe } from '@arco-design/web-vue/es/icon'
import { adminApi, type PasskeyItem, type Setting } from '../../api'
import FormSectionCard from '../../components/form-section-card.vue'
import FloatingSaveBar from '../../components/floating-save-bar.vue'
import {
  createRegistrationCredential,
  getPasskeyUnavailableReason,
  serializeRegistrationCredential,
  supportsPasskey,
} from '../../utils/webauthn'

const router = useRouter()
const loading = ref(false)
const isDirty = ref(false)
const initialForm = ref('')
const initialAccountForm = ref('')
const initialRedisSignature = ref('')
const accountLoading = ref(false)
const passkeyLoading = ref(false)
const passkeyBusy = ref(false)
const settingForm = reactive<Partial<Setting>>({
  AppName: '',
  AppUrl: '',
  Httpport: 8090,
  ExpirationDate: 10 as any,
  CustomerServiceContact: '',
  SecretKey: '',
  Redishost: '127.0.0.1',
  Redisport: 6379,
  Redispasswd: '',
  Redisdb: 0,
  Tgbotkey: '',
  Tgchatid: '',
  Barkkey: '',
})
const accountForm = reactive({
  username: '',
  password: '',
  confirmPassword: '',
})
const passkeyState = reactive({
  passwordLoginEnabled: true,
  passkeys: [] as PasskeyItem[],
})

function currentOrigin() {
  if (typeof window === 'undefined') {
    return ''
  }
  return window.location.origin
}

function shouldAutofillAppUrl(value?: string) {
  if (!value) {
    return true
  }

  const normalized = value.trim().toLowerCase()
  return normalized === 'http://localhost' || normalized === 'http://127.0.0.1' || normalized === 'http://localhost:8090'
}

function generateSecretKey(length = 48) {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_'
  const bytes = new Uint8Array(length)
  crypto.getRandomValues(bytes)

  return Array.from(bytes, (value) => chars[value % chars.length]).join('')
}

function handleGenerateSecretKey() {
  settingForm.SecretKey = generateSecretKey()
}

async function handleCopySecretKey() {
  if (!settingForm.SecretKey) {
    Message.warning('当前没有可复制的通信密钥')
    return
  }

  try {
    const res = await adminApi.getSecretKey()
    const secretKey = res.code === 0 ? res.data.SecretKey : ''
    if (!secretKey) {
      Message.warning('当前没有可复制的通信密钥')
      return
    }
    await navigator.clipboard.writeText(secretKey)
    Message.success('通信密钥已复制')
  } catch (error) {
    Message.error('复制失败，请手动复制')
  }
}

async function fetchSettings() {
  loading.value = true
  try {
    const res = await adminApi.getSettings()
    if (res.code === 0) {
      Object.assign(settingForm, res.data)
      if (shouldAutofillAppUrl(res.data.AppUrl)) {
        settingForm.AppUrl = currentOrigin()
      }
      if (!settingForm.SecretKey) {
        settingForm.SecretKey = generateSecretKey()
      }
      if (typeof res.data.ExpirationDate === 'number') {
        settingForm.ExpirationDate = (res.data.ExpirationDate / 1000000000 / 60) as any
      }
      initialForm.value = JSON.stringify(settingForm)
      initialRedisSignature.value = JSON.stringify({
        redishost: settingForm.Redishost,
        redisport: settingForm.Redisport,
        redispasswd: settingForm.Redispasswd,
        redisdb: settingForm.Redisdb,
      })
      isDirty.value = false
    }
  } catch (error) {
    Message.error('获取设置失败')
  } finally {
    loading.value = false
  }
}

async function fetchAccount() {
  accountLoading.value = true
  try {
    const res = await adminApi.getAccount()
    if (res.code === 0) {
      accountForm.username = res.data.username
      accountForm.password = ''
      accountForm.confirmPassword = ''
      initialAccountForm.value = JSON.stringify(accountForm)
    }
  } catch (error) {
    Message.error('获取账号信息失败')
  } finally {
    accountLoading.value = false
  }
}

async function fetchPasskeys() {
  passkeyLoading.value = true
  try {
    const res = await adminApi.getPasskeys()
    if (res.code === 0) {
      passkeyState.passwordLoginEnabled = res.data.passwordLoginEnabled
      passkeyState.passkeys = Array.isArray(res.data.passkeys) ? res.data.passkeys : []
    }
  } catch (error: any) {
    Message.error(error.message || '获取 Passkey 失败')
  } finally {
    passkeyLoading.value = false
  }
}

async function handleRegisterPasskey() {
  const unavailableReason = getPasskeyUnavailableReason()
  if (unavailableReason) {
    Message.error(unavailableReason)
    return
  }

  passkeyBusy.value = true
  try {
    const begin = await adminApi.beginPasskeyRegistration()
    const credential = await createRegistrationCredential(begin.data.publicKey)
    await adminApi.finishPasskeyRegistration({
      challengeId: begin.data.challengeId,
      credential: serializeRegistrationCredential(credential),
    })
    Message.success('Passkey 注册成功')
    await fetchPasskeys()
  } catch (error: any) {
    Message.error(error.message || 'Passkey 注册失败')
  } finally {
    passkeyBusy.value = false
  }
}

async function handlePasswordLoginChange(value: boolean | string | number) {
  const enabled = Boolean(value)
  if (!enabled && passkeyState.passkeys.length === 0) {
    Message.warning('请先注册至少一个 Passkey，再禁用密码登录')
    return
  }

  passkeyBusy.value = true
  try {
    const res = await adminApi.setPasswordLoginEnabled(enabled)
    if (res.code === 0) {
      Message.success(enabled ? '密码登录已启用' : '密码登录已禁用')
      await fetchPasskeys()
    } else {
      Message.error(res.message || '更新密码登录状态失败')
    }
  } catch (error: any) {
    Message.error(error.message || '更新密码登录状态失败')
    await fetchPasskeys()
  } finally {
    passkeyBusy.value = false
  }
}

async function handleDeletePasskey(passkey: PasskeyItem) {
  if (!passkeyState.passwordLoginEnabled && passkeyState.passkeys.length <= 1) {
    Message.warning('密码登录已禁用，不能删除最后一个 Passkey')
    return
  }

  passkeyBusy.value = true
  try {
    const res = await adminApi.deletePasskey(passkey.id)
    if (res.code === 0) {
      Message.success('Passkey 已删除')
      await fetchPasskeys()
    } else {
      Message.error(res.message || '删除 Passkey 失败')
    }
  } catch (error: any) {
    Message.error(error.message || '删除 Passkey 失败')
  } finally {
    passkeyBusy.value = false
  }
}

function formatPasskeyTime(value?: string | null) {
  if (!value) {
    return '未使用'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

watch(settingForm, (newVal) => {
  isDirty.value =
    JSON.stringify(newVal) !== initialForm.value ||
    JSON.stringify(accountForm) !== initialAccountForm.value
}, { deep: true })

watch(accountForm, (newVal) => {
  isDirty.value =
    JSON.stringify(settingForm) !== initialForm.value ||
    JSON.stringify(newVal) !== initialAccountForm.value
}, { deep: true })

async function handleSave() {
  const settingsDirty = JSON.stringify(settingForm) !== initialForm.value
  const accountDirty = JSON.stringify(accountForm) !== initialAccountForm.value

  if (!settingsDirty && !accountDirty) {
    return
  }

  if (accountDirty) {
    if (!accountForm.username) {
      Message.warning('请输入账号')
      return
    }
    if (!/^[a-zA-Z0-9]{5,12}$/.test(accountForm.username)) {
      Message.warning('账号长度需为 5-12 位字母或数字')
      return
    }
    if (accountForm.password) {
      if (accountForm.password.length < 5 || accountForm.password.length > 18) {
        Message.warning('密码长度必须在 5-18 位之间')
        return
      }
      if (!/^[a-zA-Z0-9]+$/.test(accountForm.password)) {
        Message.warning('密码仅支持字母和数字')
        return
      }
      if (accountForm.password !== accountForm.confirmPassword) {
        Message.warning('两次输入的密码不一致')
        return
      }
    }
  }

  loading.value = true
  try {
    let redisChanged = false

    if (settingsDirty) {
      redisChanged = JSON.stringify({
        redishost: settingForm.Redishost,
        redisport: settingForm.Redisport,
        redispasswd: settingForm.Redispasswd,
        redisdb: settingForm.Redisdb,
      }) !== initialRedisSignature.value

      const payload = {
        appname: settingForm.AppName,
        appurl: settingForm.AppUrl,
        httpport: settingForm.Httpport,
        expirationdate: Number(settingForm.ExpirationDate) * 60 * 1_000_000_000,
        customerservicecontact: settingForm.CustomerServiceContact,
        secretkey: settingForm.SecretKey,
        redishost: settingForm.Redishost,
        redisport: settingForm.Redisport,
        redispasswd: settingForm.Redispasswd,
        redisdb: settingForm.Redisdb,
        tgbotkey: settingForm.Tgbotkey,
        tgchatid: settingForm.Tgchatid,
        barkkey: settingForm.Barkkey,
      }
      const settingsRes = await adminApi.saveSettings(payload as any)
      if (settingsRes.code !== 0) {
        Message.error(settingsRes.message || '保存失败')
        return
      }
      await fetchSettings()
    }

    if (accountDirty) {
      const accountRes = await adminApi.updateAccount({
        username: accountForm.username,
        password: accountForm.password || undefined,
      })
      if (accountRes.code !== 0) {
        Message.error(accountRes.message || '保存失败')
        return
      }

      accountForm.password = ''
      accountForm.confirmPassword = ''

      if (accountRes.relogin) {
        Message.success(accountRes.message || '账号安全设置已更新，请重新登录')
        document.cookie = 'token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
        router.push('/login')
        return
      }

      await fetchAccount()
    }

    Message.success(
      redisChanged
        ? '保存成功，Redis 与任务队列连接已自动重载'
        : '保存成功',
    )
  } catch (error: any) {
    Message.error(error.message || '请求失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSettings()
  fetchAccount()
  fetchPasskeys()
})
</script>

<template>
  <div class="content-stack">
    <form-section-card
      title="基础配置"
      description="配置应用名称、地址及核心业务参数。"
    >
      <a-form :model="settingForm" layout="vertical">
        <div class="settings-grid settings-grid--base">
          <a-form-item field="AppName">
            <template #label>页面名称</template>
            <a-input v-model="settingForm.AppName" placeholder="例如：UPay Pro" />
          </a-form-item>
          <a-form-item field="AppUrl">
            <template #label>
              <span class="settings-label-with-tip">
                页面地址
                <a-tooltip content="用于拼接支付页地址与后台跳转链接，正式环境建议填写完整域名。">
                  <icon-question-circle />
                </a-tooltip>
              </span>
            </template>
            <a-input v-model="settingForm.AppUrl" placeholder="例如: http://pay.domain.com" />
          </a-form-item>
          <a-form-item field="Httpport" label="HTTP 监听端口">
            <a-input-number v-model="settingForm.Httpport" :min="1" :max="65535" />
          </a-form-item>
          <a-form-item field="ExpirationDate" label="订单过期时长 (分钟)">
            <a-input-number v-model="settingForm.ExpirationDate" :min="1" :max="1440" />
          </a-form-item>
          <a-form-item field="CustomerServiceContact" label="客服联系方式">
            <a-input v-model="settingForm.CustomerServiceContact" placeholder="例如：Telegram / WhatsApp / 邮箱" />
          </a-form-item>
          <a-form-item field="SecretKey">
            <template #label>
              <span class="settings-label-with-tip">
                通信密钥 (MD5 Key)
                <a-tooltip content="修改后请同步更新商户端签名配置，否则新订单请求会验签失败。">
                  <icon-question-circle />
                </a-tooltip>
              </span>
            </template>
            <div class="settings-secret-field">
              <a-input-password v-model="settingForm.SecretKey" placeholder="用于商户侧签名通信" />
              <div class="settings-secret-actions">
                <a-button type="outline" @click="handleCopySecretKey">
                  <template #icon><icon-copy /></template>
                  复制
                </a-button>
                <a-button type="outline" @click="handleGenerateSecretKey">重新生成</a-button>
              </div>
            </div>
          </a-form-item>
        </div>
      </a-form>
    </form-section-card>

    <form-section-card
      title="账号安全"
      description="当前后台仅保留单管理员入口，可在这里修改登录账号与密码。"
    >
      <a-form :model="accountForm" layout="vertical">
        <div class="settings-grid">
          <a-form-item field="username" label="登录账号">
            <a-input v-model="accountForm.username" placeholder="请输入登录账号" />
          </a-form-item>
          <a-form-item field="password" label="新密码">
            <a-input-password v-model="accountForm.password" placeholder="留空则不修改密码" />
          </a-form-item>
          <a-form-item field="confirmPassword" label="确认新密码">
            <a-input-password
              v-model="accountForm.confirmPassword"
              placeholder="如需修改密码，请再次输入"
            />
          </a-form-item>
        </div>
      </a-form>
    </form-section-card>

    <form-section-card
      title="Passkey 登录"
      description="使用设备指纹、面容或安全密钥登录后台，可在注册后关闭密码登录。"
    >
      <div class="passkey-panel" :class="{ 'is-loading': passkeyLoading }">
        <div class="passkey-policy">
          <div class="passkey-policy__copy">
            <div class="passkey-policy__title">
              <icon-safe />
              密码登录
            </div>
            <p>
              {{ passkeyState.passwordLoginEnabled ? '当前仍允许账号密码登录。' : '当前仅允许使用 Passkey 登录。' }}
            </p>
          </div>
          <a-switch
            data-testid="password-login-switch"
            :model-value="passkeyState.passwordLoginEnabled"
            :disabled="passkeyBusy || passkeyState.passkeys.length === 0"
            @change="handlePasswordLoginChange"
          />
        </div>

        <div v-if="passkeyState.passkeys.length === 0" class="passkey-empty">
          <strong>还没有 Passkey</strong>
          <span>注册一个 Passkey 后，就可以关闭密码登录。</span>
        </div>

        <div v-else class="passkey-list">
          <article
            v-for="passkey in passkeyState.passkeys"
            :key="passkey.id"
            class="passkey-item"
          >
            <div class="passkey-item__main">
              <strong>{{ passkey.deviceLabel || 'Passkey' }}</strong>
              <span>
                创建于 {{ formatPasskeyTime(passkey.createdAt) }}
                · 最近使用 {{ formatPasskeyTime(passkey.lastUsedAt) }}
              </span>
            </div>
            <div class="passkey-item__meta">
              <span v-if="passkey.transports.length">
                {{ passkey.transports.join(' / ') }}
              </span>
              <a-button
                type="text"
                status="danger"
                size="small"
                :disabled="passkeyBusy || (!passkeyState.passwordLoginEnabled && passkeyState.passkeys.length <= 1)"
                @click="handleDeletePasskey(passkey)"
              >
                删除
              </a-button>
            </div>
          </article>
        </div>

        <div class="passkey-actions">
          <a-button
            type="primary"
            :loading="passkeyBusy"
            @click="handleRegisterPasskey"
          >
            注册 Passkey
          </a-button>
          <span v-if="!supportsPasskey()" class="passkey-actions__hint">
            {{ getPasskeyUnavailableReason() }}
          </span>
        </div>
      </div>
    </form-section-card>

    <form-section-card
      title="Redis 配置"
      description="用于缓存订单、锁地址以及处理队列任务。"
    >
      <a-form :model="settingForm" layout="vertical">
        <div class="settings-warning-card">
          修改 Redis 地址、端口、密码或数据库后，系统会立即尝试重连；如果重连失败，会自动回滚到上一份可用配置。
        </div>
        <div class="settings-grid">
          <a-form-item field="Redishost" label="Redis 地址">
            <a-input v-model="settingForm.Redishost" placeholder="Docker 环境通常为 redis" />
          </a-form-item>
          <a-form-item field="Redisport" label="Redis 端口">
            <a-input-number v-model="settingForm.Redisport" :min="1" :max="65535" />
          </a-form-item>
          <a-form-item field="Redispasswd" label="Redis 密码">
            <a-input-password v-model="settingForm.Redispasswd" />
          </a-form-item>
          <a-form-item field="Redisdb" label="Redis 数据库">
            <a-input-number v-model="settingForm.Redisdb" :min="0" :max="15" />
          </a-form-item>
        </div>
      </a-form>
    </form-section-card>

    <form-section-card
      title="通知推送配置"
      description="配置 Telegram Bot 或 Bark 实时接收订单状态通知。"
    >
      <a-form :model="settingForm" layout="vertical">
        <div class="settings-grid">
          <a-form-item field="Tgbotkey" label="Telegram Bot Token">
            <a-input v-model="settingForm.Tgbotkey" placeholder="留空则不启用 Telegram 通知" />
          </a-form-item>
          <a-form-item field="Tgchatid" label="Telegram Chat ID">
            <a-input v-model="settingForm.Tgchatid" placeholder="例如：123456789" />
          </a-form-item>
          <a-form-item field="Barkkey" label="Bark Push Key">
            <a-input v-model="settingForm.Barkkey" placeholder="留空则不启用 Bark 通知" />
          </a-form-item>
        </div>
      </a-form>
    </form-section-card>

    <floating-save-bar 
      :show="isDirty" 
      :loading="loading || accountLoading" 
      @save="handleSave" 
    />
  </div>
</template>

<style scoped>
.settings-grid--base {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.settings-grid--base :deep(.arco-form-item) {
  min-width: 0;
  margin-bottom: 0;
}

.settings-grid--base :deep(.arco-form-item-control-wrapper),
.settings-grid--base :deep(.arco-form-item-control),
.settings-grid--base :deep(.arco-form-item-content) {
  width: 100%;
}

.settings-grid--base :deep(.arco-input-wrapper),
.settings-grid--base :deep(.arco-input-number),
.settings-grid--base :deep(.arco-input-password) {
  width: 100%;
}

.settings-label-with-tip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.settings-label-with-tip :deep(.arco-icon) {
  color: var(--text-tertiary);
  font-size: 14px;
  cursor: help;
}

.settings-secret-field {
  display: flex;
  align-items: center;
  gap: 10px;
}

.settings-secret-field :deep(.arco-input-wrapper) {
  flex: 1 1 auto;
}

.settings-secret-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.settings-secret-actions :deep(.arco-btn) {
  flex: 0 0 auto;
  border-radius: 12px;
}

.passkey-panel {
  display: grid;
  gap: 16px;
}

.passkey-policy {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
  border: 1px solid var(--border-soft);
  border-radius: 18px;
  background: var(--surface-secondary);
}

.passkey-policy__copy {
  min-width: 0;
}

.passkey-policy__title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 800;
}

.passkey-policy__title :deep(.arco-icon) {
  color: var(--accent);
  font-size: 18px;
}

.passkey-policy p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.passkey-empty,
.passkey-item {
  border: 1px solid var(--border-soft);
  border-radius: 16px;
  background: var(--surface-primary);
}

.passkey-empty {
  display: grid;
  gap: 4px;
  padding: 16px;
  color: var(--text-secondary);
}

.passkey-empty strong {
  color: var(--text-primary);
}

.passkey-list {
  display: grid;
  gap: 10px;
}

.passkey-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
}

.passkey-item__main {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.passkey-item__main strong {
  color: var(--text-primary);
  font-size: 14px;
}

.passkey-item__main span,
.passkey-item__meta span,
.passkey-actions__hint {
  color: var(--text-secondary);
  font-size: 12px;
}

.passkey-item__meta {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  flex: 0 0 auto;
}

.passkey-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

@media (max-width: 1200px) {
  .settings-grid--base {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .settings-grid--base {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings-secret-field {
    flex-direction: column;
    align-items: stretch;
  }

  .settings-secret-actions {
    width: 100%;
  }

  .settings-secret-actions :deep(.arco-btn) {
    flex: 1 1 0;
  }

  .passkey-policy,
  .passkey-item,
  .passkey-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .passkey-item__meta {
    justify-content: space-between;
  }
}
</style>
