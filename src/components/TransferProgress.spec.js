// TransferProgress.vue 组件的单元测试
// 测试范围：激活状态、任务信息、进度条、当前文件、等待状态、项目列表、计算属性
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TransferProgress from './TransferProgress.vue'

function factory(props = {}) {
  return mount(TransferProgress, {
    props: {
      active: false,
      task: null,
      items: [],
      ...props,
    },
  })
}

describe('TransferProgress.vue', () => {
  describe('active prop', () => {
    it('renders nothing when not active', () => {
      const wrapper = factory({ active: false })
      expect(wrapper.html()).toBe('<!--v-if-->')
    })

    it('renders content when active', () => {
      const wrapper = factory({ active: true })
      expect(wrapper.text()).toContain('Transfer Progress')
    })
  })

  describe('Task info', () => {
    it('shows task ID and progress when task exists', () => {
      const wrapper = factory({
        active: true,
        task: { id: 'task-42', item_count: 10, total_size: 5000000 },
      })
      expect(wrapper.text()).toContain('Task: task-42')
      expect(wrapper.text()).toContain('0 / 10')
    })

    it('shows progress bar', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 5 },
        items: [
          { kind: 'file', path: '/a.txt', done: true },
          { kind: 'file', path: '/b.txt', done: true },
          { kind: 'file', path: '/c.txt', done: true },
          { kind: 'file', path: '/d', done: true },
        ],
      })
      const bar = wrapper.find('.bg-primary-600')
      expect(bar.exists()).toBe(true)
    })

    it('correctly sets progress bar width to progressPercent', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 4 },
        items: [
          { kind: 'file', path: '/a.txt', done: true },
          { kind: 'file', path: '/b.txt', done: true },
        ],
      })
      const bar = wrapper.find('.bg-primary-600')
      expect(bar.attributes('style')).toBe('width: 50%;')
    })
  })

  describe('Current file', () => {
    it('shows current file path when there is an in-progress file', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 3 },
        items: [
          { kind: 'file', path: '/a.txt', done: true },
          { kind: 'file', path: '/b.txt', done: false },
        ],
      })
      expect(wrapper.text()).toContain('Transferring:')
      expect(wrapper.text()).toContain('/b.txt')
    })

    it('shows last in-progress file when multiple are pending', () => {
      const wrapper = factory({
        active: true,
        items: [
          { kind: 'file', path: '/first.txt', done: false },
          { kind: 'file', path: '/last.txt', done: false },
        ],
      })
      expect(wrapper.text()).toContain('/last.txt')
    })

    it('does not show current file section when all files are done', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 3 },
        items: [
          { kind: 'file', path: '/a.txt', done: true },
          { kind: 'file', path: '/b.txt', done: true },
        ],
      })
      expect(wrapper.text()).not.toContain('Transferring:')
    })

    it('does not show current file when there are no items', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 3 },
        items: [],
      })
      expect(wrapper.text()).not.toContain('Transferring:')
    })
  })

  describe('Waiting state', () => {
    it('shows "Waiting for transfer to start..." when no task and no items', () => {
      const wrapper = factory({
        active: true,
        task: null,
        items: [],
      })
      expect(wrapper.text()).toContain('Waiting for transfer to start...')
    })

    it('does not show waiting text when task exists', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 1 },
        items: [],
      })
      expect(wrapper.text()).not.toContain('Waiting for transfer to start...')
    })

    it('does not show waiting text when items exist but no task', () => {
      const wrapper = factory({
        active: true,
        task: null,
        items: [{ kind: 'file', path: '/a.txt', done: false }],
      })
      expect(wrapper.text()).not.toContain('Waiting for transfer to start...')
    })
  })

  describe('Item list', () => {
    it('displays items in reverse order', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 3 },
        items: [
          { kind: 'file', path: '/first.txt', done: true },
          { kind: 'file', path: '/last.txt', done: true },
        ],
      })
      const itemDivs = wrapper.findAll('.flex.items-center.gap-2.text-xs')
      expect(itemDivs.length).toBe(2)
      expect(itemDivs[0].text()).toContain('/last.txt')
      expect(itemDivs[1].text()).toContain('/first.txt')
    })

    it('limits display to 20 most recent items', () => {
      const items = Array.from({ length: 30 }, (_, i) => ({
        kind: 'file',
        path: `/file_${i}.txt`,
        done: true,
      }))
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 30 },
        items,
      })
      const itemDivs = wrapper.findAll('.flex.items-center.gap-2.text-xs')
      expect(itemDivs.length).toBe(20)
    })

    it('shows green dot for done file items', () => {
      const wrapper = factory({
        active: true,
        items: [{ kind: 'file', path: '/a.txt', done: true }],
      })
      const dots = wrapper.findAll('.rounded-full')
      const greenDot = dots.find((d) => d.classes().includes('bg-emerald-400'))
      expect(greenDot).toBeTruthy()
    })

    it('shows blue pulsing dot for in-progress file items', () => {
      const wrapper = factory({
        active: true,
        items: [{ kind: 'file', path: '/a.txt', done: false }],
      })
      const dots = wrapper.findAll('.rounded-full')
      const blueDot = dots.find((d) => d.classes().includes('bg-blue-400') && d.classes().includes('animate-pulse'))
      expect(blueDot).toBeTruthy()
    })

    it('shows item kind label', () => {
      const wrapper = factory({
        active: true,
        items: [
          { kind: 'file', path: '/a.txt', done: true },
          { kind: 'file', path: '/b.txt', done: false },
        ],
      })
      expect(wrapper.text()).toContain('file')
    })
  })

  describe('Computed properties', () => {
    it('fileCount counts only done file items', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 5 },
        items: [
          { kind: 'file', path: '/f1.txt', done: true },
          { kind: 'file', path: '/f2.txt', done: true },
          { kind: 'file', path: '/f3.txt', done: false },
          { kind: 'file', path: '/f4.txt', done: false },
        ],
      })
      expect(wrapper.vm.fileCount).toBe(2)
    })

    it('completedItems equals fileCount', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 5 },
        items: [
          { kind: 'file', path: '/f1.txt', done: true },
          { kind: 'file', path: '/f2.txt', done: true },
          { kind: 'file', path: '/f3.txt', done: false },
        ],
      })
      expect(wrapper.vm.completedItems).toBe(2)
    })

    it('progressPercent returns 0 when totalItems is 0', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 0 },
        items: [
          { kind: 'file', path: '/f1.txt', done: true },
        ],
      })
      expect(wrapper.vm.progressPercent).toBe(0)
    })

    it('progressPercent returns 0 when task is null', () => {
      const wrapper = factory({
        active: true,
        task: null,
        items: [
          { kind: 'file', path: '/f1.txt', done: true },
          { kind: 'file', path: '/f2.txt', done: true },
          { kind: 'file', path: '/f3.txt', done: false },
        ],
      })
      expect(wrapper.vm.progressPercent).toBe(0)
    })

    it('progressPercent handles 50% progress correctly', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 6 },
        items: [
          { kind: 'file', path: '/f1.txt', done: true },
          { kind: 'file', path: '/f2.txt', done: true },
          { kind: 'file', path: '/f3.txt', done: true },
          { kind: 'file', path: '/f4.txt', done: false },
          { kind: 'file', path: '/f5.txt', done: false },
          { kind: 'file', path: '/f6.txt', done: false },
        ],
      })
      expect(wrapper.vm.progressPercent).toBe(50)
    })

    it('progressPercent handles 100% progress correctly', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 2 },
        items: [
          { kind: 'file', path: '/f1.txt', done: true },
          { kind: 'file', path: '/f2.txt', done: true },
        ],
      })
      expect(wrapper.vm.completedItems).toBe(2)
      expect(wrapper.vm.progressPercent).toBe(100)
    })

    it('totalItems defaults to 0 when task is missing item_count', () => {
      const wrapper = factory({
        active: true,
        task: {},
        items: [],
      })
      expect(wrapper.vm.totalItems).toBe(0)
    })
  })

  describe('Edge cases', () => {
    it('handles items with unknown kind gracefully (renders kind label)', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 3 },
        items: [
          { kind: 'symlink', path: '/link', done: false },
          { kind: 'file', path: '/a.txt', done: true },
        ],
      })
      expect(wrapper.text()).toContain('symlink')
    })

    it('computes progress percent based on completedItems / totalItems', () => {
      const wrapper = factory({
        active: true,
        task: { id: 't1', item_count: 2 },
        items: [
          { kind: 'file', path: '/a.txt', done: true },
          { kind: 'file', path: '/b.txt', done: true },
          { kind: 'file', path: '/c.txt', done: true },
        ],
      })
      expect(wrapper.vm.progressPercent).toBe(150)
    })

    it('renders without task with items present', () => {
      const wrapper = factory({
        active: true,
        task: null,
        items: [
          { kind: 'file', path: '/a.txt', done: true },
        ],
      })
      expect(wrapper.text()).toContain('/a.txt')
      expect(wrapper.text()).not.toContain('Waiting for transfer to start...')
    })
  })
})
