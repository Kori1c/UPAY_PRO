<script setup lang="ts">
import { onMounted, onUnmounted, ref, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { adminApi, type Order } from '../../api'
import PageSectionCard from '../../components/page-section-card.vue'

const route = useRoute()
const loading = ref(false)
const autoRefresh = ref(true)
let pollTimer: number | null = null
const paymentBaseUrl = import.meta.env.DEV ? 'http://127.0.0.1:8090' : ''

const rows = ref<Order[]>([])
const total = ref(0)
const params = reactive({
  page: 1,
  limit: 10,
  search: '',
  status: '',
})

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '待支付', value: '1' },
  { label: '已支付', value: '2' },
  { label: '已过期', value: '3' },
]

const statusMeta = {
  '': { label: '全部状态', color: 'gray' as const },
  '1': { label: '待支付', color: 'blue' as const },
  '2': { label: '已支付', color: 'green' as const },
  '3': { label: '已过期', color: 'red' as const },
}

async function fetchOrders() {
  loading.value = true
  try {
    const res = await adminApi.getOrders(params)
    if (res.code === 0) {
      rows.value = res.data.orders
      total.value = res.data.total
    }
  } catch (error) {
    Message.error('获取订单列表失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  params.page = 1
  fetchOrders()
}

function handleStatusChange() {
  params.page = 1
  fetchOrders()
}

function handlePageChange(page: number) {
  params.page = page
  fetchOrders()
}

function handleOpenPayment(tradeId: string) {
  window.open(`${paymentBaseUrl}/pay/checkout-counter/${tradeId}`, '_blank')
}

function callbackStatusMeta(order: Order) {
  if (order.status !== 2) {
    return { label: '未触发', color: 'gray' as const }
  }
  if (order.call_back_confirm === 1) {
    return { label: '已确认', color: 'green' as const }
  }
  if (order.callback_num > 0) {
    return { label: '回调失败', color: 'red' as const }
  }
  return { label: '待回调', color: 'orange' as const }
}

function setupPolling() {
  if (pollTimer) clearInterval(pollTimer)
  if (autoRefresh.value) {
    pollTimer = window.setInterval(() => {
      if (!loading.value) fetchOrders()
    }, 30000)
  }
}

watch(autoRefresh, () => {
  setupPolling()
})

onMounted(() => {
  if (route.query.status) {
    params.status = route.query.status as string
  }
  fetchOrders()
  setupPolling()
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="content-stack">
    <page-section-card
      title="订单筛选与列表"
      description="统一查看 Trade ID、支付状态、回调结果与支付入口。"
    >
      <a-space direction="vertical" fill :size="18">
        <div class="page-toolbar">
          <a-input-search
            v-model="params.search"
            class="page-toolbar__search"
            placeholder="搜索 Trade ID / 商城订单号"
            @search="handleSearch"
          />
          <a-select
            v-model="params.status"
            :options="statusOptions"
            placeholder="状态筛选"
            class="page-toolbar__status"
            @change="handleStatusChange"
          />
          <a-button type="outline" :loading="loading" @click="handleSearch">刷新</a-button>

          <div class="orders-toolbar-spacer"></div>
          <a-space class="orders-toolbar-meta">
            <a-tag :color="statusMeta[params.status as keyof typeof statusMeta]?.color ?? 'gray'" bordered>
              {{ statusMeta[params.status as keyof typeof statusMeta]?.label ?? '全部状态' }}
            </a-tag>
            <span class="orders-toolbar-refresh-label">自动刷新（30s）</span>
            <a-switch v-model="autoRefresh" size="small" />
          </a-space>
        </div>

        <a-table
          :table-layout-fixed="true"
          :columns="[
            { title: 'Trade ID', dataIndex: 'trade_id', ellipsis: true, tooltip: true, width: 148 },
            { title: '商城订单号', dataIndex: 'order_id', ellipsis: true, tooltip: true, width: 164 },
            { title: '网络', dataIndex: 'type', width: 94 },
            { title: '订单金额', slotName: 'amount', width: 100, align: 'right' },
            { title: '状态', slotName: 'status', width: 82, align: 'center' },
            { title: '回调状态', slotName: 'callbackStatus', width: 88, align: 'center' },
            { title: '创建时间', slotName: 'createdAt', width: 138 },
            { title: '操作', slotName: 'actions', width: 84, align: 'center' },
          ]"
          :data="rows"
          :pagination="{
            total: total,
            current: params.page,
            pageSize: params.limit,
            showTotal: true,
          }"
          :loading="loading"
          row-key="id"
          class="data-table orders-table"
          @page-change="handlePageChange"
        >
          <template #amount="{ record }">
            <span class="orders-table__amount">{{ record.amount.toFixed(2) }} USD</span>
          </template>
          <template #status="{ record }">
            <a-tag
              class="order-status-tag"
              :color="
                record.status === 2
                  ? 'green'
                  : record.status === 3
                    ? 'red'
                    : 'blue'
              "
              bordered
            >
              {{ record.status === 2 ? '已支付' : record.status === 3 ? '已过期' : '待支付' }}
            </a-tag>
          </template>
          <template #createdAt="{ record }">
            {{ new Date(record.CreatedAt).toLocaleString() }}
          </template>
          <template #callbackStatus="{ record }">
            <a-tag
              :color="callbackStatusMeta(record).color"
              bordered
            >
              {{ callbackStatusMeta(record).label }}
            </a-tag>
          </template>
          <template #actions="{ record }">
            <a-button
              v-if="record.status === 1"
              class="orders-table__pay-button"
              type="text"
              size="small"
              @click="handleOpenPayment(record.trade_id)"
            >
              前往支付页
            </a-button>
          </template>
          <template #empty>
            <a-empty description="暂无符合条件的订单" />
          </template>
        </a-table>
      </a-space>
    </page-section-card>
  </div>
</template>

<style scoped>
.page-toolbar {
  display: grid;
  grid-template-columns: 360px 140px auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
}

.page-toolbar__search {
  width: 100%;
}

.page-toolbar__status {
  width: 100%;
}

.orders-toolbar-spacer {
  min-width: 0;
}

.orders-toolbar-meta {
  color: var(--text-secondary);
  justify-self: end;
}

.orders-toolbar-refresh-label {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.orders-table :deep(.arco-table-th) {
  white-space: nowrap;
}

.orders-table :deep(.arco-table-th),
.orders-table :deep(.arco-table-td) {
  padding-left: 10px;
  padding-right: 10px;
}

.orders-table :deep(.arco-table-td:first-child .arco-table-cell) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.orders-table :deep(.arco-table-td:nth-child(3) .arco-table-cell),
.orders-table :deep(.arco-table-td:nth-child(7) .arco-table-cell) {
  white-space: nowrap;
}

.orders-table :deep(.arco-table-td:nth-child(2) .arco-table-cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.orders-table :deep(.arco-table-content-scroll-x) {
  overflow-x: hidden;
}

.orders-table :deep(.arco-table-td:nth-child(8) .arco-table-cell) {
  white-space: nowrap;
  padding-left: 0;
  padding-right: 0;
}

.orders-table :deep(.arco-table-td:nth-child(7) .arco-table-cell) {
  font-size: 12px;
}

.orders-table :deep(.arco-table-td:nth-child(8)) {
  padding-left: 6px;
  padding-right: 6px;
}

.orders-table__pay-button {
  width: 74px;
  padding-left: 0;
  padding-right: 0;
  justify-content: center;
  font-size: 13px;
}

.orders-table__amount {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--text-primary);
}

@media (max-width: 768px) {
  .page-toolbar {
    display: flex;
    flex-wrap: wrap;
  }

  .page-toolbar__search,
  .page-toolbar__status {
    width: 100%;
  }

  .orders-toolbar-meta {
    width: 100%;
    justify-content: space-between;
  }

  .orders-table :deep(.arco-table-content-scroll-x) {
    overflow-x: auto;
  }
}
</style>
