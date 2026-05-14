# xShare 测试说明文档

## 测试概览

| 测试层级 | 工具 | 文件位置 |
|:---|:---|:---|
| Go 单元测试 | `go test` | `go-engine/pkg/protocol/codec_test.go` |
| Go 单元测试 | `go test` | `go-engine/pkg/transfer/path_test.go` |
| 前端构建验证 | `npm run build` | Vite 生产构建 |
| Tauri 集成验证 | `npm run tauri build` | 全栈构建 |

---

## 0. 前置环境配置

运行测试前需安装以下工具。

### 0.1 Go 环境

**安装方式一：使用包管理器**

| 系统 | 命令 |
|:---|:---|
| Ubuntu / Debian | `sudo apt install golang-go` |
| Fedora / CentOS | `sudo dnf install golang` |
| Arch Linux | `sudo pacman -S go` |
| macOS (Homebrew) | `brew install go` |
| Windows (Scoop) | `scoop install go` |

**安装方式二：手动安装（无需 root）**

```bash
# 下载 Go（以 1.22.5 为例，替换为最新版本）
curl -sL https://go.dev/dl/go1.22.5.linux-amd64.tar.gz -o /tmp/go.tar.gz
tar -C /tmp -xzf /tmp/go.tar.gz
export PATH="/tmp/go/bin:$PATH"
go version  # 验证安装

# 建议写入 ~/.bashrc 持久化
echo 'export PATH="/tmp/go/bin:$PATH"' >> ~/.bashrc
```

**验证安装:**
```bash
go version
# 预期: go version go1.xx.x linux/amd64
```

### 0.2 Node.js 与 npm

