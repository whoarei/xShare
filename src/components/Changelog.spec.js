// Changelog.vue 组件的单元测试
// 测试范围：渲染状态、加载状态、关闭交互
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Changelog from './Changelog.vue'

const mockReadTextFile = vi.fn().mockResolvedValue('<h2>v0.2.0</h2><ul><li>feature A</li></ul>')

vi.mock('@tauri-apps/plugin-fs', () => ({
  readTextFile: mockReadTextFile,
  BaseDirectory: { Resource: 1 }
}))

function factory(props = {}) {
  return mount(Changelog, {
    props: {
      visible: false,
      ...props,
    },
    global: {
      stubs: {
        teleport: true,
      },
    },
  })
}

describe('Changelog.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering states', () => {
    it('does not render when visible is false', () => {
      const wrapper = factory({ visible: false })
      expect(wrapper.find('.fixed').exists()).toBe(false)
    })

    it('renders when visible is true', async () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.find('.fixed').exists()).toBe(true)
    })

    it('shows title', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('更新日志')
    })

    it('shows close button', () => {
      const wrapper = factory({ visible: true })
      const closeBtn = wrapper.find('.btn-primary')
      expect(closeBtn.exists()).toBe(true)
      expect(closeBtn.text()).toBe('关闭')
    })
  })

  describe('Content loading', () => {
    it('shows loading state initially', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.find('.animate-spin').exists()).toBe(true)
    })

    it('loads and displays HTML content', async () => {
      const wrapper = factory({ visible: true })
      await vi.dynamicImportSettled()
      await new Promise(r => setTimeout(r, 50))
      await wrapper.vm.$nextTick()
      const content = wrapper.find('.changelog-content')
      expect(content.exists()).toBe(true)
      expect(content.html()).toContain('v0.2.0')
    })
  })

  describe('Close interaction', () => {
    it('emits close event when close button is clicked', async () => {
      const wrapper = factory({ visible: true })
      const closeBtn = wrapper.find('.btn-primary')
      await closeBtn.trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
      expect(wrapper.emitted('close').length).toBe(1)
    })

    it('emits close event when clicking on backdrop', async () => {
      const wrapper = factory({ visible: true })
      const overlay = wrapper.find('.fixed.inset-0')
      await overlay.trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
    })

    it('emits close event when clicking X button', async () => {
      const wrapper = factory({ visible: true })
      const xBtn = wrapper.find('button.text-gray-400')
      await xBtn.trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
    })
  })

  describe('Resource path', () => {
    it('calls readTextFile with correct resource path', async () => {
      factory({ visible: true })
      await vi.dynamicImportSettled()
      await new Promise(r => setTimeout(r, 50))
      expect(mockReadTextFile).toHaveBeenCalledWith(
        'resources/CHANGELOG.html',
        { baseDir: 1 }
      )
    })
  })

  describe('Error handling', () => {
    it('shows error message when readTextFile fails', async () => {
      mockReadTextFile.mockRejectedValueOnce(new Error('file not found'))
      const wrapper = factory({ visible: true })
      await vi.dynamicImportSettled()
      await new Promise(r => setTimeout(r, 50))
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('无法加载更新日志')
    })

    it('does not show loading spinner after error', async () => {
      mockReadTextFile.mockRejectedValueOnce(new Error('file not found'))
      const wrapper = factory({ visible: true })
      await vi.dynamicImportSettled()
      await new Promise(r => setTimeout(r, 50))
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.animate-spin').exists()).toBe(false)
    })
  })

  describe('Caching', () => {
    it('does not call readTextFile again on subsequent renders', async () => {
      mockReadTextFile.mockClear()
      const wrapper = factory({ visible: true })
      await vi.dynamicImportSettled()
      await new Promise(r => setTimeout(r, 50))
      const callCount = mockReadTextFile.mock.calls.length

      wrapper.setProps({ visible: false })
      await wrapper.vm.$nextTick()
      wrapper.setProps({ visible: true })
      await vi.dynamicImportSettled()
      await new Promise(r => setTimeout(r, 50))
      await wrapper.vm.$nextTick()

      expect(mockReadTextFile.mock.calls.length).toBe(callCount)
    })
  })
})
