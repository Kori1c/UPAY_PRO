// @vitest-environment happy-dom

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import OrdersView from './orders-view.vue'

const {
  routeQuery,
  messageErrorMock,
  messageSuccessMock,
  adminApiMock,
  arcoComponentMocks,
} = vi.hoisted(() => ({
  routeQuery: {} as Record<string, string>,
  messageErrorMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  adminApiMock: {
    getOrders: vi.fn(),
    retryOrderCallback: vi.fn(),
    getOrderCallbackEvents: vi.fn(),
  },
  arcoComponentMocks: {
    Space: { template: '<div><slot /></div>' },
    InputSearch: {
      props: ['modelValue', 'placeholder'],
      emits: ['update:modelValue', 'search'],
      template:
        '<input :value="modelValue" :placeholder="placeholder" @input="$emit(`update:modelValue`, $event.target.value)" @keyup.enter="$emit(`search`)" />',
    },
    Select: {
      props: ['modelValue', 'options'],
      emits: ['update:modelValue', 'change'],
      template: '<select :value="modelValue" @change="$emit(`change`, $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
    },
    Button: {
      template: '<button type="button" @click="$emit(`click`)"><slot name="icon" /><slot /></button>',
    },
    Tag: {
      template: '<span @click="$emit(`click`)"><slot /></span>',
    },
    Switch: {
      props: ['modelValue'],
      emits: ['update:modelValue'],
      template: '<button type="button" @click="$emit(`update:modelValue`, !modelValue)"><slot /></button>',
    },
    Tooltip: {
      template: '<span><slot name="content" /><slot /></span>',
    },
    Empty: {
      props: ['description'],
      template: '<div>{{ description }}</div>',
    },
    Modal: {
      props: ['visible'],
      template: '<div v-if="visible"><slot /></div>',
    },
    Table: {
      props: ['columns', 'data'],
      template: `
        <div>
          <div v-if="!data || data.length === 0"><slot name="empty" /></div>
          <div v-for="record in data" :key="record.id">
            <span v-for="column in columns" :key="column.title">
              <slot v-if="column.slotName" :name="column.slotName" :record="record" />
              <span v-else>{{ record[column.dataIndex] }}</span>
            </span>
          </div>
        </div>
      `,
    },
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery,
  }),
}))

vi.mock('../../api', () => ({
  adminApi: adminApiMock,
}))

vi.mock('@arco-design/web-vue', () => ({
  Message: {
    error: messageErrorMock,
    success: messageSuccessMock,
  },
  ...arcoComponentMocks,
}))

const PageSectionCardStub = defineComponent({
  template: '<section><header><slot name="header" /></header><slot /></section>',
})

async function flushUi() {
  await Promise.resolve()
  await nextTick()
}

function mountOrdersView() {
  return mount(OrdersView, {
    global: {
      stubs: {
        PageSectionCard: PageSectionCardStub,
      },
    },
  })
}

function orderFixture(overrides = {}) {
  return {
    id: 1,
    CreatedAt: '2026-04-23T08:00:00Z',
    trade_id: 'TRADE-1001',
    order_id: '会员套餐',
    amount: 12.34,
    actual_amount: 12.34,
    type: 'USDT-TRC20',
    token: 'T123456789012345678901234567890123',
    status: 1,
    callback_num: 0,
    call_back_confirm: 0,
    callback_state: 'pending',
    callback_state_label: '待回调',
    callback_message: '',
    last_callback_at: null,
    can_retry_callback: false,
    ...overrides,
  }
}

describe('orders view smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
    vi.spyOn(window, 'open').mockImplementation(() => null)
    vi.spyOn(window, 'setInterval').mockImplementation(
      () => 1 as unknown as ReturnType<typeof window.setInterval>,
    )
    vi.spyOn(window, 'clearInterval').mockImplementation(() => undefined)
  })

  it('loads orders with route status filter and renders key table fields', async () => {
    routeQuery.status = '2'
    adminApiMock.getOrders.mockResolvedValueOnce({
      code: 0,
      data: {
        total: 2,
        page: 1,
        limit: 10,
        orders: [
          orderFixture(),
          orderFixture({
            id: 2,
            trade_id: 'TRADE-1002',
            order_id: '续费订单',
            amount: 88,
            status: 2,
            callback_state: 'failed',
            callback_state_label: '回调失败',
            callback_message: 'notify endpoint returned 500',
            can_retry_callback: true,
          }),
        ],
      },
    })

    const wrapper = mountOrdersView()
    await flushUi()
    await flushUi()

    expect(adminApiMock.getOrders).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      limit: 10,
      status: '2',
    }))
    expect(wrapper.text()).toContain('TRADE-1001')
    expect(wrapper.text()).toContain('会员套餐')
    expect(wrapper.text()).toContain('USDT-TRC20')
    expect(wrapper.text()).toContain('12.34 USD')
    expect(wrapper.text()).toContain('待支付')
    expect(wrapper.text()).toContain('待回调')
    expect(wrapper.text()).toContain('回调失败')
    expect(wrapper.text()).toContain('查看原因')
    expect(wrapper.text()).toContain('补发回调')
  })

  it('opens the checkout page for pending orders', async () => {
    adminApiMock.getOrders.mockResolvedValueOnce({
      code: 0,
      data: {
        total: 1,
        page: 1,
        limit: 10,
        orders: [orderFixture({ trade_id: 'TRADE-PAY-1' })],
      },
    })

    const wrapper = mountOrdersView()
    await flushUi()
    await flushUi()

    await wrapper.findAll('button').find((button) => button.text() === '前往支付页')?.trigger('click')

    expect(window.open).toHaveBeenCalledWith(
      'http://127.0.0.1:8090/pay/checkout-counter/TRADE-PAY-1',
      '_blank',
    )
  })
})
