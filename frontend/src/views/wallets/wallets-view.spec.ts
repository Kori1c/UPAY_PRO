// @vitest-environment happy-dom

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import WalletsView from './wallets-view.vue'

const {
  messageErrorMock,
  messageSuccessMock,
  messageWarningMock,
  modalConfirmMock,
  adminApiMock,
  arcoComponentMocks,
} = vi.hoisted(() => ({
  messageErrorMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  messageWarningMock: vi.fn(),
  modalConfirmMock: vi.fn(),
  adminApiMock: {
    getWallets: vi.fn(),
    addWallet: vi.fn(),
    updateWallet: vi.fn(),
    deleteWallet: vi.fn(),
  },
  arcoComponentMocks: {
    Tag: { template: '<span><slot /></span>' },
    Button: { template: '<button><slot name="icon" /><slot /></button>' },
    Tooltip: { template: '<div><slot /></div>' },
    Empty: {
      props: ['description'],
      template: '<div><slot name="image" /><span>{{ description }}</span><slot /></div>',
    },
    Modal: {
      props: ['visible'],
      template: '<div v-if="visible"><slot /></div>',
    },
    Form: { template: '<form><slot /></form>' },
    FormItem: { template: '<div><slot /></div>' },
    Select: { template: '<div><slot /></div>' },
    Option: { template: '<option><slot /></option>' },
    Input: { template: '<input />' },
    InputNumber: { template: '<input />' },
    RadioGroup: { template: '<div><slot /></div>' },
    Radio: { template: '<label><slot /></label>' },
    Switch: { template: '<div><slot name="checked" /><slot name="unchecked" /></div>' },
  },
}))

vi.mock('../../api', () => ({
  adminApi: adminApiMock,
}))

vi.mock('@arco-design/web-vue', () => ({
  Message: {
    error: messageErrorMock,
    success: messageSuccessMock,
    warning: messageWarningMock,
  },
  ...arcoComponentMocks,
  Modal: {
    ...arcoComponentMocks.Modal,
    confirm: modalConfirmMock,
  },
}))

const PageSectionCardStub = defineComponent({
  template:
    '<section><header><slot name="header" /></header><slot /></section>',
})

const AppIconStub = defineComponent({
  template: '<span data-testid="icon" />',
})

async function flushUi() {
  await Promise.resolve()
  await nextTick()
}

function mountWalletsView() {
  return mount(WalletsView, {
    global: {
      stubs: {
        PageSectionCard: PageSectionCardStub,
        AppIcon: AppIconStub,
      },
    },
  })
}

describe('wallets view smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders wallet cards and total count after loading data', async () => {
    adminApiMock.getWallets.mockResolvedValueOnce({
      code: 0,
      data: [
        {
          id: 1,
          currency: 'USDT-TRC20',
          token: 'T123456789012345678901234567890123',
          status: 1,
          rate: 7.21,
          AutoRate: true,
        },
      ],
    })

    const wrapper = mountWalletsView()
    await flushUi()
    await flushUi()

    expect(wrapper.text()).toContain('共 1 个钱包')
    expect(wrapper.text()).toContain('USDT-TRC20')
    expect(wrapper.text()).toContain('运行中')
    expect(wrapper.text()).toContain('自动汇率')
  })

  it('renders the empty state when no wallets are available', async () => {
    adminApiMock.getWallets.mockResolvedValueOnce({
      code: 0,
      data: [],
    })

    const wrapper = mountWalletsView()
    await flushUi()
    await flushUi()

    expect(wrapper.text()).toContain('暂无钱包地址')
    expect(wrapper.text()).toContain('立即添加钱包')
  })
})
