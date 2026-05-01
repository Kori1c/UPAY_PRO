// @vitest-environment happy-dom

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import SettingsView from './settings-view.vue'

const {
  routerPushMock,
  messageErrorMock,
  messageSuccessMock,
  messageWarningMock,
  adminApiMock,
  supportsPasskeyMock,
  unavailableReasonMock,
  arcoComponentMocks,
} = vi.hoisted(() => ({
  routerPushMock: vi.fn(),
  messageErrorMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  messageWarningMock: vi.fn(),
  adminApiMock: {
    getSettings: vi.fn(),
    saveSettings: vi.fn(),
    getAccount: vi.fn(),
    updateAccount: vi.fn(),
    getSecretKey: vi.fn(),
    getPasskeys: vi.fn(),
    beginPasskeyRegistration: vi.fn(),
    finishPasskeyRegistration: vi.fn(),
    setPasswordLoginEnabled: vi.fn(),
    deletePasskey: vi.fn(),
  },
  supportsPasskeyMock: vi.fn(),
  unavailableReasonMock: vi.fn(),
  arcoComponentMocks: {
    Form: { template: '<form><slot /></form>' },
    FormItem: {
      props: ['label'],
      template: '<label><slot name="label">{{ label }}</slot><slot /></label>',
    },
    Input: {
      props: ['modelValue', 'placeholder'],
      emits: ['update:modelValue'],
      template:
        '<input :value="modelValue" :placeholder="placeholder" @input="$emit(`update:modelValue`, $event.target.value)" />',
    },
    InputPassword: {
      props: ['modelValue', 'placeholder'],
      emits: ['update:modelValue'],
      template:
        '<input :value="modelValue" :placeholder="placeholder" @input="$emit(`update:modelValue`, $event.target.value)" />',
    },
    InputNumber: {
      props: ['modelValue'],
      emits: ['update:modelValue'],
      template:
        '<input :value="modelValue" @input="$emit(`update:modelValue`, Number($event.target.value))" />',
    },
    Tooltip: {
      props: ['content'],
      template: '<span><slot />{{ content }}</span>',
    },
    Button: {
      props: ['disabled'],
      template:
        '<button type="button" :disabled="disabled" @click="$emit(`click`)"><slot name="icon" /><slot /></button>',
    },
    Switch: {
      props: ['modelValue', 'disabled'],
      emits: ['change'],
      template:
        '<button type="button" data-testid="password-login-switch" :disabled="disabled" @click="$emit(`change`, !modelValue)"><slot /></button>',
    },
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPushMock,
  }),
}))

vi.mock('../../api', () => ({
  adminApi: adminApiMock,
}))

vi.mock('../../utils/webauthn', () => ({
  supportsPasskey: supportsPasskeyMock,
  getPasskeyUnavailableReason: unavailableReasonMock,
  createRegistrationCredential: vi.fn(),
  serializeRegistrationCredential: vi.fn(),
}))

vi.mock('@arco-design/web-vue', () => ({
  Message: {
    error: messageErrorMock,
    success: messageSuccessMock,
    warning: messageWarningMock,
  },
  ...arcoComponentMocks,
}))

const FormSectionCardStub = defineComponent({
  props: ['title', 'description'],
  template: '<section><h2>{{ title }}</h2><p>{{ description }}</p><slot /></section>',
})

const FloatingSaveBarStub = defineComponent({
  props: ['show', 'loading'],
  emits: ['save'],
  template: '<button v-if="show" type="button" data-testid="save-settings" @click="$emit(`save`)">立即保存</button>',
})

const AppIconStub = defineComponent({
  template: '<span data-testid="icon" />',
})

async function flushUi() {
  await Promise.resolve()
  await nextTick()
}

