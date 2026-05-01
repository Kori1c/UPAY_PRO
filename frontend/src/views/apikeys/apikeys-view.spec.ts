// @vitest-environment happy-dom

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ApiKeysView from './apikeys-view.vue'

const {
  messageErrorMock,
  messageSuccessMock,
  adminApiMock,
  arcoComponentMocks,
} = vi.hoisted(() => ({
  messageErrorMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  adminApiMock: {
    getApiKeys: vi.fn(),
    saveApiKeys: vi.fn(),
  },
  arcoComponentMocks: {
    Tag: {
      template: '<span><slot /></span>',
    },
    InputPassword: {
      props: ['modelValue', 'placeholder'],
      emits: ['update:modelValue'],
      template:
        '<input :value="modelValue" :placeholder="placeholder" @input="$emit(`update:modelValue`, $event.target.value)" />',
    },
  },
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

const FloatingSaveBarStub = defineComponent({
  props: ['show', 'loading'],
  emits: ['save'],
  template: '<button v-if="show" type="button" data-testid="save-api-keys" @click="$emit(`save`)">立即保存</button>',
})

const AppIconStub = defineComponent({
  template: '<span data-testid="icon" />',
})

async function flushUi() {
  await Promise.resolve()
  await nextTick()
}

function mountApiKeysView() {
  return mount(ApiKeysView, {
    global: {
      stubs: {
        FloatingSaveBar: FloatingSaveBarStub,
        AppIcon: AppIconStub,
      },
    },
  })
}

describe('api keys view smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminApiMock.getApiKeys.mockResolvedValue({
      code: 0,
      data: {
        Tronscan: 'tronscan-key',
        Trongrid: '',
        Etherscan: 'etherscan-key',
      },
    })
    adminApiMock.saveApiKeys.mockResolvedValue({ code: 0, message: 'ok' })
  })

  it('loads configured keys and renders their status summary', async () => {
    const wrapper = mountApiKeysView()
    await flushUi()
    await flushUi()

    expect(adminApiMock.getApiKeys).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('扫描源配置')
    expect(wrapper.text()).toContain('2/3 已配置')
    expect(wrapper.text()).toContain('Tronscan')
    expect(wrapper.text()).toContain('TronGrid')
    expect(wrapper.text()).toContain('Etherscan')
    expect(wrapper.text()).toContain('TRON 主扫描源')
    expect(wrapper.text()).toContain('TRON 备用扫描源')
    expect(wrapper.text()).toContain('EVM 扫描源')
    expect(wrapper.find('input[placeholder="输入 Tronscan API Key"]').element).toHaveProperty('value', 'tronscan-key')
    expect(wrapper.find('input[placeholder="输入 TronGrid API Key"]').element).toHaveProperty('value', '')
    expect(wrapper.find('input[placeholder="输入 Etherscan API Key"]').element).toHaveProperty('value', 'etherscan-key')
  })

  it('saves changed keys with backend payload shape', async () => {
    const wrapper = mountApiKeysView()
    await flushUi()
    await flushUi()

    await wrapper.find('input[placeholder="输入 TronGrid API Key"]').setValue('trongrid-key')
    await flushUi()
    await wrapper.find('[data-testid="save-api-keys"]').trigger('click')
    await flushUi()

    expect(adminApiMock.saveApiKeys).toHaveBeenCalledWith({
      tronscan: 'tronscan-key',
      trongrid: 'trongrid-key',
      etherscan: 'etherscan-key',
    })
    expect(messageSuccessMock).toHaveBeenCalledWith('配置已更新')
  })
})
