// DeviceList.vue 组件的单元测试
// 测试范围：渲染状态、发现按钮、设备列表交互
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DeviceList from './DeviceList.vue'

function factory(props = {}) {
  return mount(DeviceList, {
    props: {
      peers: [],
      discovering: false,
      selectedPeer: null,
      ...props,
    },
  })
}

describe('DeviceList.vue', () => {
  describe('Rendering states', () => {
    it('shows placeholder when no peers and not discovering', () => {
      const wrapper = factory({ peers: [], discovering: false })
      expect(wrapper.text()).toContain('No peers discovered')
      expect(wrapper.text()).toContain('Click Discover')
    })

    it('shows "Searching for peers..." when discovering and no peers', () => {
      const wrapper = factory({ peers: [], discovering: true })
      expect(wrapper.text()).toContain('Searching for peers...')
    })

    it('shows spinner on discover button while discovering', () => {
      const wrapper = factory({ discovering: true })
      expect(wrapper.find('button').text()).toContain('Scanning...')
      expect(wrapper.find('.animate-spin').exists()).toBe(true)
    })

    it('shows "Discover" text on button when not discovering', () => {
      const wrapper = factory({ discovering: false })
      const btn = wrapper.find('button')
      expect(btn.text()).toBe('Discover')
    })
  })

  describe('Discover button', () => {
    it('is disabled while discovering', () => {
      const wrapper = factory({ discovering: true })
      expect(wrapper.find('button').attributes('disabled')).toBeDefined()
    })

    it('is enabled when not discovering', () => {
      const wrapper = factory({ discovering: false })
      expect(wrapper.find('button').attributes('disabled')).toBeUndefined()
    })

    it('emits discover event on click', async () => {
      const wrapper = factory()
      await wrapper.find('button').trigger('click')
      expect(wrapper.emitted('discover')).toBeTruthy()
    })
  })

  describe('Peer list', () => {
    const samplePeers = [
      { name: 'Alice-PC', addr: '192.168.1.100' },
      { name: 'Bob-Laptop', addr: '192.168.1.101' },
    ]

    it('renders peer list when peers exist', () => {
      const wrapper = factory({ peers: samplePeers })
      const peerButtons = wrapper.findAll('button.w-full')
      expect(peerButtons.length).toBe(2)
    })

    it('shows peer name and address', () => {
      const wrapper = factory({ peers: samplePeers })
      expect(wrapper.text()).toContain('Alice-PC')
      expect(wrapper.text()).toContain('192.168.1.100')
      expect(wrapper.text()).toContain('Bob-Laptop')
      expect(wrapper.text()).toContain('192.168.1.101')
    })

    it('emits update:selectedPeer with addr on peer click', async () => {
      const wrapper = factory({ peers: samplePeers })
      const peerButtons = wrapper.findAll('button.w-full')
      await peerButtons[0].trigger('click')
      expect(wrapper.emitted('update:selectedPeer')[0]).toEqual(['192.168.1.100'])
    })

    it('highlights selected peer with primary color', () => {
      const wrapper = factory({ peers: samplePeers, selectedPeer: '192.168.1.101' })
      const peerButtons = wrapper.findAll('button.w-full')

      expect(peerButtons[0].classes()).not.toContain('bg-primary-100')
      expect(peerButtons[1].classes()).toContain('bg-primary-100')
      expect(peerButtons[1].classes()).toContain('ring-1')
    })

    it('does not highlight non-selected peers', () => {
      const wrapper = factory({ peers: samplePeers, selectedPeer: null })
      const peerButtons = wrapper.findAll('button.w-full')
      expect(peerButtons[0].classes()).toContain('hover:bg-gray-100')
      expect(peerButtons[0].classes()).not.toContain('bg-primary-100')
    })
  })

  describe('Edge cases', () => {
    it('handles peer with missing name', () => {
      const wrapper = factory({
        peers: [{ addr: '10.0.0.1' }],
      })
      expect(wrapper.text()).toContain('10.0.0.1')
    })

    it('handles peer with missing addr (uses key)', () => {
      const wrapper = factory({
        peers: [{ name: 'Ghost' }],
      })
      const buttons = wrapper.findAll('button.w-full')
      expect(buttons.length).toBe(1)
    })

    it('renders long peer names without breaking layout', () => {
      const wrapper = factory({
        peers: [{ name: 'Very-Long-Device-Name-That-Exceeds-Available-Space', addr: '192.168.1.1' }],
      })
      const nameDiv = wrapper.find('.truncate')
      expect(nameDiv.exists()).toBe(true)
    })
  })
})
