// @vitest-environment happy-dom

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import LoginView from './login-view.vue'

const {
  pushMock,
  warningMock,
  successMock,
  errorMock,
  adminApiMock,
  arcoComponentMocks,
} = vi.hoisted(() => ({
  pushMock: vi.fn(),
  warningMock: vi.fn(),
  successMock: vi.fn(),
  errorMock: vi.fn(),
  adminApiMock: {
    getLoginAuthConfig: vi.fn(),
    login: vi.fn(),
    beginPasskeyLogin: vi.fn(),
    finishPasskeyLogin: vi.fn(),
  },
  arcoComponentMocks: {
    Space: { template: '<div><slot /></div>' },
    Form: {
      emits: ['submit'],
      template: '<form @submit.prevent="$emit(`submit`)"><slot /></form>',
    },
    FormItem: {
      props: ['label'],
      template: '<label><span>{{ label }}</span><slot /></label>',
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
    Button: {
      props: ['htmlType'],
      template:
        '<button :type="htmlType === `submit` ? `submit` : `button`"><slot name="icon" /><slot /></button>',
    },
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
  }),
}))

vi.mock('../../api', () => ({
  adminApi: adminApiMock,
}))

vi.mock('../../utils/webauthn', () => ({
  getPasskeyUnavailableReason: vi.fn(() => ''),
  getAssertionCredential: vi.fn(),
  serializeAssertionCredential: vi.fn(),
}))

vi.mock('@arco-design/web-vue', () => ({
  Message: {
    warning: warningMock,
    success: successMock,
    error: errorMock,
  },
  ...arcoComponentMocks,
}))

const AppThemeToggleStub = defineComponent({
  template: '<div data-testid="theme-toggle" />',
})

const AppIconStub = defineComponent({
  template: '<span data-testid="icon" />',
})

async function flushUi() {
  await Promise.resolve()
  await nextTick()
}

function mountLoginView() {
  return mount(LoginView, {
    global: {
      stubs: {
        AppThemeToggle: AppThemeToggleStub,
        AppIcon: AppIconStub,
      },
    },
  })
}

describe('login view smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminApiMock.getLoginAuthConfig.mockResolvedValue({
      code: 0,
      data: {
        passwordLoginEnabled: true,
        passkeySupported: true,
      },
    })
    adminApiMock.login.mockResolvedValue({ code: 0, message: 'ok' })
  })

  it('renders passkey-only mode when password login is disabled', async () => {
    adminApiMock.getLoginAuthConfig.mockResolvedValueOnce({
      code: 0,
      data: {
        passwordLoginEnabled: false,
        passkeySupported: true,
      },
    })

    const wrapper = mountLoginView()
    await flushUi()
    await flushUi()

    expect(wrapper.text()).toContain('Passkey 登录已启用')
    expect(wrapper.text()).toContain('使用 Passkey 登录')
    expect(wrapper.text()).not.toContain('账号')
    expect(wrapper.text()).not.toContain('密码')
  })

  it('blocks password submit with empty credentials and shows a warning', async () => {
    const wrapper = mountLoginView()
    await flushUi()
    await flushUi()

    await wrapper.find('form').trigger('submit')

    expect(warningMock).toHaveBeenCalledWith('请输入用户名和密码')
    expect(adminApiMock.login).not.toHaveBeenCalled()
    expect(pushMock).not.toHaveBeenCalled()
  })
})
