<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi, type Stats, type Order } from '../../api'
import AppIcon from '../../components/icons/app-icon.vue'
import MetricStatCard from '../../components/metric-stat-card.vue'
import PageSectionCard from '../../components/page-section-card.vue'

const router = useRouter()
let pollTimer: number | null = null

const stats = ref<Stats>({
  userCount: 0,
  successOrderCount: 0,
  pendingOrderCount: 0,
  walletCount: 0,
  todayAmount: 0,
  yesterdayAmount: 0,
  totalAmount: 0,
  todayOrderCount: 0,
  currentMonthSuccessOrderCount: 0,
})

const metrics = ref([
  { id: 'todayAmount', label: '今日收款', value: '0.00 USD', hint: '成功入账 0 笔', tone: 'success' as const, clickable: false },
  { id: 'yesterdayAmount', label: '昨日收款', value: '0.00 USD', hint: '历史入账数据', tone: 'info' as const, clickable: false },
  { id: 'totalAmount', label: '累计收款', value: '0.00 USD', hint: '系统总计流水', tone: 'success' as const, clickable: false },
  { id: 'currentMonthOrders', label: '当月成交订单', value: '0', hint: '自然月已支付订单数', tone: 'warning' as const, clickable: false },
])

const recentOrders = ref<Order[]>([])

