# xShare 详细设计文档 (v1.0)

## 1. 项目定义

**xShare** 是一款局域网文件共享工具，通过 Go 核心引擎与 Tauri UI 结合，实现跨平台、零配置的文件夹与文件高速传输。

## 2. 核心架构

```
┌─────────────────────────────────────┐
│            UI 层 (src/)              │
│     Vue 3 + Tailwind CSS + Vite     │
│   DeviceList / FileSelector /       │
│   TransferProgress                  │
└──────────────┬──────────────────────┘
               │ Tauri Events + invoke()
┌──────────────┴──────────────────────┐
│         桥接层 (src-tauri/)          │
│      Tauri v2 (Rust)                │
│  - 进程生命周期管理                  │
│  - 系统原生对话框 (DialogExt)        │
│  - Sidecar 管理 (ShellExt)          │
│  - 事件转发 (Emitter)               │
└──────────────┬──────────────────────┘
               │ Sidecar process (stdout/stderr)
┌──────────────┴──────────────────────┐
│         引擎层 (go-engine/)          │
│           Golang                     │
│  - mDNS 设备发现 (discovery/)        │
│  - xShare TCP 协议 (protocol/)       │
│  - 文件传输逻辑 (transfer/)          │
│  - CLI 入口 (main.go)               │
└─────────────────────────────────────┘
```

**数据流向：**
- **UI → Go:** Tauri `invoke()` 调用 Rust command → Rust 启动 Go 子进程 → Go 通过 stdout 输出 JSON Lines → Rust 监听 stdout 并通过 `Emitter` 转发给 UI
- **Go → 网络:** TCP 直连，不经过 Tauri/Rust 中转

**Tauri 事件通道：**

| 事件名 | 来源 | 用途 |
|:---|:---|:---|
| `server-event` | Go stdout | 服务端状态通知 (ready/task/progress/complete) |
| `server-error` | Go stderr | 服务端错误 |
| `server-terminated` | sidecar 退出信号 | 服务进程退出通知 |
| `transfer-progress` | Go stdout | 发送端进度 (progress/complete) |
| `transfer-error` | Go stderr | 发送端错误 |
| `transfer-complete` | sidecar 退出信号 | 发送进程结束 |

## 3. 传输协议 (xShare Protocol v1)

### 3.1 协议头 (Header) 定义

每个报文由 **14 字节固定头部** 开始，大端序：

| 字节偏移 | 字段 | 类型 | 说明 |
|:---|:---|:---|:---|
| 0-3 | **Magic** | uint32 | 固定为 `0x58534852` (ASCII: xSHR) |
| 4 | **Version** | uint8 | 协议版本，当前固定为 `0x01` |
| 5 | **Type** | uint8 | 消息类型 (详见 3.2) |
| 6-13 | **Length** | uint64 | 后续载荷 (Payload) 的字节长度 $L$ |

**实现文件:** `go-engine/pkg/protocol/codec.go`
- `EncodeHeader` — 编码 header 为 14 字节 `[]byte`
- `DecodeHeader` — 使用 `io.ReadFull` 从 TCP stream 读取 14 字节并解码
- `ReadPayload` — 根据 header.Length 使用 `io.ReadFull` 读取完整 payload
- `Send` / `SendJSON` — 快捷写入方法
- `EncodeMessage` / `EncodeJSON` — 编码为完整报文

### 3.2 消息类型与 Payload

| Type | 常量名 | Payload | 结构体 | 说明 |
|:---|:---|:---|:---|:---|
| `0x01` | `TypeTaskInfo` | JSON | `TaskInfo{ID, TotalSize, ItemCount}` | 目录传输任务元信息 |
| `0x02` | `TypeFileHeader` | JSON | `FileHeader{Path, Size, Mode, Hash}` | 单个文件元数据。Path 语义：单文件时为文件名；目录传输时为 `dirname/relPath` |
| `0x03` | `TypeChunk` | Binary | `[]byte` | 文件二进制碎片 (64KB buf) |
| `0x04` | `TypeAck` | JSON | `Ack{Status, Code, Msg}` | 同步回执 |
| `0x05` | `TypeDone` | None | — | 任务结束 (Length=0) |