| 系统 | 命令 |
|:---|:---|
| Ubuntu / Debian | `sudo apt install nodejs npm` |
| macOS | `brew install node` |
| 手动安装 | [nodejs.org](https://nodejs.org) 下载预编译包 |

```bash
node --version   # 预期: v18+
npm --version    # 预期: v9+
```

### 0.3 Rust (Tauri 构建时需要)

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
rustc --version   # 预期: rustc 1.xx.x
```

### 0.4 Tauri 系统依赖 (仅 Linux)

```bash
# Ubuntu / Debian
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev \
  libayatana-appindicator3-dev librsvg2-dev \
  libjavascriptcoregtk-4.1-dev libdbus-1-dev

# Fedora
sudo dnf install webkit2gtk4.1-devel gtk3-devel \
  libappindicator-gtk3-devel librsvg2-devel

# Arch
sudo pacman -S webkit2gtk-4.1 gtk3 libappindicator-gtk3 librsvg
```

### 0.5 安装项目依赖

```bash
# 前端依赖
cd xShare
npm install

# Go 依赖
cd go-engine
go mod tidy
```

### 0.6 环境检查脚本

将以下内容保存为 `check-env.sh` 并运行：

```bash
#!/bin/bash
echo "=== 环境检查 ==="

check_cmd() {
    if command -v "$1" &> /dev/null; then
        echo "  ✓ $1: $(command -v $1)"
    else
        echo "  ✗ $1: 未安装"
    fi
}

echo "--- 必需 ---"
check_cmd go
check_cmd node
check_cmd npm

echo "--- Tauri 构建 (可选) ---"
check_cmd rustc
check_cmd cargo

echo "--- Go 验证 ---"
go version 2>/dev/null || echo "  > 请安装 Go: https://go.dev/dl/"

echo "--- Node.js 验证 ---"
node --version 2>/dev/null || echo "  > 请安装 Node.js: https://nodejs.org"
```

---

## 1. Go 单元测试

### 1.1 协议解析测试 (`codec_test.go`)

**文件:** `go-engine/pkg/protocol/codec_test.go`

| 测试用例 | 说明 |
|:---|:---|
| `TestEncodeDecodeHeader` | Header 序列化/反序列化往返正确性 |
| `TestEncodeDecodeHeader_Endianness` | 大端字节序逐字节验证 (Magic 0x58534852) |
| `TestDecodeHeader_InvalidMagic` | 非法 Magic 应返回错误 |
| `TestDecodeHeader_VersionMismatch` | 版本不匹配应返回错误 |
| `TestEncodeMessage` | 完整报文 (Header + Payload) 编解码 |
| `TestEncodeDecodeJSON` | Mkdir 等 JSON 载荷编解码 |
| `TestAllMessageTypes` | TaskInfo / FileHeader / Ack / Mkdir 全类型测试 |
| `TestDoneMessage` | Done 类型零载荷验证 |
| `TestReadPayload_PartialRead` | 模拟 TCP 粘包：部分读取应报错 |

**运行:**
```bash
cd go-engine
go test ./pkg/protocol/ -v
```

**预期输出:**
```
=== RUN   TestEncodeDecodeHeader
--- PASS: TestEncodeDecodeHeader (0.00s)
=== RUN   TestEncodeDecodeHeader_Endianness
--- PASS: TestEncodeDecodeHeader_Endianness (0.00s)
=== RUN   TestDecodeHeader_InvalidMagic
--- PASS: TestDecodeHeader_InvalidMagic (0.00s)
=== RUN   TestDecodeHeader_VersionMismatch
--- PASS: TestDecodeHeader_VersionMismatch (0.00s)
=== RUN   TestEncodeMessage
--- PASS: TestEncodeMessage (0.00s)
=== RUN   TestEncodeDecodeJSON
--- PASS: TestEncodeDecodeJSON (0.00s)
=== RUN   TestAllMessageTypes
--- PASS: TestAllMessageTypes (0.00s)
=== RUN   TestDoneMessage
--- PASS: TestDoneMessage (0.00s)
=== RUN   TestReadPayload_PartialRead
--- PASS: TestReadPayload_PartialRead (0.00s)
PASS
ok  	go-engine/pkg/protocol	0.002s
```

### 1.2 路径转换测试 (`path_test.go`)

**文件:** `go-engine/pkg/transfer/path_test.go`

| 测试用例 | 说明 |
|:---|:---|
| `TestToUnixPath_WindowsToUnix` | Windows 反斜杠 → Unix 正斜杠转换 |
| `TestToNativePath_UnixToNative` | Unix 正斜杠 → 本机分隔符转换 |
| `TestPathRoundTrip` | 往返转换 + 验证无残留反斜杠 |
| `TestFilepathToSlash_OnLinux` | 文档化 Linux 上 `filepath.ToSlash` 行为 |

**运行:**
```bash
cd go-engine
go test ./pkg/transfer/ -v
```

**预期输出:**
```
=== RUN   TestToUnixPath_WindowsToUnix
--- PASS: TestToUnixPath_WindowsToUnix (0.00s)
=== RUN   TestToNativePath_UnixToNative
--- PASS: TestToNativePath_UnixToNative (0.00s)
=== RUN   TestPathRoundTrip
--- PASS: TestPathRoundTrip (0.00s)
=== RUN   TestFilepathToSlash_OnLinux
--- PASS: TestFilepathToSlash_OnLinux (0.00s)
PASS
ok  	go-engine/pkg/transfer	0.002s
```

---

## 2. 运行全部测试

```bash
cd go-engine
go test ./... -v
```

---

## 3. 集成与性能测试 (手动)

### 3.1 场景: 两台机器对传

**前提:** 两台机器在同一局域网，均已编译 Go 引擎。

**步骤:**

```bash
# === 接收端 (机器 A) ===
./go-engine serve --port=9527 --dir=./received

# === 发送端 (机器 B) ===
# 先发现对端
./go-engine discover --timeout=5
# 示例输出: {"type":"peers","peers":[{"name":"machine-a","host":"192.168.1.10","port":9527,...}]}

# 发送目录
./go-engine send --peer=192.168.1.10:9527 --dir=./test-data
```

**验证:**
```bash
diff -r ./test-data ./received/test-data
```

### 3.2 场景: 单机自测 (本地回环)

```bash
# 终端 1: 启动接收服务
./go-engine serve --port=9527 --dir=./received

# 终端 2: 发送到本地
./go-engine send --peer=127.0.0.1:9527 --dir=./test-data

# 终端 3: 验证
diff -r ./test-data ./received/test-data
```

### 3.3 流式负载测试 (20GB+ 文件)

```bash
# 生成 20GB 测试文件
dd if=/dev/urandom of=./test-data/bigfile.bin bs=1M count=20480

# 运行传输并监控内存
./go-engine serve --port=9527 --dir=./received &
PID=$!

# 在另一终端发送
./go-engine send --peer=127.0.0.1:9527 --dir=./test-data &
SEND_PID=$!

# 监控 RSS 内存 (应稳定在较低水位)
watch -n 1 "ps -o pid,rss,comm -p $PID -p $SEND_PID"

# 等待完成后验证
md5sum ./test-data/bigfile.bin ./received/test-data/bigfile.bin
```

### 3.4 海量小文件测试 (10,000+ 文件)

```bash
# 生成 10,000 个小文件
mkdir -p ./test-data/many
for i in $(seq 1 10000); do
  echo "file $i content" > ./test-data/many/file_$i.txt
done

# 发送并观察 Ack 日志
./go-engine send --peer=127.0.0.1:9527 --dir=./test-data

# 验证数量
find ./received/test-data -type f | wc -l  # 应为 10000
```

### 3.5 异常模拟

```bash
# 1. 传输中杀掉 Go 进程
./go-engine serve --port=9527 --dir=./received &
SERVER_PID=$!
./go-engine send --peer=127.0.0.1:9527 --dir=./test-data &
sleep 2
kill -9 $SERVER_PID
# 检查: 接收端残留文件应被清理, 发送端应报连接错误

# 2. 传输中断开网络 (需要手动操作)
# 启动传输后, 断开 WiFi / 拔网线, 观察超时行为和错误恢复
```

---

## 4. 前端构建验证

```bash
# 安装依赖
npm install

# 生产构建
npm run build

# 应无报错, dist/ 目录包含:
#   dist/index.html
#   dist/assets/index-*.css
#   dist/assets/index-*.js
```

---

## 5. 全栈集成验证 (Tauri)

```bash
# 需要先安装系统依赖:
# sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev ...

# 开发模式
npm run tauri dev

# 生产构建
npm run tauri build
```

---

## 6. 兼容性检查清单

| 测试项 | Linux | macOS | Windows | 状态 |
|:---|:---|:---|:---|:---|
| 协议编解码 (大端) | ✓ | 待测 | 待测 | |
| 路径 `\` → `/` | ✓ | 待测 | 待测 | |
| mDNS 注册 | ✓ | 待测 | 待测 | |
| mDNS 发现 | ✓ | 待测 | 待测 | |
| 中文文件名传输 | ✓ | 待测 | 待测 | |
| FileMode 映射 | ✓ | 待测 | 待测 | |
