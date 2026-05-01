<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'

import { adminApi } from '../../api'
import AppIcon from '../../components/icons/app-icon.vue'
import AppThemeToggle from '../../components/app-theme-toggle.vue'
import {
  getPasskeyUnavailableReason,
  getAssertionCredential,
  serializeAssertionCredential,
} from '../../utils/webauthn'

const router = useRouter()
const loading = ref(false)
const passkeyLoading = ref(false)
const authConfigLoading = ref(false)
const authConfig = ref({
  passwordLoginEnabled: true,
  passkeySupported: true,
})

const form = reactive({
  username: '',
  password: '',
})

async function fetchAuthConfig() {
  authConfigLoading.value = true
  try {
    const res = await adminApi.getLoginAuthConfig()
    if (res.code === 0) {
      authConfig.value = res.data
    }
  } catch (error) {
    // Keep password login visible if config loading fails, so existing login remains usable.
  } finally {
    authConfigLoading.value = false
  }
}

async function handleSubmit() {
  if (!authConfig.value.passwordLoginEnabled) {
    Message.warning('密码登录已禁用，请使用 Passkey 登录')
    return
  }

  if (!form.username || !form.password) {
    Message.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    const res = await adminApi.login(form)
    if (res.code === 0) {
      Message.success('登录成功')
      router.push('/dashboard')
    } else {
      Message.error(res.message || '登录失败')
    }
  } catch (error: any) {
    // Backend might return 400 for wrong credentials, our request utility handles this
    Message.error(error.message || '登录失败')
  } finally {
    loading.value = false
  }
}

async function handlePasskeyLogin() {
  const unavailableReason = getPasskeyUnavailableReason()
  if (unavailableReason) {
    Message.error(unavailableReason)
    return
  }

  passkeyLoading.value = true
  try {
    const begin = await adminApi.beginPasskeyLogin()
    if (begin.code !== 0) {
      Message.error('获取 Passkey 登录参数失败')
      return
    }

    const credential = await getAssertionCredential(begin.data.publicKey)
    const verify = await adminApi.finishPasskeyLogin({
      challengeId: begin.data.challengeId,
      credential: serializeAssertionCredential(credential),
    })

    if (verify.code === 0) {
      Message.success('登录成功')
      router.push('/dashboard')
    } else {
      Message.error(verify.message || 'Passkey 登录失败')
    }
  } catch (error: any) {
    Message.error(error.message || 'Passkey 登录失败')
  } finally {
    passkeyLoading.value = false
  }
}

onMounted(fetchAuthConfig)
</script>

<template>
  <main class="login-shell">
    <section class="login-form-panel">
      <div class="login-form-card">
        <a-space direction="vertical" :size="24" fill>
          <div class="login-form-card__heading">
            <div class="login-form-card__theme">
              <app-theme-toggle />
            </div>
            <h2>登录 UPay Pro</h2>
            <p>安全管理钱包、订单与接口配置</p>
          </div>

          <a-form
            layout="vertical"
            :model="form"
            class="login-form"
            @submit="handleSubmit"
          >
            <a-form-item v-if="authConfig.passwordLoginEnabled" field="username" label="账号">
              <a-input
                v-model="form.username"
                class="login-form__control"
                placeholder="请输入账号"
              >
                <template #prefix>
                  <app-icon name="user" />
                </template>
              </a-input>
            </a-form-item>
            <a-form-item v-if="authConfig.passwordLoginEnabled" field="password" label="密码">
              <a-input-password
                v-model="form.password"
                class="login-form__control"
                placeholder="请输入密码"
                allow-clear
              >
                <template #prefix>
                  <app-icon name="lock" />
                </template>
              </a-input-password>
            </a-form-item>
            <div v-if="!authConfig.passwordLoginEnabled" class="login-passkey-only">
              <strong>Passkey 登录已启用</strong>
              <span>点击下方按钮，使用本机凭证完成验证。</span>
            </div>
            <a-button
              v-if="authConfig.passwordLoginEnabled"
              type="primary"
              :loading="loading || authConfigLoading"
              long
              size="large"
              class="login-submit"
              html-type="submit"
            >
              登录
            </a-button>
            <a-button
              :type="authConfig.passwordLoginEnabled ? 'outline' : 'primary'"
              :loading="passkeyLoading"
              long
              size="large"
              class="login-submit login-passkey"
              html-type="button"
              @click="handlePasskeyLogin"
            >
              <template #icon>
                <app-icon name="safe" />
              </template>
              使用 Passkey 登录
            </a-button>
          </a-form>
        </a-space>
      </div>
    </section>
  </main>