**Ack 错误码定义：**

| Code | 场景 | 说明 |
|:---|:---|:---|
| 0 | 成功 | Status="ok" |
| 1 | 目录创建失败 | FileHeader 阶段父目录 mkdir 失败 |
| 2 | 文件创建失败 | FileHeader 阶段 os.Create 失败 |
| 3 | 磁盘空间不足 | (预留) |
| 4 | Hash 校验失败 | Chunk 传输完毕 Hash 不一致 |
| 5 | 路径穿越 | FileHeader.path 包含 `..` 或解析后超出目标根目录 |

### 3.3 编解码实现细节

**类型定义:** `go-engine/pkg/protocol/types.go`
```go
type Header struct {
    Magic   uint32
    Version uint8
    Type    uint8
    Length  uint64
}
type TaskInfo struct { ID string `json:"id"`; TotalSize int64 `json:"total_size"`; ItemCount int `json:"item_count"` }
type FileHeader struct { Path string `json:"path"`; Size int64 `json:"size"`; Mode int `json:"mode"`; Hash string `json:"hash"` }
type Ack struct { Status string `json:"status"`; Code int `json:"code"`; Msg string `json:"msg"` }
```

**常量定义:** `go-engine/pkg/protocol/constants.go`
```go
const Magic uint32 = 0x58534852
const Version uint8 = 0x01
const HeaderSize = 14
// Type 常量: TypeTaskInfo=0x01 ... TypeDone=0x05
```

## 4. 单文件传输流程

适用于 `cmdSend --file=FILE`，**不发送** `TaskInfo` / `Done`。

```
Sender                              Receiver
  │                                     │
  │ ─── Type 0x02 FileHeader ────────► │  ├─ json.Unmarshal → FileHeader
  │                                     │  ├─ MkdirAll(filepath.Dir(fullPath))
  │                                     │  ├─ os.Create(fullPath)
  │ ◄─── Type 0x04 Ack ────────────── │  └─ sendAck("ok", 0, "")
  │                                     │
  │ ─── Type 0x03 Chunk (×N) ───────► │  写入文件 + sha256.New().Write()
  │                                     │
  │ ─── Last Chunk ──────────────────► │  receivedSize >= currentSize
  │                                     │  ├─ 计算 sha256:hex
  │                                     │  ├─ 比对 FileHeader.Hash
  │ ◄─── Type 0x04 Ack ────────────── │  └─ sendAck(结果)
  │                                     │
```

**实现:**
- Sender: `sender.go` → `SendFile()` → `sendFile()` (私有)
- Receiver: `receiver.go` → `handleFileHeader()` → `handleChunk()` → `verifyAndSendFinalAck()`

**关键细节：**
- `sendFile` 先计算 SHA-256 (完整 io.Copy to hasher)，再 Seek(0,0) 后分块发送
- 空文件 (Size=0) 发送一个零长度 `Chunk` 标记传输结束
- 输出 JSON: `{"type":"progress","path":"...","kind":"file","done":true}` 和 `{"type":"complete","total_files":1}`

## 5. 目录传输流程

适用于 `cmdSend --dir=DIR`。目录传输将树结构平铺为有序 TCP 指令序列。

```
Sender                                      Receiver
  │                                             │
  │ ─── Type 0x01 TaskInfo ──────────────────► │ → handleTaskInfo()
  │    {id: baseName, total_size: N, item_count: M} 输出 task JSON
  │                                             │
  │ filepath.Walk() 遍历:                        │
  │   IsDir → skip                              │
  │                                             │
  │     ─── Type 0x02 FileHeader ────────────► │ → handleFileHeader()
  │     ◄─── Type 0x04 Ack ─────────────────   │    → sendAck("ok",0,"")
  │     ─── Type 0x03 Chunk ×N ─────────────► │ → handleChunk()
  │     ◄─── Type 0x04 Ack ─────────────────   │    → verifyAndSendFinalAck()
  │                                             │
  │   # 严格串行：上一文件 final Ack ok 后        │
  │   # 才发送下一文件 FileHeader                │
  │                                             │
  │ ─── Type 0x05 Done ──────────────────────► │ → finalize()
  │    (Length=0)                                │    → resetCurrentFile() + "complete"
```

