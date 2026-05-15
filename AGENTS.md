# AGENTS.md — xShare

## Architecture

Three-layer monorepo:
- `src/` — Vue 3 + Tailwind CSS frontend (Vite, port 1420)
- `src-tauri/` — Tauri v2 Rust bridge (sidecar mgmt, native dialogs)
- `go-engine/` — Go sidecar binary (mDNS, xShare TCP protocol, file I/O)

The Go binary runs as a **Tauri sidecar** (`src-tauri/binaries/go-engine-{target-triple}[.exe]`). Tauri spawns it for each operation. The UI communicates with Go via Tauri events (not direct HTTP).

## Commands always run from repo root

```
npm run dev              # Vite HMR frontend
npm run build            # Production frontend → dist/
npm test                 # Frontend unit tests (vitest)
npm run test:watch       # Frontend tests in watch mode
npm run tauri dev        # Full desktop dev (needs Go sidecar prebuilt + Linux deps)
npm run tauri build      # Production desktop build
```

Go tests and vet (run inside `go-engine/`):
```bash
cd go-engine && go test ./... -v
cd go-engine && go vet ./...
```

Rust check (from repo root):
```bash
cargo check --manifest-path src-tauri/Cargo.toml
cargo clippy --manifest-path src-tauri/Cargo.toml -- -D warnings
```

## Build prerequisites

- **Go 1.21+** — `go-engine/go.mod`
- **Node 18+** — `package.json`
- **Rust 1.70+** — `src-tauri/Cargo.toml`
- **Linux only:** `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev libdbus-1-dev libayatana-appindicator3-dev librsvg2-dev libjavascriptcoregtk-4.1-dev`
- **Ubuntu < 22.04 cannot build Tauri** — webkit2gtk 4.1 not available without PPA

## Sidecar binary naming

Tauri v2 matches sidecars by target triple. Before any `tauri dev` or `tauri build`, compile the Go binary to:

```
src-tauri/binaries/go-engine-{target-triple}[.exe]
```

| Platform | Target |
|:---|:---|
| Linux x64 | `x86_64-unknown-linux-gnu` |
| Windows x64 | `x86_64-pc-windows-msvc` |
| macOS x64 | `x86_64-apple-darwin` |
| macOS ARM | `aarch64-apple-darwin` |

```bash
# Current platform
cd go-engine && go build -o ../src-tauri/binaries/go-engine-$(go env GOOS)-$(go env GOARCH) .

# Cross-compile
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ../src-tauri/binaries/go-engine-x86_64-pc-windows-msvc.exe .
```

## Protocol constraints

Binary protocol with 14-byte fixed header (big-endian). All payloads are either JSON or raw bytes.

- Magic: `0x58534852`, Version: `0x01`
- Header decode uses `io.ReadFull` to prevent TCP short reads
- Go tests in `go-engine/pkg/protocol/codec_test.go` validate byte-order correctness
- The protocol types and codec live in `go-engine/pkg/protocol/` — do not duplicate

## CI

- **PRs** trigger `.github/workflows/pr.yml`: `go test + go vet` / `npm ci + build + test` / `cargo check + clippy` in parallel. All three must pass.
- **Tags `v*`** trigger `.github/workflows/release.yml`: test → cross-compile Go ×4 → Tauri build ×3 platforms → GitHub Release draft
- Frontend tests use vitest + @vue/test-utils — see `docs/TESTING.md` for details

## Key files

| File | Purpose |
|:---|:---|
| `docs/xShare.md` | Technical design doc (protocol spec, transfer flow) |
| `docs/TESTING.md` | Test instructions and environment setup |
| `go-engine/main.go` | CLI entry — subcommands `serve` / `discover` / `send` |
| `src-tauri/src/lib.rs` | All Tauri commands + sidecar lifecycle |
| `src-tauri/tauri.conf.json` | Window config, sidecar externalBin, bundle settings |
| `src-tauri/capabilities/default.json` | Permissions for shell & dialog plugins |
| `go-engine/pkg/protocol/` | Header codec, message types, todo tests |
| `go-engine/pkg/transfer/` | Sender/Receiver that implement the protocol flow |

## Gotchas

- The `go-engine` Go module name is `go-engine` — imports use `go-engine/pkg/...`
- Tauri v2 requires `use tauri::Emitter;` to call `.emit()` on `AppHandle`
- Icons must be **RGBA** PNGs (color type 6), not RGB — Tauri will panic at build time otherwise
- The Go engine outputs JSON lines to stdout for progress/status; Tauri parses these into events
- Linux cross-compile to Windows needs `mingw-w64` + Rust `x86_64-pc-windows-gnu` target + `.cargo/config.toml` linker config

## Git commit rules

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

## Guidelines


1. **Design before coding**: Always update the detailed design document before implementing new features.
2. **Test plan before modifying**: Determine test cases (additions, deletions, or updates) and define the testing strategy before changing any code.
