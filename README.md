# Corra

**English** | [中文](#中文)

---

## English

Corra is a Markdown **notes app** with a **fully native UI**, written in
Go with the [MyGo](https://mygo.egoist.dev) toolkit. No webview, no HTML,
no JavaScript. Go draws the shell on the GPU; macOS uses an AppKit
`NSTextView` for rich-text editing.

### Highlights

- **Apple Notes-style three-column layout** — folders in the sidebar,
  notes grouped by date (Today / Yesterday / Previous 7 / 30 days) in
  the list, a dated editor column on the right
- **Native WYSIWYG on macOS** — edit headings, body text, inline styles and
  checklists directly, with native selection, clipboard and undo/redo. Notes
  still autosave as Markdown files. Requires a cgo-enabled macOS build.
- **Source and reading modes** — `Cmd+E` switches rich text / Markdown source;
  the note menu also offers a full reading preview. Other platforms retain
  source / preview editing. Source mode has syntax highlighting, multiple
  carets and find & replace (Cmd+F).
- **GFM complete** — tables, task lists, strikethrough, footnotes
- **Typora-grade inline styles** — underline `<u>`, highlight `==x==`,
  superscript `^x^`, subscript `~x~`, `<kbd>`, emoji `:codes:`, YAML front
  matter, `[TOC]` outline blocks, clickable links (web / #anchors / .md)
- **Math** — inline `$...$` and `$$...$$` blocks, typeset by go-latex
- **Diagrams** — ` ```mermaid ` (Rust/WASM renderer), ` ```d2 `,
  ` ```goat ` ASCII diagrams, rasterized in pure Go with an embedded
  font for labels
- **Data charts** — ` ```chart ` YAML blocks (line / bar / hbar / pie),
  rendered natively
- **Syntax highlighting** — 200+ languages via chroma, themed light & dark
- **Images like Typora** — insert at the caret, paste from the clipboard,
  drag & drop into the window; size with `=300x200`, `=50%` or `<img width>`;
  attachments save to an `assets/` folder next to the note
- **GitHub image hosting** — insert an image and it is uploaded to a
  GitHub repo via the Contents API; a CDN link (jsDelivr by default,
  configurable) is inserted. The token never leaves your machine (0600
  file)
- **Export PDF / HTML / Word…** — PDF embeds document images and honors
  page-break markers; HTML is standalone with KaTeX/mermaid CDN; docx/EPUB/
  LaTeX/RTF/ODT and imports run through pandoc when it is installed
- **Copy as HTML / Markdown**, smart paste from web pages (HTML → Markdown)
- **Writer comforts** — word count (CJK-aware, reading minutes), outline
  panel with jump-to-heading, focus mode, typewriter mode, spell checking,
  auto-pairing, smart punctuation, code-fence line numbers, heading
  auto-numbering, native printing on macOS
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
go test ./...           # includes AppKit harness on macOS (no AX permission)
make dev               # macOS: enables cgo for the native rich editor
make build             # package for the current platform
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

### Known limits (v0.2)

- go-latex cannot yet typeset TeX scripts (`^`, `_`); simple ones are
  transliterated to Unicode (E=mc²); complex formulas show their source.
  MathJax extensions (mhchem chemistry, equation auto-numbering) need a JS
  engine and are out of scope
- The mermaid WASM renderer is not pixel-identical to upstream mermaid;
  `%%{init}%%` directives are ignored
- Diagrams render on a light card in both themes (by design, like GitHub)
- Custom CSS themes and embedded web content (iframe/video) need a webview,
  which Corra deliberately does not have
- Printing uses the native macOS panel; on Windows/Linux use Export PDF
  (MyGo has no print API there)
- PDF bookmarks (outline entries) need gpdf support that does not exist yet
- See docs/typora-parity.md for the full Typora comparison

---

## 中文

Corra 是一个**纯原生 UI** 的 Markdown 备忘录应用，用 Go 基于
[MyGo](https://mygo.egoist.dev) 工具包编写：无 webview、无 HTML、无
JavaScript。Go 在 GPU 上绘制外壳，macOS 编辑区使用 AppKit 原生富文本控件。

### 功能

- **类 Apple Notes 三栏布局**：左栏备忘录文件夹，中栏按日期分组
  （今天/昨天/最近 7 天/最近 30 天）的列表，右栏带日期的编辑区
- **macOS 原生所见即所得**：直接编辑标题、正文、文字格式和任务清单，
  使用原生选区、剪贴板和撤销重做，自动保存为 Markdown 文件。需启用 cgo。
- **源码和阅读视图**：`Cmd+E` 切换富文本 / Markdown 源码；备忘录菜单
  可打开完整阅读预览。其他平台保留源码 / 预览。源码模式仍支持语法高亮、
  多光标和查找替换（Cmd+F）。
- **GFM 全要素**：表格、任务列表、删除线、脚注
- **Typora 级内联样式**：下划线 `<u>`、高亮 `==x==`、上标 `^x^`、下标 `~x~`、
  `<kbd>`、emoji `:code:`、YAML 头部信息、`[TOC]` 目录块、可点击链接
  （网址 / #标题锚点 / 相对 .md 文件）
- **数学公式**：行内 `$...$` 与块级 `$$...$$`，go-latex 排版
- **图表**：` ```mermaid `（Rust/WASM 渲染）、` ```d2 `、` ```goat `
  ASCII 图，纯 Go 光栅化并内嵌字体绘制文字标签
- **数据图表**：` ```chart ` YAML 数据块（折线/柱状/条形/饼图）
- **语法高亮**：chroma 支持 200+ 语言，深浅色主题
- **图片体验对齐 Typora**：光标处插入、剪贴板粘贴、拖放入窗即插入；
  用 `=300x200`、`=50%` 或 `<img width>` 控制尺寸；附件保存到笔记旁
  的 `assets/` 文件夹
- **GitHub 图床**：插入图片即通过 Contents API 上传到指定仓库并回插
  CDN 链接（默认 jsDelivr，可自定义反代）；token 只存本机（0600 权限）
- **导出 PDF / HTML / Word 等**：PDF 嵌入正文图片并支持分页符；HTML 为
  独立页面（KaTeX/mermaid CDN）；docx/EPUB/LaTeX/RTF/ODT 与反向导入
  经由 pandoc（检测到才可用）
- **复制为 HTML / Markdown**，网页智能粘贴（HTML → Markdown）
- **写作体验**：字数统计（中英混排、阅读时长）、大纲面板点击跳转、
  专注模式、打字机模式、拼写检查、括号自动配对、智能标点、代码块行号、
  标题自动编号、macOS 原生打印
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
make dev            # macOS 自动启用 cgo，使用原生富文本编辑器
make build          # 打包当前平台
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

### 已知限制（v0.2）

- go-latex 暂不支持 TeX 上下标（`^`、` _`）：简单公式转写为 Unicode
  （E=mc²），复杂公式显示源码；mhchem 化学式、公式自动编号等 MathJax
  扩展依赖 JS 引擎，不在范围内
- mermaid 的 WASM 渲染器与官方不完全像素一致，`%%{init}%%` 指令被忽略
- 图表在两个主题下都渲染在浅色卡片上（刻意设计，与 GitHub 一致）
- 自定义 CSS 主题与 iframe/视频嵌入需要 webview，Corra 刻意不含
- Windows/Linux 使用导出 PDF 代替打印（MyGo 暂无打印接口）
- PDF 书签（大纲）需 gpdf 提供 API，暂不支持
- 与 Typora 的完整能力对照见 docs/typora-parity.md

---

<p align="center">Made with ❤️ by <a href="https://github.com/Maidang1">Maidang1</a></p>

### Local Notes workflow

The unified toolbar provides format (`Aa`), checklist, table, image attachment,
copy Markdown and PDF export. The note `...` menu contains edit/preview, move,
delete, restore and appearance options; the folder `...` menu creates/renames
folders and deletes empty ones. Search matches the full Markdown source.
Deleted notes are read-only until restored. Restore currently places the note
in the library root, adding a suffix when a filename already exists.

- `Cmd/Ctrl+N`: new note (native rich editor on macOS)
- `Cmd/Ctrl+E`: source editor / rich document (preview on other platforms)
- `Cmd/Ctrl+Shift+P`: full reading preview
- `Cmd/Ctrl+Shift+F`: search notes; `Cmd/Ctrl+F`: find within the source editor
- `Cmd/Ctrl+Alt+S`: collapse / expand folders
- `Cmd/Ctrl+B`, `Cmd/Ctrl+I`: bold / italic; `Cmd/Ctrl+Shift+L`: checklist
- `Cmd/Ctrl+Shift+N`: new folder

Both source edits and preview checkboxes autosave. Save failures are displayed
above the note and prevent switching/closing; **Save As** remains available to
recover the current buffer to another location. Existing files are saved via
same-directory temporary files and atomic replacement, with best-effort external
change checks (there is no cross-process compare-and-swap guarantee).

For an isolated development session, set `CORRA_CONFIG_DIR` to a temporary
folder before starting the app. This does not migrate or change the normal library.

### 本地备忘录操作

工具栏提供 `Aa` 格式、清单、表格、图片，以及复制 Markdown / 导出 PDF。
右侧 `…` 菜单可切换编辑和预览、移动、删除、恢复备忘录及切换外观；
文件夹标题旁的 `…` 可新建、重命名和删除空文件夹。搜索覆盖完整正文。
「最近删除」中的备忘录只读，恢复后放回库根目录，同名时自动添加编号。

编辑和预览勾选均自动保存；保存失败时会显示错误并阻止切换/关闭，
可用「另存为」将当前内容保存到其他位置。macOS 默认原生富文本编辑，
Markdown 源码仍可随时切换；不提供 iCloud 同步、Apple 账户或多人协作功能。

### Rich Editor Boundaries / 富文本编辑范围

The native editor supports simple paragraphs, headings, bold/italic/strike/inline
code, links, quotes, single-level lists and checklists. Complex Markdown (tables,
images, code fences, math, diagrams, nested lists, HTML, references) is kept as a
protected source block, not flattened into text. Use its source action to edit,
or **Reading preview** to see the full renderer. This is not full Apple Notes
feature parity: in-place table/image editing and iCloud collaboration are not
implemented. Unedited blocks preserve their original bytes; edited supported
paragraphs may normalize Markdown spelling.

原生编辑器支持普通段落、标题、粗体/斜体/删除线/行内代码、链接、引用、
单层列表和清单。表格、图片、代码块、公式、图表、嵌套列表等复杂内容以
代码块隐藏围栏后直接编辑（输入 ``` 回车创建），公式和图片就地渲染为
图片，表格在编辑区内直接编辑；图表等复杂内容以源码行编辑，通过「阅读
预览」查看完整渲染。暂不支持图片的可视化编辑，也不包含 iCloud 协作。
未编辑的块按原样保存，编辑过的普通段落可能规范化 Markdown 标记。

Use `make dev` / `make build`, or explicitly run
`CGO_ENABLED=1 go tool mygo dev` / `CGO_ENABLED=1 go tool mygo build` on macOS.
MyGo defaults to disabling cgo when the environment variable is unset; direct
commands without it use the source/preview fallback. AppKit harness tests exercise
real native text storage without Accessibility permission; interactive Chinese
input-method candidate selection still needs manual validation.