**实现:** `sender.go` → `SendDirectory()` / `receiver.go` → `Receive()` 主循环

**关键细节：**
- `TaskInfo.ID` = `filepath.Base(dirPath)`
- `TaskInfo.TotalSize` = 所有文件 Size 之和
- `TaskInfo.ItemCount` = 文件总数（不含目录）
- `FileHeader.Path` = `filepath.ToSlash(filepath.Join(baseName, relPath))`，如 `src/sub/b.txt`
- 接收端根据 FileHeader.path 自动创建父目录，保持原有层级结构
- 路径穿越检测：拒绝包含 `..` 的路径或解析后超出目标根目录
- 接收端每个连接独立 goroutine `go Receive(conn)`

## 6. 设备发现 (mDNS)

### 6.1 服务注册

`go-engine/pkg/discovery/mdns.go` → `Register(port, bindIP)`
- 服务名: `_xshare._tcp`
- 服务信息: `["xShare v1"]`
- `--ip` 指定时通过 `resolveIP()` 查找到对应网卡接口后绑定

### 6.2 设备发现

`go-engine/pkg/discovery/mdns.go` → `Discover(timeout, bindIP)`
- 使用 `hashicorp/mdns` 库 `mdns.Query` + `WantUnicastResponse`
- 回调收集 `mdns.ServiceEntry` → 转为 `Peer` 列表

### 6.3 数据结构

```go
type Peer struct {
    Name string `json:"name"`   // 主机名
    Host string `json:"host"`   // IP
    Port int    `json:"port"`   // 端口
    Addr string `json:"addr"`   // "host:port"
}

type IPInfo struct {
    IP     string `json:"ip"`
    Iface  string `json:"iface"`
    Family string `json:"family"` // "v4" | "v6"
}
```

### 6.4 IP 管理

`ListIPs()` 遍历所有网卡，过滤条件: IsUp、非 Loopback、非 Multicast、非 LinkLocalUnicast、非 Unspecified。

`resolveIP(ipStr)` 精确查找 IP 对应的 `*net.Interface`。

## 7. CLI 命令规范

入口: `go-engine/main.go`。所有输出为 **JSON Lines** 格式到 stdout。

### 7.1 `serve`

```
go-engine serve [--port=9527] [--dir=./received] [--ip=ADDR]
```
- 启动 mDNS 注册 + TCP server 监听
- 输出 `{"type":"ready","port":P,"dir":"D"}` 标记就绪
- 每个连接独立 goroutine 处理 Receive
- 监听 SIGINT 优雅退出

### 7.2 `discover`

```
go-engine discover [--timeout=5] [--ip=ADDR]
```
- 输出 `{"type":"peers","peers":[...]}`

### 7.3 `send`

```
go-engine send --peer=ADDR (--file=FILE | --dir=DIR)
```
- `--peer` 必选，格式 `host:port`
- `--file` 和 `--dir` 互斥
- `--file` → §4 单文件流程
- `--dir` → §5 目录流程

### 7.4 `list-ips`

```
go-engine list-ips
```
- 输出 `{"type":"ips","ips":[...]}`

### 7.5 参数解析

`getArg(args, name, defaultVal)` 解析 `--name=value` 格式参数，返回字符串或默认值。

## 8. Tauri 桥接命令

实现: `src-tauri/src/lib.rs`

### 8.1 命令列表

| 命令 | 参数 | Go 子命令 |
|:---|:---|:---|
| `start_server` | `port: u16, dir: String, ip?: String` | `serve --port=P --dir=D [--ip=IP]` |
| `stop_server` | — | kill sidecar |
| `get_server_port` | — | 内存状态 |
| `discover_peers` | `timeout: u64, ip?: String` | `discover --timeout=T [--ip=IP]` |
| `send_files` | `peer: String, dir?: String, file?: String` | `send --peer=P (--file=F | --dir=D)` |
| `list_ips` | — | `list-ips` |
| `open_file_dialog` | — | 系统文件选择器 |
| `open_dir_dialog` | — | 系统目录选择器 |

### 8.2 进程管理

