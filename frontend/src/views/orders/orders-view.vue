<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { adminApi, type CallbackEvent, type Order } from '../../api'
import PageSectionCard from '../../components/page-section-card.vue'

const route = useRoute()
const loading = ref(false)
const autoRefresh = ref(true)
const retryingOrderId = ref<number | null>(null)
const callbackHistoryVisible = ref(false)
const callbackHistoryLoading = ref(false)
const callbackHistoryOrder = ref<Order | null>(null)
const callbackHistoryRows = ref<CallbackEvent[]>([])
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
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / params.limit)))

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

const callbackStateColorMap = {
  confirmed: 'green',
  failed: 'red',
  pending: 'orange',
  not_applicable: 'gray',
} as const

function callbackStateColor(order: Order) {
  return callbackStateColorMap[order.callback_state as keyof typeof callbackStateColorMap] ?? 'gray'
}

function canOpenCallbackHistory(order: Order) {
  return order.callback_state !== 'not_applicable' || Boolean(order.last_callback_at) || Boolean(order.callback_message)
}

function formatDateTime(value?: string | null) {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

function callbackAuditText(order: Order) {
  if (order.last_callback_at) {
    return formatDateTime(order.last_callback_at)
  }
  if (order.callback_message) {
    return '查看原因'
  }
  if (order.callback_state === 'pending') {
    return '待执行'
  }
  return '-'
}

async function openCallbackHistory(order: Order) {
  callbackHistoryVisible.value = true
  callbackHistoryOrder.value = order
  callbackHistoryLoading.value = true
  callbackHistoryRows.value = []
  try {
    const res = await adminApi.getOrderCallbackEvents(order.id)
    if (res.code === 0) {
      callbackHistoryRows.value = res.data.events ?? []
      return
    }
    Message.error('获取回调历史失败')
  } catch (error: any) {
    Message.error(error?.message || '获取回调历史失败')
  } finally {
    callbackHistoryLoading.value = false
  }
}

function closeCallbackHistory() {
  callbackHistoryVisible.value = false
  callbackHistoryOrder.value = null
  callbackHistoryRows.value = []
}

async function handleRetryCallback(order: Order) {
  retryingOrderId.value = order.id
  try {
    const res = await adminApi.retryOrderCallback(order.id)
    if (res.code === 0) {
      Message.success(res.message || '回调补发任务已触发')
      fetchOrders()
      if (callbackHistoryVisible.value && callbackHistoryOrder.value?.id === order.id) {
        openCallbackHistory(order)
      }
      return
    }
    Message.error(res.message || '回调补发失败')
  } catch (error: any) {
    Message.error(error?.message || '回调补发失败')
  } finally {
    retryingOrderId.value = null
  }
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
      description="统一查看 Trade ID、支付状态、回调结果、最近回调与支付入口。"
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
            { title: '最近回调', slotName: 'callbackMessage', width: 176 },
            { title: '创建时间', slotName: 'createdAt', width: 138 },
            { title: '操作', slotName: 'actions', width: 124, align: 'center' },
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
              :class="{ 'orders-table__history-trigger': canOpenCallbackHistory(record) }"
              :color="callbackStateColor(record)"
              bordered
              @click="canOpenCallbackHistory(record) && openCallbackHistory(record)"
            >
              {{ record.callback_state_label || '-' }}
            </a-tag>
          </template>
          <template #callbackMessage="{ record }">
            <a-tooltip
              v-if="record.last_callback_at || record.callback_message || record.callback_state === 'pending' || record.callback_state === 'confirmed'"
              position="top"
            >
              <template #content>
                <div class="orders-table__tooltip-content">
                  <span v-if="record.last_callback_at">最近回调：{{ formatDateTime(record.last_callback_at) }}</span>
                  <span v-if="record.callback_message">失败原因：{{ record.callback_message }}</span>
                  <span v-else-if="record.callback_state === 'confirmed'">最近一次回调已确认</span>
                  <span v-else-if="record.callback_state === 'pending'">订单已支付，等待首次回调</span>
                </div>
              </template>
              <button
                v-if="canOpenCallbackHistory(record)"
                type="button"
                class="orders-table__history-link"
                @click="openCallbackHistory(record)"
              >
                {{ callbackAuditText(record) }}
              </button>
              <span v-else class="orders-table__message">{{ callbackAuditText(record) }}</span>
            </a-tooltip>
            <span v-else class="orders-table__message orders-table__message--empty">{{ callbackAuditText(record) }}</span>
          </template>
          <template #actions="{ record }">
            <a-space size="mini" class="orders-table__actions">
              <a-button
                v-if="record.status === 1"
                class="orders-table__pay-button"
                type="text"
                size="small"
                @click="handleOpenPayment(record.trade_id)"
        >
                前往支付页
              </a-button>
              <a-button
                v-if="record.can_retry_callback"
                class="orders-table__retry-button"
                type="text"
                size="small"
                :loading="retryingOrderId === record.id"
                @click="handleRetryCallback(record)"
              >
                补发回调
              </a-button>
            </a-space>
          </template>
          <template #empty>
            <a-empty description="暂无符合条件的订单" />
          </template>
        </a-table>

        <div class="orders-mobile-list">
          <article
            v-for="record in rows"
            :key="record.id"
            class="orders-mobile-card"
          >
            <div class="orders-mobile-card__top">
              <strong>{{ record.trade_id }}</strong>
              <a-tag
                class="order-status-tag"
                :color="record.status === 2 ? 'green' : record.status === 3 ? 'red' : 'blue'"
                bordered
              >
                {{ record.status === 2 ? '已支付' : record.status === 3 ? '已过期' : '待支付' }}
              </a-tag>
            </div>

            <div class="orders-mobile-card__order">{{ record.order_id || '-' }}</div>

            <div class="orders-mobile-card__grid">
              <span>网络</span>
              <strong>{{ record.type || '-' }}</strong>
              <span>金额</span>
              <strong>{{ record.amount.toFixed(2) }} USD</strong>
              <span>回调</span>
              <button
                v-if="canOpenCallbackHistory(record)"
                type="button"
                class="orders-mobile-card__link"
                @click="openCallbackHistory(record)"
              >
                {{ record.callback_state_label || callbackAuditText(record) }}
              </button>
              <strong v-else>{{ record.callback_state_label || '-' }}</strong>
              <span>创建时间</span>
              <strong>{{ formatDateTime(record.CreatedAt) }}</strong>
            </div>

            <div
              v-if="record.callback_message || record.last_callback_at || record.callback_state === 'pending'"
              class="orders-mobile-card__audit"
            >
              {{ callbackAuditText(record) }}
            </div>

            <div
              v-if="record.status === 1 || record.can_retry_callback"
              class="orders-mobile-card__actions"
            >
              <a-button
                v-if="record.status === 1"
                type="outline"
                size="small"
                @click="handleOpenPayment(record.trade_id)"
              >
                前往支付页
              </a-button>
              <a-button
                v-if="record.can_retry_callback"
                type="outline"
                status="warning"
                size="small"
                :loading="retryingOrderId === record.id"
                @click="handleRetryCallback(record)"
              >
                补发回调
              </a-button>
            </div>
          </article>

          <a-empty v-if="rows.length === 0" description="暂无符合条件的订单" />

          <div v-if="total > params.limit" class="orders-mobile-pagination">
            <a-button
              size="small"
              :disabled="params.page <= 1"
              @click="handlePageChange(params.page - 1)"
            >
              上一页
            </a-button>
            <span>{{ params.page }} / {{ totalPages }}</span>
            <a-button
              size="small"
              :disabled="params.page >= totalPages"
              @click="handlePageChange(params.page + 1)"
            >
              下一页
            </a-button>
          </div>
        </div>
      </a-space>
    </page-section-card>

    <a-modal
      v-model:visible="callbackHistoryVisible"
      width="680px"
      title="回调历史"
      ok-text="关闭"
      hide-cancel
      modal-class="callback-history-modal"
      @ok="closeCallbackHistory"
      @cancel="closeCallbackHistory"
    >
      <a-space direction="vertical" fill :size="16">
        <div class="callback-history-modal__summary">
          <span>Trade ID：{{ callbackHistoryOrder?.trade_id || '-' }}</span>
          <span>回调状态：{{ callbackHistoryOrder?.callback_state_label || '-' }}</span>
        </div>

        <a-table
          :columns="[
            { title: '时间', slotName: 'createdAt', width: 180 },
            { title: '触发方式', dataIndex: 'trigger_type_label', width: 108 },
            { title: '结果', slotName: 'result', width: 108 },
            { title: '尝试次数', slotName: 'attempt', width: 92, align: 'center' },
            { title: '说明', slotName: 'message' },
          ]"
          :data="callbackHistoryRows"
          :pagination="false"
          :loading="callbackHistoryLoading"
          row-key="id"
          class="data-table callback-history-table"
        >
          <template #createdAt="{ record }">
            {{ formatDateTime(record.created_at) }}
          </template>
          <template #result="{ record }">
            <a-tag :color="record.result === 'success' ? 'green' : record.result === 'failed' ? 'red' : 'arcoblue'" bordered>
              {{ record.result_label }}
            </a-tag>
          </template>
          <template #attempt="{ record }">
            {{ record.attempt_number > 0 ? `#${record.attempt_number}` : '-' }}
          </template>
          <template #message="{ record }">
            <span class="callback-history-table__message">{{ record.message || '-' }}</span>
          </template>
          <template #empty>
            <a-empty description="暂无回调历史" />
          </template>
        </a-table>
      </a-space>
    </a-modal>
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
.orders-table :deep(.arco-table-td:nth-child(8) .arco-table-cell) {
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

.orders-table :deep(.arco-table-td:nth-child(9) .arco-table-cell) {
  white-space: nowrap;
  padding-left: 0;
  padding-right: 0;
}

.orders-table :deep(.arco-table-td:nth-child(8) .arco-table-cell) {
  font-size: 12px;
}

.orders-table :deep(.arco-table-td:nth-child(9)) {
  padding-left: 6px;
  padding-right: 6px;
}

.orders-table__actions {
  justify-content: center;
}

.orders-table__history-trigger {
  cursor: pointer;
}

.orders-table__pay-button {
  width: 74px;
  padding-left: 0;
  padding-right: 0;
  justify-content: center;
  font-size: 13px;
}

.orders-table__retry-button {
  width: 64px;
  padding-left: 0;
  padding-right: 0;
  justify-content: center;
  font-size: 13px;
}

.orders-table__message {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.orders-table__history-link {
  max-width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--accent);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
}

.orders-table__history-link:hover {
  color: var(--accent-hover);
}

.orders-table__message--empty {
  color: var(--text-quaternary);
}

.orders-table__tooltip-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 260px;
  white-space: normal;
  line-height: 1.5;
}

.callback-history-modal__summary {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 18px;
  color: var(--text-secondary);
  font-size: 13px;
}

.callback-history-table__message {
  color: var(--text-secondary);
  line-height: 1.5;
}

.orders-table__amount {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--text-primary);
}

