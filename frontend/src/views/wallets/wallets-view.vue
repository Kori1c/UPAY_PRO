<script setup lang="ts">
import { computed, onMounted, ref, reactive } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconSafe } from '@arco-design/web-vue/es/icon'
import { adminApi, type Wallet } from '../../api'
import PageSectionCard from '../../components/page-section-card.vue'

const loading = ref(false)
const saving = ref(false)
const wallets = ref<Wallet[]>([])
const modalVisible = ref(false)
const isEdit = ref(false)
const currentId = ref<number | undefined>(undefined)

const form = reactive<Wallet>({
  currency: 'USDT-TRC20',
  token: '',
  status: 1,
  rate: 7.2,
  AutoRate: false,
})

const currencies = [
  'USDT-TRC20', 'TRX', 'USDT-Polygon', 'USDT-BSC', 'USDT-ERC20', 
  'USDT-ArbitrumOne', 'USDC-ERC20', 'USDC-Polygon', 'USDC-BSC', 'USDC-ArbitrumOne'
]

const tronLikeCurrencies = new Set(['USDT-TRC20', 'TRX'])
const walletCountText = computed(() => `共 ${wallets.value.length} 个钱包`)

function resetForm() {
  Object.assign(form, {
    currency: 'USDT-TRC20',
    token: '',
    status: 1,
    rate: 7.2,
    AutoRate: false,
  })
}

function isValidWalletToken(currency: string, token: string) {
  const trimmed = token.trim()
  if (!trimmed) return false

  if (tronLikeCurrencies.has(currency)) {
    return /^T[A-Za-z0-9]{33}$/.test(trimmed)
  }

  return /^0x[a-fA-F0-9]{40}$/.test(trimmed)
}

function getWalletPlaceholder(currency: string) {
  return tronLikeCurrencies.has(currency)
    ? '请输入 TRON 钱包地址，例如 T 开头的 34 位地址'
    : '请输入 EVM 钱包地址，例如 0x 开头的 42 位地址'
}

async function fetchWallets() {
  loading.value = true
  try {
    const res = await adminApi.getWallets()
    if (res.code === 0) {
      wallets.value = res.data
    }
  } catch (error) {
    Message.error('获取钱包列表失败')
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isEdit.value = false
  currentId.value = undefined
  resetForm()
  modalVisible.value = true
}

function handleEdit(wallet: Wallet) {
  isEdit.value = true
  currentId.value = wallet.id
  Object.assign(form, { ...wallet })
  modalVisible.value = true
}

async function saveWallet() {
  form.token = form.token.trim()

  if (!isValidWalletToken(form.currency, form.token)) {
    Message.warning(
      tronLikeCurrencies.has(form.currency)
        ? '钱包地址格式不正确，请输入有效的 TRON 地址'
        : '钱包地址格式不正确，请输入有效的 EVM 地址',
    )
    return
  }

  if (!Number.isFinite(form.rate) || Number(form.rate) <= 0) {
    Message.warning('汇率必须大于 0')
    return
  }

    saving.value = true
  try {
    let res
    if (isEdit.value && currentId.value) {
      res = await adminApi.updateWallet(currentId.value, form)
    } else {
      res = await adminApi.addWallet(form)
    }
    
    if (res.code === 0) {
      Message.success('操作成功')
      await fetchWallets()
      return true
    } else {
      Message.error(res.message || '操作失败')
    }
  } catch (error) {
    Message.error('请求失败')
  } finally {
    saving.value = false
  }

  return false
}

async function handleBeforeOk() {
  const success = await saveWallet()
  if (success) {
    modalVisible.value = false
    resetForm()
  }
  return success
}

function handleDelete(id: number) {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个钱包地址吗？此操作不可恢复。',
    onOk: async () => {
      try {
        const res = await adminApi.deleteWallet(id)
        if (res.code === 0) {
          Message.success('删除成功')
          fetchWallets()
        } else {
          Message.error(res.message || '删除失败')
        }
      } catch (error) {
        Message.error('请求失败')
      }
    }
  })
}

onMounted(() => {
  fetchWallets()
})
</script>