**ServerState:**
```rust
struct ServerState {
    child: Mutex<Option<CommandChild>>,
    port: Mutex<u16>,
}
```
同一时间仅允许一个服务进程，`start_server` 会先 kill 已有进程。

**启服模式 (start_server):** spawn 长驻进程，通过 `CommandEvent` stream 监听 stdout/stderr，转发为 Tauri events。启动超时 5s。

**短周期模式 (discover/send/list_ips):** 使用 `output()` 等待完成。

## 9. 前端组件

文件结构: `src/`

### 9.1 App.vue — 主应用

**顶层状态:**
- `serverRunning: ref<boolean>` / `serverPort: ref<number>` / `serverDir: ref<string>`
- `selectedItems: ref<{path: string, isFile: boolean}[]>`
- `selectedPeer: ref<string|null>`
- `sending: ref<boolean>` / `transferActive: ref<boolean>`
- `currentTask: ref<object|null>` / `progressItems: ref<object[]>`

**事件监听 (onMounted):**
- `transfer-progress` → `handleProgress()`: 解析 JSON，按 type 分发 (task / progress / complete)
- `transfer-error` → `handleTransferError()`: 日志记录
- `transfer-complete` → `handleTransferComplete()`: 重置 sending/transferActive
- `server-event` → `handleServerEvent()`: 解析 JSON 分发
- `server-error` → `handleServerError()`: 日志记录
- `server-terminated` → 重置 serverRunning

**发送逻辑 (`sendFiles`):**
```javascript
for (const item of selectedItems.value) {
  await invoke('send_files', {
    peer: selectedPeer.value,
    dir: item.isFile ? null : item.path,
    file: item.isFile ? item.path : null
  })
}
```

### 9.2 DeviceList.vue — 设备列表

- 显示 peers 列表，支持选择目标设备
- 提供 "Scan Network" 按钮触发 `discoverPeers()`
- 设备选择通过 `v-model:selectedPeer` 双向绑定

### 9.3 FileSelector.vue — 文件选择与发送

- "Add Files" → 调用 `open_file_dialog` → `selectedItems` 中标记 `isFile: true`
- "Add Folder" → 调用 `open_dir_dialog` → `selectedItems` 中标记 `isFile: false`
- 文件图标 (蓝色) 与目录图标 (琥珀色) 视觉区分
- 发送时根据 `item.isFile` 传入 `file` 或 `dir` 参数

### 9.4 TransferProgress.vue — 传输进度

- 显示 TaskInfo (任务名、总大小、条目数)
- 显示文件级进度列表 (path + kind + done)

## 10. 目录结构

```
xShare/
├── src/                          # Vue 3 前端
│   ├── App.vue                   # 主应用，状态管理 + 事件监听
│   └── components/
│       ├── DeviceList.vue        # 设备列表组件
│       ├── FileSelector.vue      # 文件选择与发送组件
│       └── TransferProgress.vue  # 传输进度组件
├── src-tauri/                    # Tauri v2 Rust 桥接
│   ├── src/lib.rs                # Tauri 命令 + sidecar 生命周期
│   ├── tauri.conf.json           # 窗口 + sidecar externalBin 配置
│   ├── capabilities/default.json # shell & dialog 权限
│   └── binaries/                 # Go sidecar 二进制
├── go-engine/                    # Go 引擎
│   ├── main.go                   # CLI 入口
│   ├── main_test.go              # CLI 参数解析测试
│   ├── pkg/discovery/
│   │   ├── mdns.go               # mDNS 注册/发现/IP 管理
│   │   └── mdns_test.go          # mDNS 集成测试
│   ├── pkg/protocol/
│   │   ├── constants.go          # 消息类型常量
│   │   ├── types.go              # 数据结构定义
│   │   ├── codec.go              # Header/Message 编解码
│   │   └── codec_test.go         # 协议单元测试
│   └── pkg/transfer/
│       ├── sender.go             # 发送端 (SendFile + SendDirectory)
│       ├── receiver.go           # 接收端 (Receive)
│       └── path_test.go          # 路径转换测试
├── docs/
│   ├── xShare.md                 # 技术设计文档 (简版)
│   ├── xShare-详细设计.md         # 本文档 (详细版)
│   └── TESTING.md                # 测试说明
└── AGENTS.md                     # 开发指引
```