.orders-mobile-list {
  display: none;
}

.orders-mobile-card {
  display: grid;
  gap: 10px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-soft);
}

.orders-mobile-card:first-child {
  padding-top: 0;
}

.orders-mobile-card:last-of-type {
  border-bottom: 0;
}

.orders-mobile-card__top,
.orders-mobile-card__actions,
.orders-mobile-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.orders-mobile-card__top strong {
  min-width: 0;
  color: var(--text-primary);
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.orders-mobile-card__order {
  color: var(--text-primary);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.orders-mobile-card__grid {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 8px 12px;
  font-size: 12px;
}

.orders-mobile-card__grid span {
  color: var(--text-tertiary);
}

.orders-mobile-card__grid strong,
.orders-mobile-card__link {
  min-width: 0;
  color: var(--text-secondary);
  font: inherit;
  font-weight: 600;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.orders-mobile-card__link {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--accent);
}

.orders-mobile-card__audit {
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--surface-secondary);
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.orders-mobile-card__actions {
  justify-content: flex-end;
  flex-wrap: wrap;
}

.orders-mobile-pagination {
  padding-top: 12px;
  color: var(--text-secondary);
  font-size: 12px;
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

  .orders-table {
    display: none;
  }

  .orders-mobile-list {
    display: grid;
  }

  .callback-history-modal {
    width: calc(100vw - 20px) !important;
    border-radius: 22px !important;
  }
}
</style>
