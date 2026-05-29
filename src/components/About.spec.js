// About.vue 组件的单元测试
// 测试范围：渲染状态、版本号显示、关闭交互
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import About from './About.vue'

function factory(props = {}) {
  return mount(About, {
    props: {
      visible: false,
      updateDownloaded: false,
      updateDownloading: false,
      updateChecking: false,
      newVersion: '',
      updateError: '',
      noUpdate: false,
      ...props,
    },
    global: {
      stubs: {
        teleport: true,
      },
    },
  })
}

describe('About.vue', () => {
  describe('Rendering states', () => {
    it('does not render when visible is false', () => {
      const wrapper = factory({ visible: false })
      expect(wrapper.find('.fixed').exists()).toBe(false)
    })

    it('renders when visible is true', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.find('.fixed').exists()).toBe(true)
    })

    it('shows app name xShare', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('xShare')
    })

    it('shows app description', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('局域网文件传输工具')
    })

    it('shows version number', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('版本')
    })

    it('shows technology stack tags', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('Vue 3')
      expect(wrapper.text()).toContain('Tauri v2')
      expect(wrapper.text()).toContain('Go')
      expect(wrapper.text()).toContain('Tailwind CSS')
    })

    it('shows feature descriptions', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('快速传输')
      expect(wrapper.text()).toContain('跨平台')
      expect(wrapper.text()).toContain('安全可靠')
    })

    it('shows copyright information', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('Copyright © 2024 xShare Contributors')
    })

    it('shows close button', () => {
      const wrapper = factory({ visible: true })
      const closeBtn = wrapper.find('.btn-primary')
      expect(closeBtn.exists()).toBe(true)
      expect(closeBtn.text()).toBe('关闭')
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
  })

  describe('Update controls', () => {
    it('shows check update button by default', () => {
      const wrapper = factory({ visible: true })
      expect(wrapper.text()).toContain('检查更新')
    })

    it('emits check-update when check button is clicked', async () => {
      const wrapper = factory({ visible: true })
      const buttons = wrapper.findAll('button')
      const checkBtn = buttons.find(b => b.text() === '检查更新')
      await checkBtn.trigger('click')
      expect(wrapper.emitted('check-update')).toBeTruthy()
    })

    it('shows checking state when updateChecking is true', () => {
      const wrapper = factory({ visible: true, updateChecking: true })
      expect(wrapper.text()).toContain('检查更新中...')
      const buttons = wrapper.findAll('button')
      const checkBtn = buttons.find(b => b.text() === '检查更新')
      expect(checkBtn).toBeUndefined()
    })

    it('shows downloading state when updateDownloading is true', () => {
      const wrapper = factory({ visible: true, updateDownloading: true })
      expect(wrapper.text()).toContain('正在下载新版本...')
    })

    it('shows install button when updateDownloaded is true', () => {
      const wrapper = factory({ visible: true, updateDownloaded: true, newVersion: '1.0.0' })
      expect(wrapper.text()).toContain('新版本 v1.0.0 已就绪')
      expect(wrapper.text()).toContain('安装并重启')
    })

    it('emits install-update when install button is clicked', async () => {
      const wrapper = factory({ visible: true, updateDownloaded: true, newVersion: '1.0.0' })
      const buttons = wrapper.findAll('button')
      const installBtn = buttons.find(b => b.text() === '安装并重启')
      await installBtn.trigger('click')
      expect(wrapper.emitted('install-update')).toBeTruthy()
    })

    it('does not show check button when downloaded', () => {
      const wrapper = factory({ visible: true, updateDownloaded: true })
      const buttons = wrapper.findAll('button')
      const checkBtn = buttons.find(b => b.text() === '检查更新')
      expect(checkBtn).toBeUndefined()
    })

    it('shows error message when updateError is set', () => {
      const wrapper = factory({ visible: true, updateError: 'network error' })
      expect(wrapper.text()).toContain('检查更新失败，请稍后再试')
      const buttons = wrapper.findAll('button')
      const checkBtn = buttons.find(b => b.text() === '检查更新')
      expect(checkBtn).toBeUndefined()
    })

    it('shows no update message when noUpdate is true', () => {
      const wrapper = factory({ visible: true, noUpdate: true })
      expect(wrapper.text()).toContain('已是最新版本')
      const buttons = wrapper.findAll('button')
      const checkBtn = buttons.find(b => b.text() === '检查更新')
      expect(checkBtn).toBeUndefined()
    })

    it('shows check button again after error is cleared', async () => {
      const wrapper = factory({ visible: true, updateError: 'error' })
      expect(wrapper.text()).toContain('检查更新失败')
      await wrapper.setProps({ updateError: '' })
      expect(wrapper.text()).toContain('检查更新')
    })
  })

  describe('Modal animation', () => {
    it('has transition classes', () => {
      const wrapper = factory({ visible: true })
      const modal = wrapper.find('.fixed')
      expect(modal.classes()).toContain('z-50')
    })
  })
})
