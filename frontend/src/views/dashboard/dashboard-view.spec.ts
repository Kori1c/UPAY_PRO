// @vitest-environment happy-dom

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import DashboardView from './dashboard-view.vue'

const {
  routerPushMock,
  adminApiMock,
  arcoComponentMocks,
} = vi.hoisted(() => ({
  routerPushMock: vi.fn(),
  adminApiMock: {
    getStats: vi.fn(),
    getOrders: vi.fn(),
  },
  arcoComponentMocks: {
    Button: {
      template: '<button type="button" @click="$emit(`click`)"><slot /></button>',
    },
    Tag: {
      template: '<span><slot /></span>',
    },
    Empty: {
      props: ['description'],
      template: '<div>{{ description }}</div>',
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
  useRouter: () => ({
    push: routerPushMock,
  }),
}))

vi.mock('../../api', () => ({
  adminApi: adminApiMock,
}))

vi.mock('@arco-design/web-vue', () => ({
  ...arcoComponentMocks,
}))

const MetricStatCardStub = defineComponent({
  props: ['label', 'value', 'hint'],
  template: '<article><span>{{ label }}</span><strong>{{ value }}</strong><small>{{ hint }}</small></article>',
})

const PageSectionCardStub = defineComponent({
  props: ['title', 'description'],
  template: '<section><h2>{{ title }}</h2><p>{{ description }}</p><slot /></section>',
})

const AppIconStub = defineComponent({
  template: '<span data-testid="icon" />',
})

async function flushUi() {
  await Promise.resolve()
  await nextTick()
}

function mountDashboardView() {
  return mount(DashboardView, {
    global: {
      stubs: {
        MetricStatCard: MetricStatCardStub,
        PageSectionCard: PageSectionCardStub,
        AppIcon: AppIconStub,
      },
    },
  })
}

describe('dashboard view smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'setInterval').mockImplementation(
      () => 1 as unknown as ReturnType<typeof window.setInterval>,
    )
    vi.spyOn(window, 'clearInterval').mockImplementation(() => undefined)

    adminApiMock.getStats.mockResolvedValue({
      code: 0,
      data: {
        userCount: 1,
        successOrderCount: 6,
        pendingOrderCount: 2,
        walletCount: 3,
        todayAmount: 123.45,
        yesterdayAmount: 67.89,
        totalAmount: 888.88,
        todayOrderCount: 4,
        currentMonthSuccessOrderCount: 19,
      },
    })
    adminApiMock.getOrders.mockResolvedValue({
      code: 0,
      data: {
        total: 1,
        page: 1,
        limit: 10,
        orders: [
          {
            id: 1,
            CreatedAt: '2026-04-23T08:00:00Z',
            trade_id: 'TRADE-DASH-1',
            order_id: '专业版套餐',
            amount: 99,
            actual_amount: 99,
            type: 'USDT-TRC20',
            token: 'T123456789012345678901234567890123',
            status: 2,
            callback_num: 1,
            call_back_confirm: 1,
            callback_state: 'confirmed',
            callback_state_label: '已确认',
            callback_message: '',
            last_callback_at: '2026-04-23T08:01:00Z',
            can_retry_callback: false,
          },
        ],
      },
    })
  })

  it('renders metrics and recent orders', async () => {
    const wrapper = mountDashboardView()
    await flushUi()
    await flushUi()

    expect(wrapper.text()).toContain('今日收款')
    expect(wrapper.text()).toContain('123.45 USD')
    expect(wrapper.text()).toContain('累计收款')
    expect(wrapper.text()).toContain('888.88 USD')
    expect(wrapper.text()).toContain('当月成交订单')
    expect(wrapper.text()).toContain('19')
    expect(wrapper.text()).not.toContain('运行提醒')
    expect(wrapper.text()).toContain('TRADE-DASH-1')
    expect(wrapper.text()).toContain('专业版套餐')
    expect(wrapper.text()).toContain('99.00 USD')
    expect(wrapper.text()).toContain('已支付')
  })

  it('navigates to the orders page from the dashboard footer', async () => {
    const wrapper = mountDashboardView()
    await flushUi()
    await flushUi()

    await wrapper.findAll('button').find((button) => button.text().includes('全部订单'))?.trigger('click')

    expect(routerPushMock).toHaveBeenCalledWith('/orders')
  })
})
