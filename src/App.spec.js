// App.vue 组件的单元测试
// 测试范围：服务器控制、设备发现、文件选择、发送流程、事件处理、日志系统
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import App from './App.vue'

const { mockInvoke, listeners, capturedUnlisteners } = vi.hoisted(() => {
  const listeners = {}
  const capturedUnlisteners = []
  return {
    mockInvoke: vi.fn(),
    listeners,
    capturedUnlisteners,
  }
})

vi.mock('@tauri-apps/api/core', () => ({
  invoke: (...args) => mockInvoke(...args),
}))

vi.mock('@tauri-apps/api/event', () => ({
  listen: (event, callback) => {
    listeners[event] = callback
    const unlisten = () => { delete listeners[event] }
    capturedUnlisteners.push(unlisten)
    return Promise.resolve(unlisten)
  },
}))

function triggerEvent(event, payload) {
  if (listeners[event]) {
    listeners[event]({ payload })
  }
}

function getLogs(wrapper) {
  return wrapper.vm.logs
}

async function mountApp() {
  const wrapper = mount(App)
  await flushPromises()
  await nextTick()
  return wrapper
}

beforeEach(() => {
  mockInvoke.mockReset()
  mockInvoke.mockImplementation((cmd) => {
    if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
    if (cmd === 'open_file_dialog') return Promise.resolve('')
    if (cmd === 'open_dir_dialog') return Promise.resolve('')
    if (cmd === 'start_server') return Promise.resolve('ok')
    if (cmd === 'stop_server') return Promise.resolve('ok')
    if (cmd === 'discover_peers') return Promise.resolve(JSON.stringify({ type: 'peers', peers: [] }))
    if (cmd === 'send_files') return Promise.resolve('ok')
    return Promise.resolve('{}')
  })
  Object.keys(listeners).forEach((k) => delete listeners[k])
  capturedUnlisteners.length = 0
})