</template>

<style scoped>
.login-shell {
  display: block;
  min-height: 100dvh;
}

.login-form-panel {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 20px;
  min-height: 100dvh;
  padding: 24px 20px;
  background:
    radial-gradient(circle at top, rgba(255, 255, 255, 0.7), transparent 26%),
    linear-gradient(180deg, rgba(250, 251, 254, 0.92) 0%, rgba(244, 246, 250, 0.82) 100%);
}

.login-form-panel::before,
.login-form-panel::after {
  content: '';
  position: absolute;
  border-radius: 999px;
  filter: blur(6px);
  pointer-events: none;
}

.login-form-panel::before {
  top: 12%;
  left: calc(50% - 320px);
  width: 220px;
  height: 220px;
  background: rgba(45, 188, 176, 0.1);
}

.login-form-panel::after {
  right: calc(50% - 360px);
  bottom: 14%;
  width: 260px;
  height: 260px;
  background: rgba(79, 124, 255, 0.08);
}

html[data-theme='dark'] .login-form-panel {
  background:
    radial-gradient(circle at top, rgba(255, 255, 255, 0.05), transparent 24%),
    linear-gradient(180deg, rgba(10, 18, 31, 0.8) 0%, rgba(8, 17, 30, 0.58) 100%);
}

html[data-theme='dark'] .login-form-panel::before {
  background: rgba(45, 188, 176, 0.14);
}

html[data-theme='dark'] .login-form-panel::after {
  background: rgba(79, 124, 255, 0.12);
}

.login-form-card {
  position: relative;
  overflow: hidden;
  z-index: 1;
  width: min(100%, 440px);
  padding: 30px 28px 30px;
  border-radius: 24px;
  border: 1px solid color-mix(in srgb, var(--border-strong) 92%, transparent);
  background: rgba(255, 255, 255, 0.92);
  box-shadow:
    0 24px 60px rgba(24, 45, 79, 0.08),
    0 4px 14px rgba(24, 45, 79, 0.04);
  backdrop-filter: blur(18px);
}

.login-form-card::after {
  content: '';
  position: absolute;
  inset: 1px;
  border-radius: 23px;
  border: 1px solid rgba(255, 255, 255, 0.26);
  pointer-events: none;
}

html[data-theme='dark'] .login-form-card {
  background: rgba(11, 24, 42, 0.92);
  box-shadow:
    0 24px 60px rgba(0, 0, 0, 0.28),
    0 4px 14px rgba(0, 0, 0, 0.14);
}

.login-form-card__heading {
  position: relative;
  padding-top: 6px;
  text-align: center;
}

.login-form-card__theme {
  position: absolute;
  top: 0;
  right: 0;
}

.login-form-card__heading h2 {
  margin: 0;
  font-size: clamp(28px, 2.5vw, 34px);
  line-height: 1.15;
  text-align: center;
  color: var(--text-primary);
  font-weight: 700;
  letter-spacing: -0.02em;
}