<template>
  <div class="content-stack">
    <page-section-card title="钱包管理" description="统一维护收款地址、汇率模式与启停状态。">
      <template #header>
        <a-tag color="blue" bordered>{{ walletCountText }}</a-tag>
      </template>
      <div class="page-toolbar">
        <a-button type="primary" @click="handleAdd">添加钱包</a-button>
        <a-button type="outline" :loading="loading" @click="fetchWallets">刷新列表</a-button>
      </div>
    </page-section-card>

    <section v-if="wallets.length" class="wallet-grid">
      <article
        v-for="wallet in wallets"
        :key="wallet.id"
        class="surface-card wallet-mini-card"
      >
        <div class="wallet-mini-card__header">
          <h3 class="wallet-mini-card__title">{{ wallet.currency }}</h3>
          <div class="wallet-card__rate-inline">
            <span class="wallet-card__label">汇率</span>
            <strong>{{ wallet.rate || '--' }}</strong>
          </div>
        </div>

        <article class="wallet-card">
          <div class="wallet-card__topbar">
            <div class="wallet-card__chips">
              <a-tag :color="wallet.status === 1 ? 'green' : 'red'" bordered>
                {{ wallet.status === 1 ? '运行中' : '已停用' }}
              </a-tag>
              <a-tag :color="wallet.AutoRate ? 'blue' : 'gold'" bordered>
                {{ wallet.AutoRate ? '自动汇率' : '手动汇率' }}
              </a-tag>
            </div>
          </div>

          <div class="wallet-card__body">
            <div class="wallet-card__address-block">
              <span class="wallet-card__label">钱包地址</span>
              <a-tooltip :content="wallet.token || '-'" position="top">
                <strong class="wallet-addr">
                  {{ wallet.token && wallet.token.length > 24 ? wallet.token.slice(0, 12) + '...' + wallet.token.slice(-10) : (wallet.token || '-') }}
                </strong>
              </a-tooltip>
            </div>

            <div class="wallet-card__actions">
              <a-button type="primary" size="small" @click="handleEdit(wallet)">编辑</a-button>
              <a-button status="danger" size="small" @click="handleDelete(wallet.id!)">删除</a-button>
            </div>
          </div>
        </article>
      </article>
    </section>

    <page-section-card v-else title="钱包列表" description="当前还没有可用钱包，先添加正式收款地址再开始接单。">
      <a-empty description="暂无钱包地址">
        <template #image>
          <icon-safe class="wallet-empty-icon" />
        </template>
        <a-button type="primary" @click="handleAdd">立即添加钱包</a-button>
      </a-empty>
    </page-section-card>

    <!-- Add/Edit Modal -->
    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑钱包' : '添加钱包'"
      :ok-loading="saving"
      width="560px"
      modal-class="wallet-form-modal"
      ok-text="保存"
      cancel-text="取消"
      :on-before-ok="handleBeforeOk"
      @cancel="resetForm"
    >
      <a-form :model="form" layout="vertical">
        <a-form-item label="币种/网络">
          <a-select v-model="form.currency">
            <a-option v-for="c in currencies" :key="c" :value="c">{{ c }}</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="钱包地址">
          <a-input v-model="form.token" :placeholder="getWalletPlaceholder(form.currency)" />
        </a-form-item>
        <a-form-item label="汇率">
          <a-input-number v-model="form.rate" :precision="4" :min="0.0001" placeholder="请输入汇率" />
        </a-form-item>
        <a-form-item label="状态">
          <a-radio-group v-model="form.status">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="2">停用</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="汇率模式">
          <a-switch v-model="form.AutoRate">
            <template #checked>自动</template>
            <template #unchecked>手动</template>
          </a-switch>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style>
.wallet-empty-icon {
  font-size: 44px;
  color: var(--text-tertiary);
}

@media (max-width: 768px) {
  .wallet-form-modal {
    width: calc(100vw - 20px) !important;
    border-radius: 22px !important;
  }

  .wallet-form-modal .arco-modal-header {
    height: 58px !important;
    padding: 0 18px !important;
  }

  .wallet-form-modal .arco-modal-title {
    font-size: 17px !important;
  }

  .wallet-form-modal .arco-modal-body {
    max-height: calc(100dvh - 210px);
    padding: 20px 18px 16px !important;
    overflow-y: auto;
  }

  .wallet-form-modal .arco-modal-footer {
    position: sticky;
    bottom: 0;
    padding: 14px 18px calc(18px + env(safe-area-inset-bottom)) !important;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    background: var(--surface-primary) !important;
    border-top: 1px solid var(--border-soft) !important;
  }

  .wallet-form-modal .arco-modal-footer .arco-btn {
    width: 100%;
    height: 42px !important;
    padding: 0 12px !important;
  }

  .wallet-form-modal .arco-form-item {
    margin-bottom: 16px;
  }

  .wallet-form-modal .arco-radio-group {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
  }
}
</style>
