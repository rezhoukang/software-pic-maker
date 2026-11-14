# EZ软图

Wails 2 + Go + Vue 3 + TypeScript 的本地软件设计绘图应用，带独立 stdio MCP 服务。

## 开始

已构建版本：`dist/EZ软图-0.2.14/EZ软图.exe`。便携压缩包：`dist/EZ软图-0.2.14-windows-amd64.zip`。详细说明见 [使用说明](docs/使用说明.md)。

开发环境：Go 1.25+、Node.js 22.12+（本机 24）、npm、Windows WebView2 Runtime。纯 Go 构建，不需要 MSVC/C++ Build Tools。已固定 Wails 2.16.0，CLI 安装到项目 `.tools`。

```powershell
# 首次安装前端依赖
npm.cmd ci --prefix frontend
# 启动 Wails 开发窗口
./scripts/dev.ps1
# 测试、构建 exe 和 ZIP
./scripts/build.ps1
```

首次构建需要下载依赖。Go 模块和编译缓存使用系统默认位置。打包后用户无需安装开发工具。

## 结构

| 路径 | 职责 |
| --- | --- |
| `main.go` / `app.go` | 桌面窗口、原生文件对话框、项目读写和恢复 |
| `frontend/src` | Vue 编辑界面、Naive UI 控件、X6 画布 |
| `internal/core` | Schema/语义校验、图种布局、SVG/draw.io 导出 |
| `shared` | 桌面和 MCP 共用 Schema、八种示例 |
| `cmd/mcp` | 独立 MCP 服务，官方 Go SDK |
| `cmd/preview` | 仅用于浏览器开发测试的本机 HTTP 接口 |
| `scripts` | 开发、构建与资源生成脚本 |

左侧目录树挂载真实 `data` 文件夹，一张图对应一个 JSON 文件。文件夹支持新建文件夹/文件、重命名、删除；文件支持重命名、删除、设计表。按分类下拉筛选文件，顶栏显示当前路径。

源码目录内的构建共用仓库根目录 `data`，外部便携副本使用程序旁的 `data`。恢复和 WebView 缓存收纳在 `data/.ezbitu`。删除移入该目录下的 `deleted-*`，不会直接永久擦除。

JSON 是项目源数据。Go 根据图种生成统一场景，X6 呈现节点并处理选择、拖动和缩放。SVG、draw.io 和 MCP 使用同一生成核心，MCP 的矢量输出无需浏览器，PNG 使用本机 Edge/Chrome 的隐藏无头进程。前端通过 AJV 做结构校验，Go 同时校验结构与引用。

## 测试

```powershell
go test ./internal/core ./cmd/mcp
go vet ./...
```

Python 辅助脚本的类型检查：`npm exec --yes --package=pyright -- pyright`。检查范围和规则由 `pyrightconfig.json` 定义。

浏览器测试必须使用隔离目录，同时运行以下两个服务（各开一个终端），然后执行测试。Playwright 使用已安装的 Microsoft Edge，无需额外下载 Chromium。

```powershell
go run ./cmd/preview --workspace test-results/file-workspaces --isolated-workspaces
npm.cmd run dev --prefix frontend
# 第三个终端
npm.cmd test --prefix frontend
```

覆盖八种模板渲染、引用错误、循环关系、空关系项目往返、XML 可解析性与 ID/连接完整性、位置稳定性、MCP 工具握手调用，以及编辑、拖动、撤销重做、JSON、恢复、四种导出和项目打开。21 组浏览器场景还覆盖未应用草稿保护、迟到校验响应、无效导入/保存、损坏恢复副本和最小窗口布局；浏览器预览与桌面保存均使用 Go 项目校验。

注意：XML 结构通过不等于已在所有 draw.io 版本中验证显示效果。原生对话框仍需实际桌面验证；浏览器预览的下载/打开流程与 Windows 原生对话框不同。

## 限制

详见使用说明。当前构建目标为 Windows amd64；尚未验证 macOS/Linux 或 ARM64。draw.io 是额外的单向导出，不是项目保存格式。无联网模型调用，无数据库连接功能，无自动更新服务。
