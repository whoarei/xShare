// UpdateChecker.vue 组件的单元测试
// 测试范围：组件渲染、状态暴露、checkForUpdate 流程、错误处理、超时处理
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import UpdateChecker from './UpdateChecker.vue'

const { mockCheck } = vi.hoisted(() => ({
  mockCheck: vi.fn()
}))

vi.mock('@tauri-apps/plugin-updater', () => ({
  check: mockCheck
}))

vi.mock('@tauri-apps/plugin-process', () => ({
  relaunch: vi.fn()
}))

function factory() {
  return mount(UpdateChecker, {
    slots: {
      default: '<div class="child">child content</div>'
    }
  })
}

function createMockUpdate(overrides = {}) {
  return {
    version: '1.0.0',
    body: 'Bug fixes',
    date: '2025-01-01',
    download: vi.fn().mockResolvedValue(undefined),
    install: vi.fn().mockResolvedValue(undefined),
    ...overrides
  }
}

describe('UpdateChecker.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('renders slot content', () => {
      const wrapper = factory()
      expect(wrapper.find('.child').exists()).toBe(true)
      expect(wrapper.text()).toContain('child content')
    })

    it('is a renderless component with no visible wrapper', () => {
      const wrapper = factory()
      expect(wrapper.element.tagName).toBe('DIV')
    })
  })

  describe('Exposed state', () => {
    it('exposes reactive updateAvailable', () => {
      const wrapper = factory()
      expect(wrapper.vm.updateAvailable).toBe(false)
    })

    it('exposes reactive downloaded', () => {
      const wrapper = factory()
      expect(wrapper.vm.downloaded).toBe(false)
    })

    it('exposes reactive checking', () => {
      const wrapper = factory()
      expect(wrapper.vm.checking).toBe(false)
    })

    it('exposes reactive error', () => {
      const wrapper = factory()
      expect(wrapper.vm.error).toBe('')
    })

    it('exposes reactive noUpdate', () => {
      const wrapper = factory()
      expect(wrapper.vm.noUpdate).toBe(false)
    })

    it('exposes checkForUpdate method', () => {
      const wrapper = factory()
      expect(typeof wrapper.vm.checkForUpdate).toBe('function')
    })
  })

  describe('checkForUpdate - update available', () => {
    it('sets checking to true during check', async () => {
      let resolveCheck
      mockCheck.mockReturnValue(new Promise(r => { resolveCheck = r }))
      const wrapper = factory()

      wrapper.vm.checkForUpdate(true)
      await nextTick()

      expect(wrapper.vm.checking).toBe(true)

      resolveCheck(createMockUpdate())
      await vi.waitFor(() => expect(wrapper.vm.checking).toBe(false))
    })

    it('sets updateAvailable and newVersion when update found', async () => {
      const update = createMockUpdate({ version: '2.0.0', body: 'New features' })
      mockCheck.mockResolvedValue(update)
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(true)

      expect(wrapper.vm.updateAvailable).toBe(true)
      expect(wrapper.vm.newVersion).toBe('2.0.0')
    })

    it('calls download and sets downloaded on success', async () => {
      const update = createMockUpdate()
      mockCheck.mockResolvedValue(update)
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(true)

      expect(update.download).toHaveBeenCalledWith(expect.any(Function), { timeout: 60000 })
      expect(wrapper.vm.downloaded).toBe(true)
      expect(wrapper.vm.downloading).toBe(false)
    })

    it('sets downloading to true during download', async () => {
      let resolveDownload
      const update = createMockUpdate({
        download: vi.fn().mockReturnValue(new Promise(r => { resolveDownload = r }))
      })
      mockCheck.mockResolvedValue(update)
      const wrapper = factory()

      const promise = wrapper.vm.checkForUpdate(true)
      await nextTick()
      await nextTick()

      expect(wrapper.vm.downloading).toBe(true)

      resolveDownload()
      await promise

      expect(wrapper.vm.downloading).toBe(false)
      expect(wrapper.vm.downloaded).toBe(true)
    })

    it('skips download if already downloaded', async () => {
      const update = createMockUpdate()
      mockCheck.mockResolvedValue(update)
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(true)
      expect(update.download).toHaveBeenCalledTimes(1)

      await wrapper.vm.checkForUpdate(true)
      expect(update.download).toHaveBeenCalledTimes(1)
    })

    it('clears error and noUpdate states on new check', async () => {
      const update = createMockUpdate()
      mockCheck.mockResolvedValue(update)
      const wrapper = factory()

      wrapper.vm.error = 'previous error'
      wrapper.vm.noUpdate = true

      await wrapper.vm.checkForUpdate(true)

      expect(wrapper.vm.error).toBe('')
      expect(wrapper.vm.noUpdate).toBe(false)
    })
  })

  describe('checkForUpdate - no update', () => {
    it('sets noUpdate when check returns null', async () => {
      mockCheck.mockResolvedValue(null)
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(false)

      expect(wrapper.vm.noUpdate).toBe(true)
      expect(wrapper.vm.updateAvailable).toBe(false)
      expect(wrapper.vm.downloaded).toBe(false)
    })
  })

  describe('checkForUpdate - error handling', () => {
    it('sets error on check failure', async () => {
      mockCheck.mockRejectedValue(new Error('network error'))
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(false)

      expect(wrapper.vm.error).toBe('network error')
      expect(wrapper.vm.checking).toBe(false)
      expect(wrapper.vm.updateAvailable).toBe(false)
    })

    it('shows timeout-specific message for timeout errors', async () => {
      mockCheck.mockRejectedValue(new Error('request timed out'))
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(false)

      expect(wrapper.vm.error).toBe('网络超时，请检查网络连接或代理设置')
    })

    it('shows timeout message for "timeout" keyword', async () => {
      mockCheck.mockRejectedValue(new Error('connection timeout'))
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(false)

      expect(wrapper.vm.error).toBe('网络超时，请检查网络连接或代理设置')
    })

    it('handles non-Error rejection', async () => {
      mockCheck.mockRejectedValue('string error')
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(false)

      expect(wrapper.vm.error).toBe('string error')
    })

    it('sets checking to false even on error', async () => {
      mockCheck.mockRejectedValue(new Error('fail'))
      const wrapper = factory()

      await wrapper.vm.checkForUpdate(false)

      expect(wrapper.vm.checking).toBe(false)
    })
  })

  describe('checkForUpdate - guard conditions', () => {
    it('skips if already checking', async () => {
      let resolveCheck
      mockCheck.mockReturnValue(new Promise(r => { resolveCheck = r }))
      const wrapper = factory()

      wrapper.vm.checkForUpdate(true)
      await nextTick()
      expect(wrapper.vm.checking).toBe(true)

      await wrapper.vm.checkForUpdate(true)
      expect(mockCheck).toHaveBeenCalledTimes(1)

      resolveCheck(null)
      await vi.waitFor(() => expect(wrapper.vm.checking).toBe(false))
    })

    it('skips if currently downloading', async () => {
      let resolveDownload
      const update = createMockUpdate({
        download: vi.fn().mockReturnValue(new Promise(r => { resolveDownload = r }))
      })
      mockCheck.mockResolvedValue(update)
      const wrapper = factory()

      const promise = wrapper.vm.checkForUpdate(true)
      await nextTick()
      await nextTick()

      mockCheck.mockClear()
      await wrapper.vm.checkForUpdate(true)
      expect(mockCheck).not.toHaveBeenCalled()

      resolveDownload()
      await promise
    })
  })
})
