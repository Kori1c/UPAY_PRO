<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { adminApi, type ApiKey } from '../../api'
import AppIcon from '../../components/icons/app-icon.vue'
import FloatingSaveBar from '../../components/floating-save-bar.vue'

const loading = ref(false)
const isDirty = ref(false)
const initialForm = ref('')
const form = reactive<Partial<ApiKey>>({
  Tronscan: '',
  Trongrid: '',
  Etherscan: '',
})

const keyCards = computed(() => [
  {
    key: 'Tronscan',
    title: 'Tronscan',
    status: form.Tronscan ? '已配置' : '未配置',
    statusColor: form.Tronscan ? 'green' : 'orangered',
    role: 'TRON 主扫描源',
    tip: '建议优先配置个人 Key，直接影响 TRC20 订单扫描稳定性。',
    url: 'https://tronscan.org/#/developer',
    placeholder: '输入 Tronscan API Key',
  },
  {
    key: 'Trongrid',
    title: 'TronGrid',
    status: form.Trongrid ? '已配置' : '未配置',
    statusColor: form.Trongrid ? 'green' : 'arcoblue',
    role: 'TRON 备用扫描源',
    tip: '当主扫描源压力较高或限流时，用于补充查询能力。',
    url: 'https://www.trongrid.io/dashboard/keys',
    placeholder: '输入 TronGrid API Key',
  },
  {
    key: 'Etherscan',
    title: 'Etherscan',
    status: form.Etherscan ? '已配置' : '未配置',
    statusColor: form.Etherscan ? 'green' : 'gold',
    role: 'EVM 扫描源',
    tip: '用于 Ethereum、BSC、Polygon、Arbitrum 等 EVM 网络。',
    url: 'https://etherscan.io/myapikey',
    placeholder: '输入 Etherscan API Key',
  },
])

async function fetchApiKeys() {
  loading.value = true
  try {
    const res = await adminApi.getApiKeys()
    if (res.code === 0) {
      Object.assign(form, res.data)
      initialForm.value = JSON.stringify(form)
      isDirty.value = false
    }
  } catch (error) {
    Message.error('获取 API 密钥失败')
  } finally {
    loading.value = false
  }
}

watch(form, (newVal) => {
  isDirty.value = JSON.stringify(newVal) !== initialForm.value
}, { deep: true })

async function handleSave() {
  loading.value = true
  try {
    const payload = {
      tronscan: form.Tronscan,
      trongrid: form.Trongrid,
      etherscan: form.Etherscan,
    }
    const res = await adminApi.saveApiKeys(payload as any)
    if (res.code === 0) {
      Message.success('配置已更新')
      await fetchApiKeys()
    } else {
      Message.error(res.message || '保存失败')
    }
  } catch (error) {
    Message.error('请求失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchApiKeys()
})
</script>

<template>
  <div class="content-stack">
    <section class="surface-card apikey-summary">
      <div class="apikey-summary__copy">
        <strong>扫描源配置</strong>
        <span>建议正式环境统一使用自有 API Key，减少公共配额带来的限流风险。</span>
      </div>
      <a-tag :color="keyCards.every((item) => item.status === '已配置') ? 'green' : 'orange'" bordered>
        {{ keyCards.filter((item) => item.status === '已配置').length }}/{{ keyCards.length }} 已配置
      </a-tag>
    </section>

    <section class="integration-grid">
      <article class="surface-card service-card">
        <article
          v-for="item in keyCards"
          :key="item.key"
          class="service-card__item"
        >
          <div class="service-card__header">
            <div class="service-card__title">
              <strong>{{ item.title }}</strong>
              <span>{{ item.role }}</span>
            </div>
            <a-tag :color="item.statusColor" size="small" bordered>{{ item.status }}</a-tag>
          </div>
          <p class="service-card__desc">{{ item.tip }}</p>
          <div class="service-card__input">
            <label>{{ item.title }} API Key</label>
            <a-input-password
              v-if="item.key === 'Tronscan'"
              v-model="form.Tronscan"
              :placeholder="item.placeholder"
            />
            <a-input-password
              v-else-if="item.key === 'Trongrid'"
              v-model="form.Trongrid"
              :placeholder="item.placeholder"
            />
            <a-input-password
              v-else
              v-model="form.Etherscan"
              :placeholder="item.placeholder"
            />
          </div>
          <a :href="item.url" target="_blank" class="service-link">
            前往申请 <app-icon name="launch" />
          </a>
        </article>
      </article>
    </section>

    <floating-save-bar 
      :show="isDirty" 
      :loading="loading" 
      @save="handleSave" 
    />
  </div>
</template>

<style scoped>
.apikey-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
}

.apikey-summary__copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.apikey-summary__copy strong {
  font-size: 16px;
  line-height: 1.2;
}

.apikey-summary__copy span {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.integration-grid {
  margin-bottom: 32px;
}

.service-card {
  padding: 20px;
}

.service-card__item + .service-card__item {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--border-soft);
}

.service-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 10px;
}

.service-card__title {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.service-card__title strong {
  font-size: 16px;
}

.service-card__title span {
  color: var(--text-tertiary);
  font-size: 12px;
}

.service-link {
  color: var(--accent);
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
  text-decoration: none;
  font-weight: 500;
}
.service-link:hover {
  color: var(--accent-strong);
}

.service-card__desc {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
  margin: 0 0 14px;
}

.service-card__input label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-tertiary);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

@media (max-width: 768px) {
  .apikey-summary {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