## 11. 测试策略

### 11.1 现有单元测试

| 包 | 文件 | 覆盖内容 |
|:---|:---|:---|
| `go-engine` | `main_test.go` | `getArg` 参数解析、`cmdListIPs` JSON 输出 |
| `protocol` | `codec_test.go` | Header 编解码往返、字节序校验、无效 Magic/Version、Done、短读、所有消息类型 |
| `discovery` | `mdns_test.go` | Peer/IPInfo JSON 序列化、字段标签、ListIPs 过滤、resolveIP、Register+Discover 往返集成 |
| `transfer` | `path_test.go` | Unix/Windows 路径转换往返 |

### 11.2 计划中的测试

- 大文件流式负载测试 (20GB+)
- 海量小文件目录测试 (10,000+)
- 传输中断异常模拟 (kill 进程 / 断网)
- Win/Linux 跨系统兼容性测试

### 11.3 运行命令

```bash
cd go-engine && go test ./... -v && go vet ./...
cargo check --manifest-path src-tauri/Cargo.toml
npm test  # 前端单元测试
```

## 12. 异常处理

| 场景 | 处理方式 |
|:---|:---|
| 协议版本不匹配 | `DecodeHeader` 返回 error，终止连接 |
| TCP 粘包/短读 | `io.ReadFull` 读够 Header (14B) + Payload (Length) |
| Hash 校验失败 | 删除文件，`Ack{status:"error", code:4}` |
| 路径穿越攻击 | `Ack{status:"error", code:5}` 拒绝传输 |
| 磁盘空间/权限 | FileHeader 阶段 `sendAck("error", code:1-3)` |
| 目录遍历错误 | `filepath.Walk` 报告 error，终止发送 |
| mDNS 超时 | 返回部分已发现 peers |
| Sidecar 启动超时 | `start_server` 返回 error，不进监听状态 |

## 13. 构建与部署

### 13.1 前置条件

- Go 1.21+ / Node 18+ / Rust 1.70+
- Linux: `libwebkit2gtk-4.1-dev libgtk-3-dev libdbus-1-dev libayatana-appindicator3-dev librsvg2-dev libjavascriptcoregtk-4.1-dev`

### 13.2 构建命令

```bash
# 编译 Go sidecar
cd go-engine && go build -o ../src-tauri/binaries/go-engine-$(go env GOOS)-$(go env GOARCH) .

# 开发模式
npm run tauri dev

# 生产构建
npm run tauri build
```

### 13.3 跨平台编译

| 平台 | 目标 | 二进制名 |
|:---|:---|:---|
| Linux x64 | `x86_64-unknown-linux-gnu` | `go-engine-x86_64-unknown-linux-gnu` |
| Windows x64 | `x86_64-pc-windows-msvc` | `go-engine-x86_64-pc-windows-msvc.exe` |
| macOS x64 | `x86_64-apple-darwin` | `go-engine-x86_64-apple-darwin` |
| macOS ARM | `aarch64-apple-darwin` | `go-engine-aarch64-apple-darwin` |

## 14. 实现状态总览

| 模块 | 状态 | 备注 |
|:---|:---|:---|
| Protocol codec | ✅ | Header 编解码、5 种消息类型、JSON/Binary payload |
| mDNS 发现 | ✅ | Register / Discover / ListIPs / IP 绑定 |
| 单文件发送 (SendFile) | ✅ | §4 流程：FileHeader→Ack→Chunks→Ack |
| 目录发送 (SendDirectory) | ✅ | §5 流程：TaskInfo→Walk→(File/Mkdir→Ack)→Done |
| 接收端 (Receiver) | ✅ | 兼容单文件和目录两种模式 |
| CLI (main.go) | ✅ | serve / discover / send / list-ips |
| Tauri 桥接 (lib.rs) | ✅ | 8 个命令，sidecar 生命周期管理 |
| 前端 UI | ✅ | 设备列表 / 文件选择 / 进度展示 |
| 协议单元测试 | ✅ | 字节序/魔数校验/版本校验/短读/消息类型 |
| mDNS 集成测试 | ✅ | Register+Discover 往返 / IP 绑定 |