function formatDateTime(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

async function fetchStats() {
  try {
    const res = await adminApi.getStats()
    if (res.code === 0) {
      stats.value = res.data
      metrics.value[0].value = `${res.data.todayAmount.toFixed(2)} USD`
      metrics.value[0].hint = `今日入账 ${res.data.successOrderCount} 笔`
      
      metrics.value[1].value = `${res.data.yesterdayAmount.toFixed(2)} USD`
      
      metrics.value[2].value = `${res.data.totalAmount.toFixed(2)} USD`
      
      metrics.value[3].value = res.data.currentMonthSuccessOrderCount.toString()
      metrics.value[3].hint = `${new Date().getMonth() + 1} 月已成交订单`
    }
  } catch (error) {
    console.error('Failed to fetch stats:', error)
  }
}

async function fetchRecentOrders() {
  try {
    const res = await adminApi.getOrders({ page: 1, limit: 10 })
    if (res.code === 0) {
      recentOrders.value = res.data.orders
    }
  } catch (error) {
    console.error('Failed to fetch recent orders:', error)
  }
}

async function refreshAll() {
  await Promise.all([fetchStats(), fetchRecentOrders()])
}

onMounted(() => {
  refreshAll()
  pollTimer = window.setInterval(refreshAll, 30000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="content-stack dashboard-v2">
    <!-- Metrics Row -->
    <section class="stats-grid">
      <metric-stat-card
        v-for="metric in metrics"
        :key="metric.label"
        :label="metric.label"
        :value="metric.value"
        :hint="metric.hint"
        :tone="metric.tone"
        :clickable="metric.clickable"
      />
    </section>

    <!-- Recent Activity (Full Width) -->
    <page-section-card title="最近订单" description="系统实时同步最新产生的 10 笔交易记录">
      <a-table
        :columns="[
          { title: 'Trade ID', dataIndex: 'trade_id', width: 260 },
          { title: '商品名称', dataIndex: 'order_id', width: 260, ellipsis: true, tooltip: true },
          { title: '网络', dataIndex: 'type', width: 160 },
          { title: '金额', slotName: 'amount', width: 180 },
          { title: '状态', slotName: 'status', width: 140 },
          { title: '订单创建时间', slotName: 'createdAt', width: 220 },
        ]"
        :data="recentOrders"
        :pagination="false"
        row-key="id"
        class="data-table dashboard-orders-table"
      >
        <template #amount="{ record }">
          <span class="dashboard-amount">{{ record.amount.toFixed(2) }} USD</span>
        </template>
        <template #status="{ record }">
          <a-tag
            class="order-status-tag"
            :color="record.status === 2 ? 'green' : record.status === 3 ? 'red' : 'blue'"
            bordered
          >
            {{ record.status === 2 ? '已支付' : record.status === 3 ? '已过期' : '待支付' }}
          </a-tag>
        </template>
        <template #createdAt="{ record }">
          {{ new Date(record.CreatedAt).toLocaleString() }}
        </template>
        <template #empty>
          <a-empty description="暂无近期订单" />
        </template>
      </a-table>

      <div class="dashboard-orders-mobile">
        <article
          v-for="order in recentOrders"
          :key="order.id"
          class="dashboard-order-card"
        >
          <div class="dashboard-order-card__top">
            <strong>{{ order.trade_id }}</strong>
            <a-tag
              class="order-status-tag"
              :color="order.status === 2 ? 'green' : order.status === 3 ? 'red' : 'blue'"
              bordered
            >
              {{ order.status === 2 ? '已支付' : order.status === 3 ? '已过期' : '待支付' }}
            </a-tag>
          </div>
          <div class="dashboard-order-card__name">{{ order.order_id || '-' }}</div>
          <div class="dashboard-order-card__meta">
            <span>{{ order.type }}</span>
            <strong>{{ order.amount.toFixed(2) }} USD</strong>
          </div>
          <time>{{ formatDateTime(order.CreatedAt) }}</time>
        </article>
        <a-empty v-if="recentOrders.length === 0" description="暂无近期订单" />
      </div>

      <div class="dashboard-footer">
        <a-button type="text" size="small" @click="router.push('/orders')">
          全部订单 <app-icon name="right" />
        </a-button>
      </div>
    </page-section-card>
  </div>
</template>

<style scoped>
.dashboard-v2 {
  gap: 24px;
}

.dashboard-orders-table :deep(.arco-table-th) {
  white-space: nowrap;
}

.dashboard-orders-table :deep(.arco-table-th),
.dashboard-orders-table :deep(.arco-table-td) {
  padding-left: 24px;
  padding-right: 24px;
}

.dashboard-orders-table :deep(.arco-table-td) {
  vertical-align: middle;
}

.dashboard-orders-table :deep(.arco-table-cell) {
  color: var(--text-secondary);
}

.dashboard-orders-table :deep(.arco-table-cell) strong,
.dashboard-orders-table :deep(.arco-table-cell) .dashboard-amount {
  color: var(--text-primary);
}

.dashboard-orders-table :deep(.arco-table-td:nth-child(3) .arco-table-cell),
.dashboard-orders-table :deep(.arco-table-td:nth-child(4) .arco-table-cell),
.dashboard-orders-table :deep(.arco-table-td:nth-child(6) .arco-table-cell) {
  white-space: nowrap;
}

.dashboard-orders-table :deep(.arco-table-td:nth-child(2) .arco-table-cell) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dashboard-orders-table :deep(.arco-table-td:first-child .arco-table-cell) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dashboard-orders-table :deep(.arco-table-td:nth-child(3) .arco-table-cell) {
  white-space: nowrap;
}

.dashboard-orders-table :deep(.arco-table-td:nth-child(4) .arco-table-cell) {
  font-weight: 700;
  color: var(--text-primary);
}

.dashboard-orders-table :deep(.arco-table-td:nth-child(6) .arco-table-cell) {
  overflow: hidden;
  text-overflow: ellipsis;
}

.dashboard-amount {
  display: inline-block;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.dashboard-footer {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}

.dashboard-footer :deep(.arco-btn) {
  color: var(--text-secondary);
}

.dashboard-footer :deep(.arco-btn):hover {
  color: var(--accent);
}

.dashboard-orders-mobile {
  display: none;
}

.dashboard-order-card {
  display: grid;
  gap: 8px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-soft);
}

.dashboard-order-card:first-child {
  padding-top: 0;
}

.dashboard-order-card:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.dashboard-order-card__top,
.dashboard-order-card__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.dashboard-order-card__top strong {
  min-width: 0;
  color: var(--text-primary);
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-order-card__name {
  color: var(--text-primary);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-order-card__meta {
  color: var(--text-secondary);
  font-size: 12px;
}

.dashboard-order-card__meta strong {
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.dashboard-order-card time {
  color: var(--text-tertiary);
  font-size: 12px;
}

@media (max-width: 768px) {
  .dashboard-orders-table {
    display: none;
  }

  .dashboard-orders-mobile {
    display: grid;
  }
}

</style>
