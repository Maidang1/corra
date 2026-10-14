# Corra

**English** | [中文](#中文)

---

## English

Corra is a Markdown **notes app** with a **fully native UI**, written in
Go with the [MyGo](https://mygo.egoist.dev) toolkit. No webview, no HTML,
no JavaScript — the whole interface is drawn by Go on the GPU.

### Highlights

- **Apple Notes-style three-column layout** — folders in the sidebar,
  notes grouped by date (Today / Yesterday / Previous 7 / 30 days) in
  the list, a dated editor column on the right
- **Full Markdown editing that works** — the editor column is a real
  code editor widget: syntax highlighting, soft wrap, multiple carets,
  find & replace (Cmd+F), undo/redo (Cmd+Z); every keystroke autosaves
- **Edit / preview** — `Ctrl/Cmd+E` toggles between source editing and
  the rendered document; task checkboxes stay clickable in preview, and
  code/tables/math/diagrams open their source on click
- **GFM complete** — tables, task lists, strikethrough, footnotes
- **Math** — inline `$...$` and `$$...$$` blocks, typeset by go-latex
- **Diagrams** — ` ```mermaid ` (Rust/WASM renderer), ` ```d2 `,
  ` ```goat ` ASCII diagrams, rasterized in pure Go with an embedded
  font for labels
- **Data charts** — ` ```chart ` YAML blocks (line / bar / hbar / pie),
  rendered natively
- **Syntax highlighting** — 200+ languages via chroma, themed light & dark
- **GitHub image hosting** — insert an image and it is uploaded to a
  GitHub repo via the Contents API; a CDN link (jsDelivr by default,
  configurable) is inserted. The token never leaves your machine (0600
  file)
- **Export PDF** — embedded CJK font subset, works on any machine
- **Deep native integration** — system dark mode, drag & drop, file
  associations, external-change detection with a conflict dialog, global
  menu with accelerators

### Storage

Notes live in a plain folder of Markdown files
(`<config>/备忘录` by default — one folder of the app per
subfolder, like Apple Notes' folders). Anything that edits the files on
disk (Sync, git, other editors) shows up in the app; the app autosaves
back to the same files.

### Build

```sh
go mod tidy             # resolve modules (go.sum is not committed)
go run ./internal/genicon  # draw the app icon (pure Go, no binary assets)
go build ./...          # compile
go test ./...           # tests (headless: no window needed)
go tool mygo dev        # run in development
go tool mygo build      # package for the current platform
```

### Release

Enable the workflows once after cloning (the GitHub API token used by
agents cannot create workflow files, so they ship in `setup/`):

```sh
sh setup/install-ci.sh
```

Push a `v*` tag: GitHub Actions builds signed-ready packages for macOS,
Windows and Linux in parallel and uploads them to a draft release
(see `.github/workflows/release.yml`).

### Known limits (v0.1)

- go-latex cannot yet typeset TeX scripts (`^`, `_`); simple ones are
  transliterated to Unicode (E=mc²); complex formulas show their source
- The mermaid WASM renderer is not pixel-identical to upstream mermaid;
  `%%{init}%%` directives are ignored
- Diagrams render on a light card in both themes (by design, like GitHub)
- Printing is not exposed by MyGo for native-UI windows yet; use Export PDF

---

## 中文

Corra 是一个**纯原生 UI** 的 Markdown 备忘录应用，用 Go 基于
[MyGo](https://mygo.egoist.dev) 工具包编写：无 webview、无 HTML、无
JavaScript，整个界面由 Go 在 GPU 上绘制。

### 功能

- **类 Apple Notes 三栏布局**：左栏备忘录文件夹，中栏按日期分组
  （今天/昨天/最近 7 天/最近 30 天）的列表，右栏带日期的编辑区
- **完整可用的 Markdown 编辑**：右栏是真正的代码编辑器控件——语法
  高亮、软换行、多光标、查找替换（Cmd+F）、撤销重做（Cmd+Z），
  每次输入即时自动保存
- **编辑 / 预览**：`Ctrl/Cmd+E` 在源码编辑与渲染阅读间切换；预览里
  任务复选框仍可点击，代码块/表格/公式/图表点击即回源码编辑
- **GFM 全要素**：表格、任务列表、删除线、脚注
- **数学公式**：行内 `$...$` 与块级 `$$...$$`，go-latex 排版
- **图表**：` ```mermaid `（Rust/WASM 渲染）、` ```d2 `、` ```goat `
  ASCII 图，纯 Go 光栅化并内嵌字体绘制文字标签
- **数据图表**：` ```chart ` YAML 数据块（折线/柱状/条形/饼图）
- **语法高亮**：chroma 支持 200+ 语言，深浅色主题
- **GitHub 图床**：插入图片即通过 Contents API 上传到指定仓库并回插
  CDN 链接（默认 jsDelivr，可自定义反代）；token 只存本机（0600 权限）
- **导出 PDF**：内嵌中文字体子集，任何机器上都能正常显示
- **深度系统集成**：系统深浅色、拖放打开、文件关联、外部修改检测
  与冲突处理、全局菜单与快捷键

### 存储

备忘录就是一个普通文件夹里的 Markdown 文件（默认在
`<config>/备忘录`；库根目录下的每个子文件夹对应左栏一个文件夹，
与 Apple Notes 一致）。磁盘上的文件被别的工具（同步、git、其他
编辑器）修改会实时反映到应用里；应用的自动保存也直接写回同一批
文件。

### 构建

```sh
go mod tidy                # 解析依赖（go.sum 不入库，由 CI 重新生成）
go run ./internal/genicon  # 生成应用图标（纯 Go 绘制，无二进制资源）
go build ./...
go test ./...
go tool mygo dev     # 开发运行
go tool mygo build   # 打包当前平台
```

### 发布

克隆后先启用一次工作流（agent 用的 GitHub API token 没有 workflow
权限，所以工作流放在 `setup/` 里）：

```sh
sh setup/install-ci.sh
```

推送 `v*` 标签，GitHub Actions 会自动并行构建 macOS / Windows /
Linux 三平台安装包并上传到草稿 Release（见
`.github/workflows/release.yml`）。

### 已知限制（v0.1）

- go-latex 暂不支持 TeX 上下标（`^`、` _`）：简单公式转写为 Unicode
  （E=mc²），复杂公式显示源码
- mermaid 的 WASM 渲染器与官方不完全像素一致，`%%{init}%%` 指令被忽略
- 图表在两个主题下都渲染在浅色卡片上（刻意设计，与 GitHub 一致）
- MyGo 尚未对原生 UI 窗口提供打印接口，请使用导出 PDF

---

<p align="center">Made with ❤️ by <a href="https://github.com/Maidang1">Maidang1</a></p>