function settingsFixture(overrides = {}) {
  return {
    AppName: 'UPAY PRO',
    AppUrl: 'http://localhost',
    Httpport: 8090,
    ExpirationDate: 600_000_000_000,
    CustomerServiceContact: '@support',
    SecretKey: 'existing-secret-key',
    Redishost: 'redis',
    Redisport: 6379,
    Redispasswd: '',
    Redisdb: 0,
    Tgbotkey: '',
    Tgchatid: '',
    Barkkey: '',
    ...overrides,
  }
}

function mountSettingsView() {
  return mount(SettingsView, {
    global: {
      stubs: {
        FormSectionCard: FormSectionCardStub,
        FloatingSaveBar: FloatingSaveBarStub,
        AppIcon: AppIconStub,
      },
    },
  })
}

describe('settings view smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    supportsPasskeyMock.mockReturnValue(true)
    unavailableReasonMock.mockReturnValue('')
    adminApiMock.getSettings.mockResolvedValue({ code: 0, data: settingsFixture() })
    adminApiMock.getAccount.mockResolvedValue({
      code: 0,
      data: {
        id: 1,
        username: 'admin',
      },
    })
    adminApiMock.getPasskeys.mockResolvedValue({
      code: 0,
      data: {
        passwordLoginEnabled: true,
        passkeys: [],
      },
    })
    adminApiMock.saveSettings.mockResolvedValue({ code: 0, message: 'ok' })
    adminApiMock.updateAccount.mockResolvedValue({ code: 0, message: 'ok' })
    adminApiMock.setPasswordLoginEnabled.mockResolvedValue({ code: 0, message: 'ok' })
  })

  it('loads settings, account, and passkey state into the form', async () => {
    adminApiMock.getPasskeys.mockResolvedValueOnce({
      code: 0,
      data: {
        passwordLoginEnabled: false,
        passkeys: [
          {
            id: 1,
            credentialId: 'credential-1',
            deviceLabel: 'Mac Touch ID',
            transports: ['internal'],
            createdAt: '2026-04-23T08:00:00Z',
            lastUsedAt: null,
          },
        ],
      },
    })

    const wrapper = mountSettingsView()
    await flushUi()
    await flushUi()

    expect(adminApiMock.getSettings).toHaveBeenCalledTimes(1)
    expect(adminApiMock.getAccount).toHaveBeenCalledTimes(1)
    expect(adminApiMock.getPasskeys).toHaveBeenCalledTimes(1)
    expect(wrapper.find('input[placeholder="例如：UPay Pro"]').element).toHaveProperty('value', 'UPAY PRO')
    expect(wrapper.find('input[placeholder="例如: http://pay.domain.com"]').element).toHaveProperty(
      'value',
      window.location.origin,
    )
    expect(wrapper.text()).toContain('当前仅允许使用 Passkey 登录。')
    expect(wrapper.text()).toContain('Mac Touch ID')
  })

  it('saves changed base settings with backend payload shape', async () => {
    const wrapper = mountSettingsView()
    await flushUi()
    await flushUi()

    await wrapper.find('input[placeholder="例如：UPay Pro"]').setValue('UPAY Enterprise')
    await flushUi()
    await wrapper.find('[data-testid="save-settings"]').trigger('click')
    await flushUi()

    expect(adminApiMock.saveSettings).toHaveBeenCalledWith(expect.objectContaining({
      appname: 'UPAY Enterprise',
      appurl: window.location.origin,
      httpport: 8090,
      expirationdate: 600_000_000_000,
      redishost: 'redis',
      redisport: 6379,
    }))
    expect(adminApiMock.updateAccount).not.toHaveBeenCalled()
    expect(messageSuccessMock).toHaveBeenCalledWith('保存成功')
  })

  it('disables the password-login switch until at least one passkey exists', async () => {
    const wrapper = mountSettingsView()
    await flushUi()
    await flushUi()

    const switchButton = wrapper.find('[data-testid="password-login-switch"]')
    expect(switchButton.attributes('disabled')).toBeDefined()
    await switchButton.trigger('click')
    await flushUi()

    expect(adminApiMock.setPasswordLoginEnabled).not.toHaveBeenCalled()
  })
})
