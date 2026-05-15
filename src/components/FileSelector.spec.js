import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FileSelector from './FileSelector.vue'

function factory(props = {}) {
  return mount(FileSelector, {
    props: {
      selectedItems: [],
      selectedPeer: null,
      peers: [],
      sending: false,
      canSend: false,
      ...props,
    },
  })
}

const samplePeers = [
  { name: 'Alice-PC', addr: '192.168.1.100' },
  { name: 'Bob-Laptop', addr: '192.168.1.101' },
]

describe('FileSelector.vue', () => {
  describe('Rendering states', () => {
    it('shows dashed placeholder when no items selected', () => {
      const wrapper = factory({ selectedItems: [] })
      expect(wrapper.text()).toContain('Select files or folders to share')
      expect(wrapper.find('.border-dashed').exists()).toBe(true)
    })

    it('shows path list when items exist', () => {
      const wrapper = factory({
        selectedItems: [
          { path: '/home/user/test.txt', isFile: true },
          { path: '/home/user/docs', isFile: false },
        ],
      })
      const pathItems = wrapper.findAll('.bg-gray-50.rounded-lg')
      expect(pathItems.length).toBe(2)
    })

    it('renders each path text in the list', () => {
      const wrapper = factory({ selectedItems: [{ path: '/a/b.txt', isFile: true }] })
      expect(wrapper.text()).toContain('/a/b.txt')
    })

    it('does not show placeholder when items exist', () => {
      const wrapper = factory({ selectedItems: [{ path: '/test', isFile: true }] })
      expect(wrapper.text()).not.toContain('Select files or folders to share')
    })
  })

  describe('Buttons', () => {
    it('"Add Files" button emits browse-files', async () => {
      const wrapper = factory()
      const btn = wrapper.findAll('button').find((b) => b.text().includes('Add Files'))
      await btn.trigger('click')
      expect(wrapper.emitted('browse-files')).toBeTruthy()
    })

    it('"Add Folder" button emits browse-dir', async () => {
      const wrapper = factory()
      const btn = wrapper.findAll('button').find((b) => b.text().includes('Add Folder'))
      await btn.trigger('click')
      expect(wrapper.emitted('browse-dir')).toBeTruthy()
    })
  })

  describe('Path removal', () => {
    it('each path item has a remove button', () => {
      const wrapper = factory({
        selectedItems: [
          { path: '/a/b.txt', isFile: true },
          { path: '/c/d.txt', isFile: true },
        ],
      })
      const pathRows = wrapper.findAll('.bg-gray-50.rounded-lg')
      expect(pathRows.length).toBe(2)
      pathRows.forEach((row) => {
        const xBtn = row.find('button')
        expect(xBtn.exists()).toBe(true)
      })
    })

    it('clicking remove emits remove-path with correct index', async () => {
      const wrapper = factory({
        selectedItems: [
          { path: '/first', isFile: true },
          { path: '/second', isFile: true },
        ],
      })
      const removeButtons = wrapper.findAll('button.text-gray-400')
      await removeButtons[1].trigger('click')
      expect(wrapper.emitted('remove-path')[0]).toEqual([1])
    })

    it('remove buttons are disabled while sending', () => {
      const wrapper = factory({
        selectedItems: [{ path: '/test', isFile: true }],
        sending: true,
      })
      const buttons = wrapper.findAll('button')
      const removeBtn = buttons.find((b) => b.attributes('class')?.includes('hover:text-red-500'))
      expect(removeBtn.attributes('disabled')).toBeDefined()
    })
  })

  describe('Peer dropdown', () => {
    it('shows all peers as options', () => {
      const wrapper = factory({ peers: samplePeers })
      const options = wrapper.find('select').findAll('option')
      expect(options.length).toBe(3)
      expect(options[0].text()).toContain('Select peer...')
      expect(options[1].text()).toContain('Alice-PC')
      expect(options[2].text()).toContain('Bob-Laptop')
    })

    it('emits update:selectedPeer on peer selection', async () => {
      const wrapper = factory({ peers: samplePeers })
      const select = wrapper.find('select')
      await select.setValue('192.168.1.101')
      expect(wrapper.emitted('update:selectedPeer')[0]).toEqual(['192.168.1.101'])
    })

    it('shows default "Select peer..." option when no peer selected', () => {
      const wrapper = factory({ peers: samplePeers, selectedPeer: null })
      const select = wrapper.find('select')
      expect(select.element.value).toBe('')
    })

    it('reflects selectedPeer prop as the selected value', () => {
      const wrapper = factory({ peers: samplePeers, selectedPeer: '192.168.1.100' })
      const select = wrapper.find('select')
      expect(select.element.value).toBe('192.168.1.100')
    })

    it('handles empty peers list gracefully', () => {
      const wrapper = factory({ peers: [] })
      const options = wrapper.find('select').findAll('option')
      expect(options.length).toBe(1)
      expect(options[0].text()).toContain('Select peer...')
    })
  })

  describe('Send button', () => {
    it('emits send event on click', async () => {
      const wrapper = factory({ canSend: true })
      const btn = wrapper.findAll('button').find((b) => b.text().includes('Send'))
      await btn.trigger('click')
      expect(wrapper.emitted('send')).toBeTruthy()
    })

    it('is disabled when canSend is false', () => {
      const wrapper = factory({ canSend: false })
      const btn = wrapper.findAll('button').find((b) => {
        const text = b.text()
        return text.includes('Send') && !text.includes('Add')
      })
      expect(btn.attributes('disabled')).toBeDefined()
    })

    it('is enabled when canSend is true', () => {
      const wrapper = factory({ canSend: true })
      const btn = wrapper.findAll('button').find((b) => {
        const text = b.text()
        return text.includes('Send') && !text.includes('Add')
      })
      expect(btn.attributes('disabled')).toBeUndefined()
    })

    it('shows "Sending..." text when sending', () => {
      const wrapper = factory({ sending: true })
      expect(wrapper.text()).toContain('Sending...')
    })

    it('shows spinner when sending', () => {
      const wrapper = factory({ sending: true })
      expect(wrapper.find('.animate-spin').exists()).toBe(true)
    })
  })

  describe('Edge cases', () => {
    it('handles paths with special characters', () => {
      const wrapper = factory({
        selectedItems: [{ path: '/Users/name/My Documents/file (copy).txt', isFile: true }],
      })
      expect(wrapper.text()).toContain('/Users/name/My Documents/file (copy).txt')
    })

    it('renders multiple paths from long list', () => {
      const items = Array.from({ length: 20 }, (_, i) => ({ path: `/path/to/file_${i}.txt`, isFile: true }))
      const wrapper = factory({ selectedItems: items })
      expect(wrapper.findAll('.bg-gray-50.rounded-lg').length).toBe(20)
    })

    it('handles peer with missing name gracefully', () => {
      const wrapper = factory({
        peers: [{ addr: '10.0.0.1' }],
      })
      const select = wrapper.find('select')
      expect(select.text()).toContain('10.0.0.1')
    })
  })
})