describe('App.vue', () => {
  describe('Server Controls', () => {
    it('renders Start Server button when server is not running', async () => {
      const wrapper = await mountApp()
      expect(wrapper.text()).toContain('Start Server')
      expect(wrapper.text()).not.toContain('Stop Server')
    })

    it('renders Stop Server button when server is running', async () => {
      const wrapper = await mountApp()
      wrapper.vm.serverRunning = true
      await nextTick()
      expect(wrapper.text()).toContain('Stop Server')
      expect(wrapper.text()).not.toContain('Start Server')
    })

    it('shows server status in header', async () => {
      const wrapper = await mountApp()
      expect(wrapper.text()).toContain('Server stopped')

      wrapper.vm.serverRunning = true
      wrapper.vm.serverPort = 8080
      await nextTick()
      expect(wrapper.text()).toContain('Listening :8080')
    })

    it('calls invoke start_server with correct args', async () => {
      const wrapper = await mountApp()
      wrapper.vm.serverPort = 7777
      wrapper.vm.serverDir = '/tmp/share'
      wrapper.vm.selectedIP = '192.168.1.10'
      await nextTick()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('Start Server'))
      await btn.trigger('click')
      await flushPromises()

      expect(mockInvoke).toHaveBeenCalledWith('start_server', {
        port: 7777,
        dir: '/tmp/share',
        ip: '192.168.1.10',
      })
    })

    it('uses default receive directory when not changed', async () => {
      const wrapper = await mountApp()
      await nextTick()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('Start Server'))
      await btn.trigger('click')
      await flushPromises()

      expect(mockInvoke).toHaveBeenCalledWith('start_server', {
        port: 9527,
        dir: './received',
        ip: null,
      })
    })

    it('sends null IP when selectedIP is empty', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedIP = ''
      await nextTick()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('Start Server'))
      await btn.trigger('click')
      await flushPromises()

      const call = mockInvoke.mock.calls.find((c) => c[0] === 'start_server')
      expect(call[1].ip).toBeNull()
    })

    it('calls invoke stop_server', async () => {
      const wrapper = await mountApp()
      wrapper.vm.serverRunning = true
      await nextTick()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('Stop Server'))
      await btn.trigger('click')
      await flushPromises()

      expect(mockInvoke).toHaveBeenCalledWith('stop_server')
    })

    it('disables port/dir inputs when server is running', async () => {
      const wrapper = await mountApp()
      wrapper.vm.serverRunning = true
      await nextTick()

      const inputs = wrapper.findAll('input')
      const portInput = inputs.find((i) => i.attributes('type') === 'number')
      expect(portInput.attributes('disabled')).toBeDefined()
    })

    it('handles start server failure', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'start_server') return Promise.reject(new Error('port in use'))
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('Start Server'))
      await btn.trigger('click')
      await flushPromises()
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Failed to start server'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.msg).toContain('port in use')
      expect(errorLog.type).toBe('error')
    })

    it('handles stop server failure', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'stop_server') return Promise.reject(new Error('not found'))
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()
      wrapper.vm.serverRunning = true
      await nextTick()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('Stop Server'))
      await btn.trigger('click')
      await flushPromises()
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Failed to stop server'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.type).toBe('error')
    })

    it('populates IP dropdown from availableIPs', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips')
          return Promise.resolve(
            JSON.stringify({
              type: 'ips',
              ips: [
                { ip: '10.0.0.1', iface: 'eth0', family: 'IPv4' },
                { ip: 'fe80::1', iface: 'lo', family: 'IPv6' },
              ],
            })
          )
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()
      await flushPromises()
      await nextTick()

      const select = wrapper.find('select')
      const options = select.findAll('option')
      expect(options.length).toBe(3)
      expect(options[1].text()).toContain('10.0.0.1')
      expect(options[1].text()).toContain('eth0')
      expect(options[2].text()).toContain('fe80::1')
    })

    it('disables IP select and inputs when server is running', async () => {
      const wrapper = await mountApp()
      wrapper.vm.serverRunning = true
      await nextTick()

      const select = wrapper.find('select')
      const inputs = wrapper.findAll('input')
      const portInput = inputs.find((i) => i.attributes('type') === 'number')

      expect(select.attributes('disabled')).toBeDefined()
      expect(portInput.attributes('disabled')).toBeDefined()
    })
  })

  describe('Peer Discovery', () => {
    it('calls invoke discover_peers', async () => {
      const wrapper = await mountApp()
      const deviceList = wrapper.findComponent({ name: 'DeviceList' })
      await deviceList.vm.$emit('discover')
      await flushPromises()

      expect(mockInvoke).toHaveBeenCalledWith('discover_peers', { timeout: 5, ip: null })
    })

    it('updates peers on successful discovery', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'discover_peers')
          return Promise.resolve(
            JSON.stringify({
              type: 'peers',
              peers: [
                { name: 'Alice-PC', addr: '192.168.1.100' },
                { name: 'Bob-Laptop', addr: '192.168.1.101' },
              ],
            })
          )
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const deviceList = wrapper.findComponent({ name: 'DeviceList' })
      await deviceList.vm.$emit('discover')
      await flushPromises()
      await nextTick()

      expect(wrapper.vm.peers).toHaveLength(2)
      expect(wrapper.vm.peers[0].name).toBe('Alice-PC')
      expect(wrapper.vm.peers[1].addr).toBe('192.168.1.101')

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Found 2 peer')).type).toBe('success')
    })

    it('handles empty peers response', async () => {
      const wrapper = await mountApp()

      const deviceList = wrapper.findComponent({ name: 'DeviceList' })
      await deviceList.vm.$emit('discover')
      await flushPromises()
      await nextTick()

      expect(wrapper.vm.peers).toHaveLength(0)
    })

    it('handles discovery failure', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'discover_peers') return Promise.reject(new Error('network error'))
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const deviceList = wrapper.findComponent({ name: 'DeviceList' })
      await deviceList.vm.$emit('discover')
      await flushPromises()
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Discovery failed'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.type).toBe('error')
      expect(wrapper.vm.discovering).toBe(false)
    })

    it('transitions discovering state', async () => {
      let resolveDiscover
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'discover_peers')
          return new Promise((resolve) => {
            resolveDiscover = resolve
          })
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const deviceList = wrapper.findComponent({ name: 'DeviceList' })
      deviceList.vm.$emit('discover')
      await nextTick()

      expect(wrapper.vm.discovering).toBe(true)

      resolveDiscover(JSON.stringify({ type: 'peers', peers: [] }))
      await flushPromises()
      await nextTick()

      expect(wrapper.vm.discovering).toBe(false)
    })
  })

  describe('File Selection', () => {
    it('calls invoke open_file_dialog', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'open_file_dialog') return Promise.resolve('/home/user/test.txt')
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('browse-files')
      await flushPromises()
      await nextTick()

      expect(mockInvoke).toHaveBeenCalledWith('open_file_dialog')
      expect(wrapper.vm.selectedItems.some((i) => i.path === '/home/user/test.txt' && i.isFile)).toBe(true)
    })

    it('calls invoke open_dir_dialog', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'open_dir_dialog') return Promise.resolve('/home/user/docs')
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('browse-dir')
      await flushPromises()
      await nextTick()

      expect(mockInvoke).toHaveBeenCalledWith('open_dir_dialog')
      expect(wrapper.vm.selectedItems.some((i) => i.path === '/home/user/docs' && !i.isFile)).toBe(true)
    })

    it('does not add duplicate paths', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'open_file_dialog') return Promise.resolve('/home/user/test.txt')
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()
      wrapper.vm.selectedItems = [{ path: '/home/user/test.txt', isFile: true }]
      await nextTick()

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('browse-files')
      await flushPromises()
      await nextTick()

      expect(wrapper.vm.selectedItems.length).toBe(1)
    })

    it('removes path by index', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedItems = [
        { path: '/a/b.txt', isFile: true },
        { path: '/c/d.txt', isFile: true },
      ]
      await nextTick()

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('remove-path', 0)
      await nextTick()

      expect(wrapper.vm.selectedItems).toEqual([{ path: '/c/d.txt', isFile: true }])
    })

    it('handles file dialog error', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'open_file_dialog') return Promise.reject(new Error('cancelled'))
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('browse-files')
      await flushPromises()
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('File dialog error'))
      expect(errorLog).toBeTruthy()
    })
  })

  describe('Send Flow', () => {
    it('guards against sending with no peer', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      wrapper.vm.selectedPeer = null
      await nextTick()

      await wrapper.vm.sendFiles()
      await flushPromises()

      expect(mockInvoke).not.toHaveBeenCalledWith('send_files', expect.anything())
    })

    it('guards against sending with no paths', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedItems = []
      wrapper.vm.selectedPeer = '192.168.1.100'
      await nextTick()

      await wrapper.vm.sendFiles()
      await flushPromises()

      expect(mockInvoke).not.toHaveBeenCalledWith('send_files', expect.anything())
    })

    it('calls invoke send_files for each path', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedPeer = '192.168.1.100'
      wrapper.vm.selectedItems = [
        { path: '/a/b', isFile: true },
        { path: '/c/d', isFile: false },
      ]
      await nextTick()

      mockInvoke.mockClear()
      await wrapper.vm.sendFiles()
      await flushPromises()

      expect(mockInvoke).toHaveBeenCalledTimes(2)
      expect(mockInvoke).toHaveBeenCalledWith('send_files', { peer: '192.168.1.100', path: '/a/b' })
      expect(mockInvoke).toHaveBeenCalledWith('send_files', { peer: '192.168.1.100', path: '/c/d' })
    })

    it('handles send failure', async () => {
      mockInvoke.mockImplementation((cmd) => {
        if (cmd === 'list_ips') return Promise.resolve(JSON.stringify({ type: 'ips', ips: [] }))
        if (cmd === 'send_files') return Promise.reject(new Error('connection refused'))
        return Promise.resolve('{}')
      })
      const wrapper = await mountApp()
      wrapper.vm.selectedPeer = '192.168.1.100'
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      await nextTick()

      await wrapper.vm.sendFiles()
      await flushPromises()
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Send error'))
      expect(errorLog).toBeTruthy()
      expect(wrapper.vm.sending).toBe(false)
      expect(wrapper.vm.transferActive).toBe(false)
    })
  })

  describe('Event Handling: transfer-progress', () => {
    it('task event sets currentTask and logs formatted size', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't1', item_count: 3, total_size: 1048576 }))
      await nextTick()

      expect(wrapper.vm.currentTask).toBeTruthy()
      expect(wrapper.vm.currentTask.id).toBe('t1')

      const logs = getLogs(wrapper)
      const taskLog = logs.find((l) => l.msg.includes('Receiving task'))
      expect(taskLog).toBeTruthy()
      expect(taskLog.msg).toContain('1.0 MB')
    })

    it('progress event pushes item to progressItems', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'progress', kind: 'file', path: '/a.txt', done: false }))
      await nextTick()

      expect(wrapper.vm.progressItems).toHaveLength(1)
      expect(wrapper.vm.progressItems[0].path).toBe('/a.txt')
    })

    it('complete event resets state', async () => {
      const wrapper = await mountApp()
      wrapper.vm.transferActive = true
      wrapper.vm.sending = true
      await nextTick()

      triggerEvent('transfer-progress', JSON.stringify({ type: 'complete' }))
      await nextTick()

      expect(wrapper.vm.transferActive).toBe(false)
      expect(wrapper.vm.sending).toBe(false)

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Transfer complete!'))).toBeTruthy()
    })

    it('error event logs error and resets state', async () => {
      const wrapper = await mountApp()
      wrapper.vm.transferActive = true
      wrapper.vm.sending = true
      await nextTick()

      triggerEvent('transfer-progress', JSON.stringify({ type: 'error', msg: 'disk full' }))
      await nextTick()

      expect(wrapper.vm.transferActive).toBe(false)
      expect(wrapper.vm.sending).toBe(false)

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Transfer error'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.type).toBe('error')
    })

    it('malformed JSON is silently ignored', async () => {
      const wrapper = await mountApp()
      wrapper.vm.transferActive = true
      wrapper.vm.sending = true
      await nextTick()

      triggerEvent('transfer-progress', '{bad json')
      await nextTick()

      expect(wrapper.vm.transferActive).toBe(true)
      expect(wrapper.vm.sending).toBe(true)
    })
  })

  describe('Event Handling: server-event', () => {
    it('ready event sets serverRunning', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', JSON.stringify({ type: 'ready', port: 9527, dir: '/tmp' }))
      await nextTick()

      expect(wrapper.vm.serverRunning).toBe(true)

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Server ready on port 9527'))).toBeTruthy()
    })

    it('task event sets currentTask and transferActive', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', JSON.stringify({ type: 'task', id: 'recv-1', item_count: 10, total_size: 5000 }))
      await nextTick()

      expect(wrapper.vm.currentTask).toBeTruthy()
      expect(wrapper.vm.transferActive).toBe(true)
    })

    it('progress event pushes to progressItems', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', JSON.stringify({ type: 'progress', kind: 'file', path: '/out', done: true }))
      await nextTick()

      expect(wrapper.vm.progressItems).toHaveLength(1)
    })

    it('complete event logs and resets transferActive', async () => {
      const wrapper = await mountApp()
      wrapper.vm.transferActive = true
      await nextTick()

      triggerEvent('server-event', JSON.stringify({ type: 'complete' }))
      await nextTick()

      expect(wrapper.vm.transferActive).toBe(false)
      expect(getLogs(wrapper).find((l) => l.msg.includes('Receive complete!'))).toBeTruthy()
    })

    it('error event logs error', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', JSON.stringify({ type: 'error', msg: 'disk full' }))
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Server error'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.type).toBe('error')
    })

    it('raw text is appended to serverOutput', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', 'some raw output line')
      await nextTick()

      expect(wrapper.vm.serverOutput).toContain('some raw output line')
    })

    it('serverOutput accumulates raw lines showing received files', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', 'Listening on :9527')
      triggerEvent('server-event', 'Received file: ./received/photo.jpg')
      triggerEvent('server-event', 'Received file: ./received/doc.pdf')
      await nextTick()

      const output = wrapper.vm.serverOutput
      expect(output.some((line) => line.includes('photo.jpg'))).toBe(true)
      expect(output.some((line) => line.includes('doc.pdf'))).toBe(true)

      wrapper.vm.serverRunning = true
      await nextTick()
      expect(wrapper.text()).toContain('Server Events')
      expect(wrapper.text()).toContain('photo.jpg')
    })

    it('malformed JSON is appended as raw text to serverOutput', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-event', '{invalid json')
      await nextTick()

      expect(wrapper.vm.serverOutput).toContain('{invalid json')
    })

    it('integrated receive flow: ready -> task -> progress -> complete', async () => {
      const wrapper = await mountApp()

      triggerEvent('server-event', JSON.stringify({ type: 'ready', port: 9527, dir: './received' }))
      await nextTick()
      expect(wrapper.vm.serverRunning).toBe(true)

      triggerEvent('server-event', JSON.stringify({ type: 'task', id: 'recv-job', item_count: 3, total_size: 2048 }))
      await nextTick()
      expect(wrapper.vm.currentTask.id).toBe('recv-job')
      expect(wrapper.vm.transferActive).toBe(true)

      triggerEvent('server-event', JSON.stringify({ type: 'progress', kind: 'file', path: './received/photos', done: true }))
      triggerEvent('server-event', JSON.stringify({ type: 'progress', kind: 'file', path: './received/photos/img1.png', done: true }))
      triggerEvent('server-event', JSON.stringify({ type: 'progress', kind: 'file', path: './received/photos/img2.png', done: true }))
      await nextTick()
      expect(wrapper.vm.progressItems).toHaveLength(3)
      expect(wrapper.vm.progressItems[0].path).toContain('photos')

      triggerEvent('server-event', JSON.stringify({ type: 'complete' }))
      await nextTick()
      expect(wrapper.vm.transferActive).toBe(false)
      expect(wrapper.vm.sending).toBe(false)

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Server ready on port 9527'))).toBeTruthy()
      expect(logs.find((l) => l.msg.includes('Receive complete!'))).toBeTruthy()
    })

    it('new receive task sets currentTask but does not clear previous progressItems', async () => {
      const wrapper = await mountApp()
      wrapper.vm.progressItems = [{ kind: 'file', path: '/old.txt', done: true }]
      wrapper.vm.transferActive = true
      await nextTick()

      triggerEvent('server-event', JSON.stringify({ type: 'task', id: 'new-task', item_count: 2, total_size: 100 }))
      await nextTick()

      expect(wrapper.vm.currentTask.id).toBe('new-task')
      expect(wrapper.vm.transferActive).toBe(true)
      expect(wrapper.vm.progressItems).toHaveLength(1)
      expect(wrapper.vm.progressItems[0].path).toBe('/old.txt')
    })

    it('receive complete does not clear sending flag (send side independent)', async () => {
      const wrapper = await mountApp()
      wrapper.vm.sending = true
      wrapper.vm.transferActive = true
      await nextTick()

      triggerEvent('server-event', JSON.stringify({ type: 'complete' }))
      await nextTick()

      expect(wrapper.vm.transferActive).toBe(false)
      expect(wrapper.vm.sending).toBe(true)
    })
  })

  describe('Event Handling: transfer-error', () => {
    it('logs transfer error', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-error', 'connection lost')
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Transfer error'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.type).toBe('error')
    })
  })

  describe('Event Handling: transfer-complete', () => {
    it('resets state and logs', async () => {
      const wrapper = await mountApp()
      wrapper.vm.sending = true
      wrapper.vm.transferActive = true
      await nextTick()

      triggerEvent('transfer-complete', '0')
      await nextTick()

      expect(wrapper.vm.sending).toBe(false)
      expect(wrapper.vm.transferActive).toBe(false)

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('finished with code'))).toBeTruthy()
    })
  })

  describe('Event Handling: server-error', () => {
    it('logs server error', async () => {
      const wrapper = await mountApp()
      triggerEvent('server-error', 'crashed')
      await nextTick()

      const logs = getLogs(wrapper)
      const errorLog = logs.find((l) => l.msg.includes('Server error'))
      expect(errorLog).toBeTruthy()
      expect(errorLog.type).toBe('error')
    })
  })

  describe('Event Handling: server-terminated', () => {
    it('sets serverRunning to false and logs', async () => {
      const wrapper = await mountApp()
      wrapper.vm.serverRunning = true
      await nextTick()

      triggerEvent('server-terminated')
      await nextTick()

      expect(wrapper.vm.serverRunning).toBe(false)

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Server process terminated'))).toBeTruthy()
    })
  })

  describe('Log System', () => {
    it('addLog prepends entries with timestamp', async () => {
      const wrapper = await mountApp()
      wrapper.vm.logs = []
      await nextTick()

      wrapper.vm.addLog('test message', 'info')
      wrapper.vm.addLog('second', 'success')
      await nextTick()

      const logs = getLogs(wrapper)
      expect(logs.length).toBe(2)
      expect(logs[0].msg).toBe('second')
      expect(logs[1].msg).toBe('test message')
      expect(logs[0].time).toBeTruthy()
      expect(logs[0].type).toBe('success')
    })

    it('caps at 200 entries', async () => {
      const wrapper = await mountApp()
      for (let i = 0; i < 250; i++) {
        wrapper.vm.addLog(`msg ${i}`)
      }
      await nextTick()

      expect(getLogs(wrapper).length).toBe(200)
    })

    it('serverOutput caps at 100 entries', async () => {
      const wrapper = await mountApp()
      for (let i = 0; i < 150; i++) {
        triggerEvent('server-event', `line ${i}`)
      }
      await nextTick()

      expect(wrapper.vm.serverOutput.length).toBe(100)
    })

    it('shows empty state when no logs', async () => {
      const wrapper = await mountApp()
      wrapper.vm.logs = []
      await nextTick()
      expect(wrapper.text()).toContain('No logs yet')
    })

    it('colors log lines by type', async () => {
      const wrapper = await mountApp()
      wrapper.vm.addLog('info msg', 'info')
      wrapper.vm.addLog('success msg', 'success')
      wrapper.vm.addLog('error msg', 'error')
      wrapper.vm.addLog('warn msg', 'warn')
      await nextTick()

      const logLines = wrapper.findAll('.text-emerald-600')
      expect(logLines.length).toBeGreaterThan(0)

      const errorLines = wrapper.findAll('.text-red-600')
      expect(errorLines.length).toBeGreaterThan(0)
    })
  })

  describe('formatSize (indirectly via task events)', () => {
    it('formats 0 as 0 B', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't', item_count: 0, total_size: 0 }))
      await nextTick()

      const logs = getLogs(wrapper)
      const taskLog = logs.find((l) => l.msg.includes('Receiving task'))
      expect(taskLog.msg).toContain('0 B')
    })

    it('formats bytes < 1024 as n B', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't', item_count: 1, total_size: 512 }))
      await nextTick()

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Receiving task')).msg).toContain('512.0 B')
    })

    it('formats KB range', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't', item_count: 1, total_size: 1536 }))
      await nextTick()

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Receiving task')).msg).toContain('1.5 KB')
    })

    it('formats MB range', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't', item_count: 1, total_size: 10485760 }))
      await nextTick()

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Receiving task')).msg).toContain('10.0 MB')
    })

    it('formats GB range', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't', item_count: 1, total_size: 1073741824 }))
      await nextTick()

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Receiving task')).msg).toContain('1.0 GB')
    })

    it('formats null/undefined as 0 B', async () => {
      const wrapper = await mountApp()
      triggerEvent('transfer-progress', JSON.stringify({ type: 'task', id: 't', item_count: 1 }))
      await nextTick()

      const logs = getLogs(wrapper)
      expect(logs.find((l) => l.msg.includes('Receiving task')).msg).toContain('0 B')
    })
  })

  describe('canSend computed', () => {
    it('is true when peer selected, items exist, and not sending', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedPeer = '192.168.1.100'
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      wrapper.vm.sending = false
      await nextTick()

      expect(wrapper.vm.canSend).toBe(true)
    })

    it('is false without peer', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      await nextTick()

      expect(wrapper.vm.canSend).toBeFalsy()
    })

    it('is false without items', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedPeer = '192.168.1.100'
      await nextTick()

      expect(wrapper.vm.canSend).toBe(false)
    })

    it('is false when sending', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedPeer = '192.168.1.100'
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      wrapper.vm.sending = true
      await nextTick()

      expect(wrapper.vm.canSend).toBe(false)
    })
  })

  describe('Integration: reactive chain peer/selection -> canSend -> button', () => {
    it('FileSelector dropdown select updates selectedPeer and canSend', async () => {
      const wrapper = await mountApp()
      wrapper.vm.peers = [{ name: 'Alice-PC', addr: '192.168.1.100' }]
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      await nextTick()

      expect(wrapper.vm.canSend).toBe(false)

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('update:selectedPeer', '192.168.1.100')
      await nextTick()

      expect(wrapper.vm.selectedPeer).toBe('192.168.1.100')
      expect(wrapper.vm.canSend).toBe(true)

      const sendBtn = wrapper.findAll('button').find((b) => {
        const t = b.text()
        return t.includes('Send') && !t.includes('Add') && !t.includes('Server')
      })
      expect(sendBtn).toBeTruthy()
      expect(sendBtn.attributes('disabled')).toBeUndefined()
    })

    it('DeviceList peer click updates selectedPeer and canSend', async () => {
      const wrapper = await mountApp()
      wrapper.vm.peers = [{ name: 'Alice-PC', addr: '192.168.1.100' }]
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      await nextTick()

      expect(wrapper.vm.canSend).toBe(false)

      const deviceList = wrapper.findComponent({ name: 'DeviceList' })
      await deviceList.vm.$emit('update:selectedPeer', '192.168.1.100')
      await nextTick()

      expect(wrapper.vm.selectedPeer).toBe('192.168.1.100')
      expect(wrapper.vm.canSend).toBe(true)

      const sendBtn = wrapper.findAll('button').find((b) => {
        const t = b.text()
        return t.includes('Send') && !t.includes('Add') && !t.includes('Server')
      })
      expect(sendBtn).toBeTruthy()
      expect(sendBtn.attributes('disabled')).toBeUndefined()
    })

    it('clearing peer via dropdown disables canSend', async () => {
      const wrapper = await mountApp()
      wrapper.vm.peers = [{ name: 'Alice-PC', addr: '192.168.1.100' }]
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      await nextTick()

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('update:selectedPeer', '192.168.1.100')
      await nextTick()
      expect(wrapper.vm.canSend).toBe(true)

      await fileSelector.vm.$emit('update:selectedPeer', '')
      await nextTick()
      expect(wrapper.vm.canSend).toBe(false)

      const sendBtn = wrapper.findAll('button').find((b) => {
        const t = b.text()
        return t.includes('Send') && !t.includes('Add') && !t.includes('Server')
      })
      expect(sendBtn.attributes('disabled')).toBeDefined()
    })

    it('removing last path disables canSend', async () => {
      const wrapper = await mountApp()
      wrapper.vm.selectedPeer = '192.168.1.100'
      wrapper.vm.selectedItems = [{ path: '/test', isFile: true }]
      await nextTick()
      expect(wrapper.vm.canSend).toBe(true)

      const fileSelector = wrapper.findComponent({ name: 'FileSelector' })
      await fileSelector.vm.$emit('remove-path', 0)
      await nextTick()

      expect(wrapper.vm.canSend).toBe(false)
      const sendBtn = wrapper.findAll('button').find((b) => {
        const t = b.text()
        return t.includes('Send') && !t.includes('Add') && !t.includes('Server')
      })
      expect(sendBtn.attributes('disabled')).toBeDefined()
    })
  })

  describe('Lifecycle', () => {
    it('registers 6 event listeners on mount', async () => {
      await mountApp()

      expect(listeners['transfer-progress']).toBeDefined()
      expect(listeners['transfer-error']).toBeDefined()
      expect(listeners['transfer-complete']).toBeDefined()
      expect(listeners['server-event']).toBeDefined()
      expect(listeners['server-error']).toBeDefined()
      expect(listeners['server-terminated']).toBeDefined()
    })

    it('calls unlisten functions on unmount', async () => {
      const wrapper = await mountApp()
      wrapper.unmount()

      expect(capturedUnlisteners.length).toBe(6)
      expect(Object.keys(listeners).length).toBe(0)
    })
  })
})
