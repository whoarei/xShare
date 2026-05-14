# xShare

局域网共享工具，通过 Go 核心引擎 + Tauri UI，实现跨平台、零配置的文件夹与文件高速同步。

## 架构

```
┌──────────────────────────────────────────┐
│  Vue 3 + Tailwind CSS                    │  ← UI 层
│  设备列表 / 文件选择 / 进度展示            │
├──────────────────────────────────────────┤
│  Tauri (Rust)                            │  ← 桥接层
│  进程管理 / 原生对话框 / 托管 Go Sidecar   │
├──────────────────────────────────────────┤
│  Golang Engine                           │  ← 引擎层
│  mDNS 发现 / xShare 协议 / 流式文件 IO    │
└──────────────────────────────────────────┘
```

### xShare 协议 (v1)

| 字节偏移 | 字段 | 类型 | 说明 |
|:---|:---|:---|:---|
| 0-3 | Magic | uint32 | 固定 `0x58534852` (xSHR) |
| 4 | Version | uint8 | 当前 `0x01` |
| 5 | Type | uint8 | 见下表 |
| 6-13 | Length | uint64 | Payload 字节长度 |

| Type | 名称 | Payload |
|:---|:---|:---|
| `0x01` | TaskInfo | JSON: `{id, total_size, item_count}` |
| `0x02` | FileHeader | JSON: `{path, size, mode, hash}` |
| `0x03` | Chunk | Binary: 原始文件数据 |
| `0x04` | Ack | JSON: `{status, code, msg}` |
| `0x05` | Mkdir | JSON: `{path}` |
| `0x06` | Done | 空载荷 |

## 项目结构

```
xShare/
├── docs/
│   ├── xShare.md            # 技术设计文档
│   └── TESTING.md           # 测试说明文档
├── go-engine/               # Go 核心引擎
│   ├── main.go              # CLI 入口 (serve / discover / send)
│   └── pkg/
│       ├── protocol/        # xShare 协议编解码 (14 字节定长头)
│       ├── discovery/       # mDNS 设备发现
│       └── transfer/        # 文件发送与接收
├── src-tauri/               # Tauri Rust 后端
│   ├── src/
│   │   ├── main.rs          # 入口
│   │   └── lib.rs           # 命令定义 + 进程管理
│   ├── binaries/            # Go sidecar 二进制
│   ├── tauri.conf.json      # 窗口 / sidecar 配置
│   └── capabilities/        # 权限声明
├── src/                     # Vue 3 前端
│   ├── App.vue              # 主布局
│   ├── main.js              # 入口
│   ├── style.css            # Tailwind 样式
│   └── components/
│       ├── DeviceList.vue   # 设备列表 + 发现
│       ├── FileSelector.vue # 文件选择 + 发送
│       └── TransferProgress.vue # 传输进度
├── package.json
├── vite.config.js
└── tailwind.config.js
```

## 编译

### 系统要求

| 平台 | 最低版本 |
|:---|:---|
| Ubuntu / Debian | **22.04+** (需 webkit2gtk 4.1) |
| Fedora | 38+ |
| Arch | 滚动更新 |
| macOS | 12+ |
| Windows | 10+ |

> Ubuntu 20.04 仅提供 webkit2gtk 4.0，**无法构建** Tauri v2。需升级到 22.04，或通过 PPA `ppa:webkit-team/ppa` 安装 4.1。

### 安装工具链

```bash
node --version   # >= 18
go version       # >= 1.21
rustc --version  # >= 1.70
```

**Linux 额外依赖：**

```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev \
  libdbus-1-dev libayatana-appindicator3-dev librsvg2-dev \
  libjavascriptcoregtk-4.1-dev
```

### 获取源码

```bash
git clone <repo-url> xShare
cd xShare
npm install
cd go-engine && go mod tidy && cd ..
```

### 开发模式 (分层独立运行)

项目每层可独立调试，不强制依赖 Tauri：

| 层级 | 命令 | 说明 |
|:---|:---|:---|
| Go 引擎 | `cd go-engine && go run . serve` | 命令行收发，配合 `go test` |
| Vue 前端 | `npm run dev` | Vite HMR，浏览器调试 → `http://localhost:1420` |
| Tauri 全栈 | `npm run tauri dev` | 桌面窗口，Rust 变更需重新编译 |

**Go 本地回环测试：**

```bash
# 终端 1
cd go-engine && go run . serve --port=9527 --dir=./test-recv

# 终端 2
mkdir -p test-data && echo "hello" > test-data/f.txt
cd go-engine && go run . send --peer=127.0.0.1:9527 --dir=./test-data
```

**运行测试：**

```bash
cd go-engine && go test ./... -v
```

### 生产构建

最终应用由前端 + Tauri 外壳 + Go sidecar 组成，`npm run tauri build` 将三者打包为安装包。

#### 1. 编译 Go Sidecar

Tauri 通过目标三元组 (target triple) 匹配 sidecar 二进制。先将对应平台的 Go 二进制编译到 `src-tauri/binaries/`：

| 平台 | 目标三元组 | 文件名 |
|:---|:---|:---|
| Linux x64 | `x86_64-unknown-linux-gnu` | `go-engine-x86_64-unknown-linux-gnu` |
| Windows x64 | `x86_64-pc-windows-msvc` | `go-engine-x86_64-pc-windows-msvc.exe` |
| macOS x64 | `x86_64-apple-darwin` | `go-engine-x86_64-apple-darwin` |
| macOS ARM | `aarch64-apple-darwin` | `go-engine-aarch64-apple-darwin` |

