# 更新日志

所有重要更改都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，

并且本项目遵循 [语义化版本控制](https://semver.org/lang/zh-CN/)。


## [0.4.2] - 2026-05-29


### ✅ 测试 (Tests)


- **updater:** Add comprehensive checkForUpdate unit tests — *libb*


### 🐛 修复 (Bug Fixes)


- **updater:** Expose checking ref from UpdateChecker component — *libb*
- **updater:** Add timeout and better error messages for update check — *libb*
- **updater:** Enable system-proxy feature for reqwest — *libb*
- **updater:** Correct endpoint URL and improve error messages — *libb*


## [0.4.1] - 2026-05-29


### Release


- V0.4.1 — *libb*


### 🐛 修复 (Bug Fixes)


- **updater:** Add error handling and user feedback for update check — *libb*
- **updater:** Make error/no-update messages clickable to retry — *libb*


## [0.4.0] - 2026-05-29


### Release


- V0.4.0 — *libb*


### ✨ 新功能 (Features)


- **settings:** Add save directory persistence and history — *libb*
- **ui:** Show full save directory path on hover tooltip — *libb*
- **updater:** Integrate tauri-plugin-updater for auto-update support — *libb*


## [0.3.0] - 2026-05-29


### Release


- V0.3.0 — *libb*


### ✨ 新功能 (Features)


- **ui:** Add About modal with version info and tech stack — *libb*
- **ui:** Add browse button for save directory — *libb*
- **tauri:** 限制应用为单实例 — *libb*
- **scripts:** Add git-cliff changelog generation — *libb*
- **changelog:** Embed changelog viewer in About dialog — *libb*


### 🐛 修复 (Bug Fixes)


- **go-engine:** Fix usage print — *libb*


## [0.2.0] - 2026-05-28


### Release


- V0.2.0 — *libb*


### ♻️ 重构 (Refactoring)


- **discovery:** Unify Register code path via collectIfaceConfigs — *libb*


### ✨ 新功能 (Features)


- **discovery:** Add list-ifaces command to show all network interfaces — *libb*
- **scripts:** Unify version management with single source of truth — *libb*


### 🐛 修复 (Bug Fixes)


- **discovery:** Bind mDNS to all LAN interfaces instead of system default — *libb*
- **discovery:** Replace hashicorp/mdns with grandcat/zeroconf to fix Windows multi-NIC mDNS — *libb*
- **version:** Fix wrong version string — *libb*


### 📝 文档 (Documentation)


- 为代码和测试用例添加简洁注释 — *libb*


### 📦 杂项 (Miscellaneous)


- Ignore src-tauri/gen/ build artifacts — *libb*


## [0.1.8] - 2026-05-20


### 🐛 修复 (Bug Fixes)


- **ci:** Use npm install instead of npm ci for cross-npm-version compat — *libb*


## [0.1.7] - 2026-05-20


### 🐛 修复 (Bug Fixes)


- Regenerate package-lock.json to include transitive esbuild deps — *libb*


## [0.1.6] - 2026-05-20


### ✨ 新功能 (Features)


- IP-based binding, single file send, and remove Mkdir protocol type — *libb*


## [0.1.5] - 2026-05-14


### ✨ 新功能 (Features)


- Update app icons and sync config — *libb*


### 🐛 修复 (Bug Fixes)


- Close entriesCh with recover to prevent deadlock on UDP6 failure — *libb*


## [0.1.4] - 2026-05-14


### 🐛 修复 (Bug Fixes)


- Prevent deadlock in discovery when mdns.Query fails — *libb*


### 📦 杂项 (Miscellaneous)


- Add GitHub issue templates — *libb*


## [0.1.2] - 2026-05-14


### 🐛 修复 (Bug Fixes)


- Fix rust build error — *libb*


## [0.1.1] - 2026-05-14


### 🐛 修复 (Bug Fixes)


- Fix releaseName — *libb*