.login-form-card__heading p {
  max-width: 280px;
  margin: 10px auto 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.login-submit {
  height: 50px;
  border-radius: 12px;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.02em;
  background: var(--accent);
  box-shadow:
    0 12px 24px rgba(20, 163, 154, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.login-passkey {
  margin-top: 12px;
  background: color-mix(in srgb, var(--surface-primary) 92%, transparent);
  box-shadow: none;
}

.login-passkey-only {
  display: grid;
  gap: 6px;
  padding: 16px;
  border: 1px solid var(--border-soft);
  border-radius: 16px;
  background: var(--surface-secondary);
  text-align: center;
}

.login-passkey-only strong {
  color: var(--text-primary);
  font-size: 15px;
}

.login-passkey-only span {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.login-submit:hover {
  transform: translateY(-1px);
  box-shadow:
    0 14px 28px rgba(20, 163, 154, 0.22),
    inset 0 1px 0 rgba(255, 255, 255, 0.22);
}

.login-submit:active {
  transform: translateY(0);
}

.login-form-card :deep(.arco-input-wrapper),
.login-form-card :deep(.arco-input-password) {
  min-height: 50px;
  border-radius: 12px;
  border-color: color-mix(in srgb, var(--border-strong) 92%, transparent);
  background: color-mix(in srgb, var(--surface-primary) 98%, transparent);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.22);
}

.login-form-card :deep(.arco-input-wrapper:hover),
.login-form-card :deep(.arco-input-password:hover) {
  border-color: color-mix(in srgb, var(--accent) 28%, var(--border-strong));
}

.login-form-card :deep(.arco-input-wrapper:focus-within),
.login-form-card :deep(.arco-input-password:focus-within) {
  border-color: color-mix(in srgb, var(--accent) 44%, var(--border-strong));
  box-shadow:
    0 0 0 4px color-mix(in srgb, var(--accent-soft) 64%, transparent),
    inset 0 1px 0 rgba(255, 255, 255, 0.24);
}

.login-form-card :deep(.arco-input-wrapper input),
.login-form-card :deep(.arco-input-password input) {
  font-size: 14px;
}

.login-form-card :deep(.arco-input-prefix) {
  color: var(--text-tertiary);
}

@media (max-width: 1120px) {
  .login-form-panel {
    min-height: auto;
    padding: 28px 20px;
  }
}

@media (max-width: 860px) {
  .login-form-card {
    width: 100%;
    max-width: 100%;
    padding: 24px 18px 22px;
  }

  .login-form-card__theme {
    position: static;
    display: flex;
    justify-content: flex-end;
    margin-bottom: 10px;
  }

  .login-form-card__heading h2 {
    font-size: 28px;
  }

  .login-form-card__heading p {
    max-width: 100%;
    font-size: 13px;
  }

  .login-submit {
    height: 48px;
    font-size: 16px;
  }

  .login-passkey {
    margin-top: 10px;
  }
}

@media (max-width: 520px) {
  .login-form-panel {
    min-height: 100dvh;
    padding:
      max(16px, env(safe-area-inset-top))
      14px
      max(16px, env(safe-area-inset-bottom));
  }

  .login-form-panel::before,
  .login-form-panel::after {
    display: none;
  }

  .login-form-card {
    border-radius: 22px;
    padding: 20px 16px 18px;
    box-shadow:
      0 14px 34px rgba(24, 45, 79, 0.08),
      0 2px 10px rgba(24, 45, 79, 0.04);
  }

  .login-form-card::after {
    border-radius: 21px;
  }

  .login-form-card__heading {
    padding-top: 0;
  }

  .login-form-card__theme {
    margin-bottom: 6px;
  }

  .login-form-card__heading h2 {
    font-size: 24px;
    line-height: 1.18;
  }

  .login-form-card__heading p {
    margin-top: 8px;
    line-height: 1.5;
  }

  .login-form-card :deep(.arco-form-item) {
    margin-bottom: 18px;
  }

  .login-form-card :deep(.arco-input-wrapper),
  .login-form-card :deep(.arco-input-password),
  .login-submit {
    min-height: 46px;
  }
}
</style>