```bash
# 当前平台（自动检测三元组）
cd go-engine
go build -o ../src-tauri/binaries/go-engine-$(go env GOOS)-$(go env GOARCH) .

# 交叉编译示例
GOOS=windows GOARCH=amd64 go build -o ../src-tauri/binaries/go-engine-x86_64-pc-windows-msvc.exe .
GOOS=darwin  GOARCH=amd64 go build -o ../src-tauri/binaries/go-engine-x86_64-apple-darwin .
GOOS=darwin  GOARCH=arm64 go build -o ../src-tauri/binaries/go-engine-aarch64-apple-darwin .
```

#### 2. 构建安装包

```bash
npm run tauri build
```

产物位置 `src-tauri/target/release/bundle/`：

| 平台 | 产物 |
|:---|:---|
| Linux | `bundle/deb/xshare_1.0.0_amd64.deb`<br>`bundle/appimage/xshare_1.0.0_amd64.AppImage`<br>`bundle/rpm/xshare-1.0.0-1.x86_64.rpm` |
| Windows | `bundle/msi/xshare_1.0.0_x64_en-US.msi`<br>`bundle/nsis/xshare_1.0.0_x64-setup.exe` |
| macOS | `bundle/dmg/xshare_1.0.0_x64.dmg`<br>`bundle/macos/xshare.app` |

#### 3. 一键构建脚本

```bash
#!/bin/bash
set -e
PLATFORM="${1:-$(go env GOOS)}"
ARCH="${2:-$(go env GOARCH)}"

case "$PLATFORM" in
    linux)   TARGET="x86_64-unknown-linux-gnu";    EXT="" ;;
    windows) TARGET="x86_64-pc-windows-msvc";       EXT=".exe" ;;
    darwin)  TARGET="x86_64-apple-darwin";          EXT="" ;;
    *) echo "Unknown platform: $PLATFORM"; exit 1 ;;
esac

echo "=== 1/3 编译 Go sidecar ($TARGET) ==="
cd go-engine && GOOS="$PLATFORM" GOARCH="$ARCH" go build -o "../src-tauri/binaries/go-engine-$TARGET$EXT" . && cd ..

echo "=== 2/3 安装前端依赖 ==="
npm install --silent

echo "=== 3/3 构建 Tauri 安装包 ==="
npm run tauri build

echo "=== 完成 ==="
ls -lh src-tauri/target/release/bundle/*/ 2>/dev/null || true
```

用法：`./build.sh linux | windows | darwin`

### CI/CD

项目包含两个 GitHub Actions 工作流：

| 工作流 | 触发条件 | 行为 |
|:---|:---|:---|
| `pr.yml` | Pull Request → main/master | Go 测试 / 前端构建 / Rust 检查 并行运行 |
| `release.yml` | 推送 `v*` 标签 或手动触发 | 测试 → Go 交叉编译 → Tauri 打包 → 发布 Release |

**PR 检查流程：**

```
pull_request
    │
    ├── test-go       (go test + go vet)
    ├── test-frontend (npm ci + build)
    └── check-rust    (cargo check + clippy)
```

三个 job 并行执行，任一失败则 PR 标记 ❌，阻止合并。

**Release 发布流程：**

推送 `v*` 标签即可触发多平台构建，产物自动发布到 GitHub Release。

```bash
git tag v1.0.0
git push origin v1.0.0
```

流水线自动执行：

1. **测试** — `go test` + `go vet` + `npm run build`，任一步失败则终止发布
2. **Go 交叉编译** — `GOOS` 矩阵生成 Linux / Windows / macOS 四个 binary
3. **Tauri 桌面构建** — ubuntu / windows / macos 三台 runner 并行打包
4. **发布** — `.deb` / `.AppImage` / `.msi` / `.exe` / `.dmg` 上传为 Release Asset

工作流定义文件：`.github/workflows/pr.yml` / `.github/workflows/release.yml`。Release 也可在 Actions 页面手动触发 (`workflow_dispatch`)。

## 使用

### 命令行 (Go 引擎独立运行)

```bash
cd go-engine

# 接收端 — 启动服务，等待文件
go run . serve --port=9527 --dir=./received

# 发送端 — 先发现对端
go run . discover --timeout=5

# 发送端 — 发送目录
go run . send --peer=192.168.1.10:9527 --dir=./my-files
```

### 桌面应用

1. 启动 xShare 桌面应用
2. 点击 **Start Server**，设置接收目录和端口
3. 点击 **Discover** 扫描局域网内的其他 xShare 节点
4. 在右侧面板选择要发送的文件或文件夹，从下拉菜单选择目标设备
5. 点击 **Send**，底部进度条显示传输状态
6. 接收端自动保存文件到设定的接收目录

接收到的文件会保留原有目录结构。传输完成后可对比 hash 校验完整性。

## 文档

| 文档 | 路径 |
|:---|:---|
| 技术设计 | [docs/xShare.md](docs/xShare.md) |
| 测试说明 | [docs/TESTING.md](docs/TESTING.md) |

## 常见问题

### `npm run tauri build` 报错找不到 libwebkit2gtk-4.1

```
Package webkit2gtk-4.1 was not found
```

**原因：** Ubuntu < 22.04 没有该包。

```bash
# Ubuntu 22.04+
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev \
  libdbus-1-dev libayatana-appindicator3-dev librsvg2-dev

# Ubuntu 20.04 需加 PPA
sudo add-apt-repository ppa:webkit-team/ppa -y && sudo apt update
sudo apt install libwebkit2gtk-4.1-dev
```

### Go 编译报错找不到 go

从 `https://go.dev/dl/` 下载预编译包，解压后加入 PATH。

### mDNS 发现不到设备

确保两台设备在同一局域网，防火墙放行 UDP 5353 端口。

### 怎么生成图标

```shell
npm run tauri icon ./src-tauri/icon.png
```

## License

MIT
