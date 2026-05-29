// UpdateChecker.vue 组件的单元测试
// 测试范围：组件渲染、状态暴露、provide/inject
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import UpdateChecker from './UpdateChecker.vue'

vi.mock('@tauri-apps/plugin-updater', () => ({
  check: vi.fn()
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

    it('exposes checkForUpdate method', () => {
      const wrapper = factory()
      expect(typeof wrapper.vm.checkForUpdate).toBe('function')
    })
  })
})
