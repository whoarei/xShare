# xShare 技术设计文档 (v1.0)

## 1. 项目定义
**xShare** 是一款局域网共享工具，通过 Go 核心引擎与 Tauri UI 结合，实现跨平台、零配置的文件夹与文件高速同步。

## 2. 核心架构
*   **UI层:** Vue 3 + Tailwind CSS，负责交互逻辑与任务进度展示。
*   **桥接层:** Tauri (Rust) 管理进程生命周期，调用系统原生对话框，托管 Go Sidecar。
*   **引擎层:** Golang 处理 mDNS 设备发现、自定义 TCP 协议解析及高性能文件 IO。

---

## 3. 传输协议 (xShare Protocol v1)

为了支持协议演进与多类型数据交换，引入版本控制与固定头部设计。

### 3.1 协议头 (Header) 定义
每个报文由 **14 字节固定头部** 开始：

| 字节偏移 | 字段 | 类型 | 说明 |
| :--- | :--- | :--- | :--- |
| 0-3 | **Magic** | uint32 | 固定为 `0x58534852` (ASCII: xSHR) |
| 4 | **Version** | uint8 | 协议版本，当前固定为 `0x01` |
| 5 | **Type** | uint8 | 消息类型 (详见 3.2) |
| 6-13 | **Length** | uint64 | 后续载荷 (Payload) 的字节长度 $L$ |

### 3.2 消息类型与 Payload 详细定义

| Type (Hex) | 名称 | Payload 内容格式 | 详细描述 |
| :--- | :--- | :--- | :--- |
| `0x01` | **TaskInfo** | JSON | `{"id": string, "total_size": int64, "item_count": int}`<br>告知接收端任务总体规模，用于显示总进度。 |
| `0x02` | **FileHeader** | JSON | `{"path": string, "size": int64, "mode": int, "hash": string}`<br>单个文件的元数据。path 以 `/` 分隔。<br>**单文件:** path = 文件名 (如 `hello.txt`)<br>**目录传输:** path = `dirname/subpath` (如 `src/sub/b.txt`) |
| `0x03` | **Chunk** | Binary | `[]byte`<br>文件的原始二进制碎片数据。 |
| `0x04` | **Ack** | JSON | `{"status": "ok" / "error", "code": int, "msg": string}`<br>对上一个关键操作的同步回执。 |
| `0x05` | **Done** | None | 载荷长度为 0。表示当前任务全部逻辑结束。 |

---

## 4. 传输

基于TCP做协议报文的传输。

### 4.1 文件传输流程
单个文件的传输遵循"预检-流式对冲-校验"的逻辑：

1.  **握手阶段:** 发送端通过 TCP 连接发送 `Type 02 (FileHeader)`。
2.  **预检阶段:** 接收端收到文件信息，检查磁盘空间及权限，返回 `Type 04 (Ack)`。
3.  **传输阶段:** 发送端分块读取文件（Buffer 建议 64KB），循环发送 `Type 03 (Chunk)`。
4.  **收尾阶段:** 发送端传输完毕，接收端计算接收数据的 Hash，比对无误后返回 `Type 04 (Ack)`。

### 4.2 目录传输方案
目录传输通过将结构平铺化，转化为一系列有序的 TCP 指令：

1.  **初始化:** 发送 `Type 01 (TaskInfo)` 开启任务上下文。
2.  **遍历发送:** 发送端利用 `filepath.Walk` 递归扫描，跳过目录节点，仅发送文件。每个文件的 `FileHeader.path` 格式为 `源目录名/源内相对路径`（`/` 分隔）。<br>例：发送目录 `src/`（内含 `a.txt`、`sub/b.txt`）→ 接收端创建 `src/a.txt`、`src/sub/b.txt`。
3.  **目录自动创建:** 接收端收到 `FileHeader` 时，从 `path` 中提取目录部分，用 `MkdirAll` 自动创建缺失的父目录，**保持原有的目录层级结构**。同时校验路径是否超出目标根目录，防止路径穿越攻击。
4.  **串行同步:** 只有当一个文件的最后一个 `Chunk` 收到 `Ack` 后，才开始下一个文件的 `FileHeader` 发送。
5.  **任务结束:** 遍历完成后发送 `Type 05 (Done)`。

---

## 5. 测试方案

### 5.1 单元测试 (Unit Tests)
*   **协议解析:** 验证 `Header` 到 `Payload` 的解析逻辑，特别是大端/小端字节序处理。
*   **路径转换:** 测试 `Windows(path\to\file)` 到 `Unix(path/to/file)` 的一致性。

### 5.2 集成与性能测试
*   **流式负载测试:** 传输 20GB+ 的文件，验证内存占用 (RSS) 是否稳定在较低水位（避免一次性 load 进内存）。
*   **海量小文件测试:** 传输包含 10,000 个小文件的目录，验证 `Ack` 机制下的传输稳定性。
*   **异常模拟:** 在传输过程中强制杀死 Go 进程或断开 WiFi，测试接收端的文件残留清理或任务挂起逻辑。

### 5.3 兼容性测试
*   **跨系统传输:** 重点测试 Win 与 Linux 之间的 `FileMode` 映射及中文字符集转义。

---

## 6. 异常处理
*   **版本协商:** 若 `Version` 不匹配，接收端应返回包含错误码的 `Ack` 并主动挂断。
*   **TCP 粘包:** Go 端使用 `io.ReadFull` 强制读够 Header 长度，再根据 Header 里的 Length 读够 Payload 长度。