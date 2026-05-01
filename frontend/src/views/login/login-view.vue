<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconLock,
  IconSafe,
  IconUser,
} from '@arco-design/web-vue/es/icon'

import { adminApi } from '../../api'
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
                  <icon-user />
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
                  <icon-lock />
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
                <icon-safe />
              </template>
              使用 Passkey 登录
            </a-button>
          </a-form>
        </a-space>
      </div>
    </section>
  </main>
</template>
