# Corra

**English** | [中文](#中文)

---

## English

Corra is a **Markdown desktop editor** with a **fully native UI**, written in
Go with the [MyGo](https://mygo.egoist.dev) toolkit. No webview, no HTML,
no JavaScript. Go draws the shell on the GPU; macOS uses an AppKit
`NSTextView` for rich-text editing.

### Highlights

- **Document library and editing workspace** — folders and a
  document list grouped by modification date in the sidebar, with an editor
  showing the document's modification date and an optional outline
- **Native WYSIWYG on macOS** — edit headings, body text, inline styles and
  checklists directly, with native selection, clipboard and undo/redo. Documents
  still autosave as Markdown files. Requires a cgo-enabled macOS build.
- **Source and reading modes** — `Cmd+E` switches rich text / Markdown source;
  the document menu also offers a full reading preview. Other platforms retain
  source / preview editing. Source mode has syntax highlighting, multiple
  carets and find & replace (Cmd+F).
- **GFM complete** — tables, task lists, strikethrough, footnotes
- **Typora-grade inline styles** — underline `<u>`, highlight `==x==`,
  superscript `^x^`, subscript `~x~`, `<kbd>`, emoji `:codes:`, YAML front
  matter, `[TOC]` outline blocks, clickable links (web / #anchors / .md)
- **Math** — inline `$...$` and `$$...$$` blocks, typeset offline by embedded
  MathJax; native editing supports chemistry, physics, shared macros, references
  and equation numbering without a WebView or runtime Node dependency
- **Diagrams** — ` ```mermaid ` (Rust/WASM renderer), ` ```d2 `,
  ` ```goat ` ASCII diagrams, rasterized in pure Go with an embedded
  font for labels
- **Data charts** — ` ```chart ` YAML blocks (line / bar / hbar / pie),
  rendered natively
- **Syntax highlighting** — 200+ languages via chroma, themed light & dark
- **Images like Typora** — insert at the caret, paste from the clipboard,
  drag & drop into the window; size with `=300x200`, `=50%` or `<img width>`;
  attachments save to an `assets/` folder next to the document
- **GitHub image hosting** — insert an image and it is uploaded to a
  GitHub repo via the Contents API; a CDN link (jsDelivr by default,
  configurable) is inserted. The token never leaves your machine (0600
  file)
- **Export PDF / HTML / Word…** — the share menu and command palette open an
  export panel with one described card per format; PDF embeds document
  images and honors page-break markers; HTML is standalone with
  KaTeX/mermaid CDN; docx/EPUB/LaTeX/RTF/ODT and imports run through pandoc
  when it is installed (the panel says how to install it otherwise)
- **Copy as HTML / Markdown**, smart paste from web pages (HTML → Markdown)
- **Quick open / command palette** — `⌘⇧P` filters the library by title,
  folder or path; an empty query groups recently opened notes above the rest;
  type `>` to switch to actions (new note, insert table,
  export, theme, settings…) with their menu shortcuts; Return runs the
  highlighted row and Escape dismisses
- **Link completion** — type `[[` before a note name to search the library;
  the list shows titles and relative paths, Return inserts a Markdown link to
  that note (`[Beta](sub/beta.md)`), and the path stays clickable; `[[#` lists
  the current note's headings and inserts an anchor link (`[Setup](#setup)`)
- **Emoji picker** — type `:` at a word start for a shortcode list (`:smi` →
  😄); keep typing to filter, arrows to browse, Return to insert, Escape to
  dismiss
- **Slash menu** — type `/` on an empty line for a Notion-style block picker
  (headings, lists, quotes, callouts, code, math, tables, TOC, footnotes,
  images, page breaks), grouped into sections; filter by name or keyword,
  navigate with arrows, confirm with Return, dismiss with Escape; the
  matched part of a title is highlighted as you filter
- **Selection toolbar** — selecting prose raises a floating pill with bold,
  italic, underline, strike, inline code, highlight, link and clear;
  active traits are highlighted, and the link prompt accepts or removes a URL
- **Block actions** — move a paragraph, list item, code block or table past
  its neighbour (Cmd+Shift+↑/↓), duplicate it (Cmd+D), delete it
  (Cmd+Shift+⌫), or turn it into another block kind from the context menu;
  a six-dot handle in the left margin opens that menu, and dragging the handle
  reorders blocks with a live insertion line (one undo step)
- **Code block comforts** — hover a fenced block for a one-click copy button
  (with a checkmark flash) and a clickable language tag that swaps the fence
  language; the tag makes room for the button
- **Image controls** — hover an image for a size pill (original / 25 / 50 /
  75 / 100%); the current size is highlighted, and copy / delete buttons sit
  after a divider so the image can be managed without selecting its block
- **Document stats** — click the word count in the status bar for a strip of
  chips: words, characters, lines, headings, links, images, tasks, code
  blocks and reading time, counted outside fenced code
- **Table controls** — hovering a table raises a "+" under it (add row) and
  at its right edge (add column), so a table grows without the context menu
- **Footnote preview** — hovering a footnote reference raises a readable card
  with its definition instead of waiting for the system tooltip; ⌘⌥↓/⌘⌥↑ jump
  between a reference and its definition, and ⌘⌥← returns to where the last
  jump started (press again to toggle)
- **Writer comforts** — word count (CJK-aware, reading minutes), outline
  panel with jump-to-heading, focus mode, typewriter mode, spell checking,
  auto-pairing, smart punctuation, code-fence line numbers, heading
  auto-numbering, native printing on macOS
- **Deep native integration** — system dark mode, drag & drop, file
  associations, external-change detection with a conflict dialog, global
  menu with accelerators
- **Preferences (Cmd+,)** — appearance (system / light / dark), body and
  code typefaces, and the writing aids in one settings panel, persisted
  across launches

### Storage

Documents live in a plain folder of Markdown files
(`<config>/备忘录` by default; each subfolder appears as a folder in the sidebar).
Anything that edits the files on disk (Sync, git, other editors) shows up in
the app; the app autosaves back to the same files.

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

- Math uses embedded MathJax 3.2.2 with mhchem and physics. Unsupported expressions
  remain editable as source. Document-wide numbering, macros and references are
  connected to native editing and HTML export, plus display equations in reading
  preview and PDF. Inline equation images in reading preview and PDF remain pending
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

Corra 是一个**纯原生 UI** 的 Markdown 桌面编辑器，用 Go 基于
[MyGo](https://mygo.egoist.dev) 工具包编写：无 webview、无 HTML、无
JavaScript。Go 在 GPU 上绘制外壳，macOS 编辑区使用 AppKit 原生富文本控件。

### 功能

- **文档库与编辑工作区**：侧栏提供文件夹和按修改日期分组的文档列表，
  编辑区显示文档修改时间，并可开启大纲
- **macOS 原生所见即所得**：直接编辑标题、正文、文字格式和任务清单，
  使用原生选区、剪贴板和撤销重做，自动保存为 Markdown 文件。需启用 cgo。
- **源码和阅读视图**：`Cmd+E` 切换富文本 / Markdown 源码；文档菜单
  可打开完整阅读预览。其他平台保留源码 / 预览。源码模式仍支持语法高亮、
  多光标和查找替换（Cmd+F）。
- **GFM 全要素**：表格、任务列表、删除线、脚注
- **Typora 级内联样式**：下划线 `<u>`、高亮 `==x==`、上标 `^x^`、下标 `~x~`、
  `<kbd>`、emoji `:code:`、YAML 头部信息、`[TOC]` 目录块、可点击链接
  （网址 / #标题锚点 / 相对 .md 文件）
- **数学公式**：行内 `$...$` 与块级 `$$...$$`，离线 MathJax 排版；原生编辑区
  支持化学式、物理宏、跨公式宏与引用、公式编号，无需 WebView 或运行时 Node
- **图表**：` ```mermaid `（Rust/WASM 渲染）、` ```d2 `、` ```goat `
  ASCII 图，纯 Go 光栅化并内嵌字体绘制文字标签
- **数据图表**：` ```chart ` YAML 数据块（折线/柱状/条形/饼图）
- **语法高亮**：chroma 支持 200+ 语言，深浅色主题
- **图片体验对齐 Typora**：光标处插入、剪贴板粘贴、拖放入窗即插入；
  用 `=300x200`、`=50%` 或 `<img width>` 控制尺寸；附件保存到当前文档旁
  的 `assets/` 文件夹
- **GitHub 图床**：插入图片即通过 Contents API 上传到指定仓库并回插
  CDN 链接（默认 jsDelivr，可自定义反代）；token 只存本机（0600 权限）
- **导出 PDF / HTML / Word 等**：分享菜单与命令面板打开导出面板，每种格式
  一张说明卡片；PDF 嵌入正文图片并支持分页符；HTML 为独立页面
  （KaTeX/mermaid CDN）；docx/EPUB/LaTeX/RTF/ODT 与反向导入经由 pandoc
  （未安装时面板会给出安装提示）
- **复制为 HTML / Markdown**，网页智能粘贴（HTML → Markdown）
- **快速打开 / 命令面板**：`⌘⇧P` 按标题/文件夹/路径即时筛选文档，空查询时
  顶部按「最近打开 / 全部文档」分组；输入 `>`
  切换为命令模式（新建、插入表格、导出、主题、设置…并显示对应快捷键），
  回车执行首项，Esc 关闭，不会改动当前文档
- **文档链接补全**：输入 `[[` 即可按标题检索文档库，列表显示标题与相对路径，
  回车插入指向该文档的 Markdown 链接（`[Beta](sub/beta.md)`），可直接点击跳转；
  `[[#` 列出当前文档的标题并插入锚点链接（`[Setup](#setup)`）
- **表情选择器**：在词首输入 `:` 打开 shortcode 列表（`:smi` → 😄），
  继续输入即过滤，方向键浏览，回车插入，Esc 关闭
- **斜杠菜单**：在空行输入 `/` 打开 Notion 风格块选择面板（标题、列表、
  引用、提示卡片、代码、公式、表格、目录、脚注、图片、分页符），按标题分组，
  按名称或关键词过滤（命中的标题片段会高亮），方向键选择，回车应用，Esc 关闭
- **选区浮动工具栏**：选中正文即浮出格式药丸（粗体、斜体、下划线、删除线、
  行内代码、高亮、链接、清除格式），已生效格式会高亮，链接支持输入或移除
- **块操作**：段落、列表项、代码块、表格整体上移 / 下移（Cmd+Shift+↑/↓）、
  复制（Cmd+D）、删除（Cmd+Shift+⌫），右键菜单还可「转换为」其他块类型；
  悬停时左边距出现六点块手柄，点开是同一套块菜单，拖动即可重排（带插入
  指示线，一次撤销可还原）
- **代码块体验**：悬停围栏代码块出现一键复制按钮（带勾选反馈），
  语言标签可点击切换语言，并会给复制按钮让位
- **图片控制**：悬停图片出现尺寸条（原图 / 25% / 50% / 75% / 100%）、
  分隔线与复制/删除按钮，当前尺寸高亮，点击即改尺寸或管理图片
- **文档统计**：点击状态栏的字数展开统计条，显示字数、字符、行数、标题、
  链接、图片、任务、代码块与阅读时间（统计时跳过围栏代码）
- **表格控制**：悬停表格时下方出现「+」加行、右侧出现「+」加列，无需右键菜单
- **脚注预览**：悬停脚注引用即浮出定义卡片，无需等待系统提示；
  ⌘⌥↓/⌘⌥↑ 在引用与定义之间跳转，⌘⌥← 回到上次跳转的起点（再按一次往返切换）
- **写作体验**：字数统计（中英混排、阅读时长）、大纲面板点击跳转、
  专注模式、打字机模式、拼写检查、括号自动配对、智能标点、代码块行号、
  标题自动编号、macOS 原生打印
- **深度系统集成**：系统深浅色、拖放打开、文件关联、外部修改检测
  与冲突处理、全局菜单与快捷键
- **设置面板（Cmd+,）**：外观（跟随系统 / 浅色 / 深色）、正文与代码
  字体、写作辅助开关集中一处，并持久化保存

### 存储

文档以普通 Markdown 文件存放在文档库中（默认目录为
`<config>/备忘录`；库根目录下的每个子文件夹对应侧栏中的一个文件夹）。
磁盘上的文件被别的工具（同步、git、其他编辑器）修改会实时反映到应用里；
应用的自动保存也直接写回同一批文件。

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

- 数学排版使用内嵌 MathJax 3.2.2，含 mhchem 与 physics；不支持的表达式保留
  可编辑源码。文档级编号、宏和引用已接入原生编辑区、HTML 导出及阅读预览/
  PDF 的块公式；阅读预览和 PDF 的行内公式图仍待接入
- mermaid 的 WASM 渲染器与官方不完全像素一致，`%%{init}%%` 指令被忽略
- 图表在两个主题下都渲染在浅色卡片上（刻意设计，与 GitHub 一致）
- 自定义 CSS 主题与 iframe/视频嵌入需要 webview，Corra 刻意不含
- Windows/Linux 使用导出 PDF 代替打印（MyGo 暂无打印接口）
- PDF 书签（大纲）需 gpdf 提供 API，暂不支持
- 与 Typora 的完整能力对照见 docs/typora-parity.md

---

<p align="center">Made with ❤️ by <a href="https://github.com/Maidang1">Maidang1</a></p>

### Local document workflow

The unified toolbar provides format (`Aa`), checklist, table, image attachment,
copy Markdown and PDF export. The document `...` menu contains edit/preview, move,
delete, restore and appearance options; the folder `...` menu creates/renames
folders and deletes empty ones. Search matches the full Markdown source.
Deleted documents are read-only until restored. Restore currently places the document
in the library root, adding a suffix when a filename already exists.

- `Cmd/Ctrl+N`: new document (native rich editor on macOS)
- `Cmd/Ctrl+E`: source editor / rich document (preview on other platforms)
- `Cmd/Ctrl+Shift+P`: full reading preview
- `Cmd/Ctrl+Shift+F`: search documents; `Cmd/Ctrl+F`: find within the source editor
- `Cmd/Ctrl+Alt+S`: collapse / expand folders
- `Cmd/Ctrl+B`, `Cmd/Ctrl+I`: bold / italic; `Cmd/Ctrl+Shift+L`: checklist
- `Cmd/Ctrl+Shift+N`: new folder

Both source edits and preview checkboxes autosave. Save failures are displayed
above the document and prevent switching/closing; **Save As** remains available to
recover the current buffer to another location. Existing files are saved via
same-directory temporary files and atomic replacement, with best-effort external
change checks (there is no cross-process compare-and-swap guarantee).

For an isolated development session, set `CORRA_CONFIG_DIR` to a temporary
folder before starting the app. This does not migrate or change the normal library.

### 本地文档操作

工具栏提供 `Aa` 格式、清单、表格、图片，以及复制 Markdown / 导出 PDF。
右侧 `…` 菜单可切换编辑和预览、移动、删除、恢复文档及切换外观；
文件夹标题旁的 `…` 可新建、重命名和删除空文件夹。搜索覆盖完整正文。
「最近删除」中的文档只读，恢复后放回库根目录，同名时自动添加编号。

编辑和预览勾选均自动保存；保存失败时会显示错误并阻止切换/关闭，
可用「另存为」将当前内容保存到其他位置。macOS 默认原生富文本编辑，
Markdown 源码仍可随时切换；不提供 iCloud 同步、Apple 账户或多人协作功能。

### Rich Editor Boundaries / 富文本编辑范围

The native editor supports simple paragraphs, headings, bold/italic/strike/inline
code, titled/resolved reference links, quotes, nested lists and checklists.
Tab/Shift+Tab and Cmd+]/Cmd+[ indent/outdent list items with their descendants;
empty nested items outdent on Return. Native tables provide row/column insertion,
deletion, movement and alignment in their context menu. Cmd+Return inserts a row,
Cmd+Shift+Backspace deletes a body row, and tab-separated clipboard data pastes as
a grid. Code block selections indent/outdent without replacing their contents.
Typing highlight, superscript, subscript and common inline HTML updates the rich
format immediately. Click an image to edit its Markdown in place.

Click styled inline text or move into it with arrow keys to edit its Markdown
in place. Leaving the fragment, blurring the editor or pressing Cmd+Return folds
it back into native text. This includes links and table-cell formats, with UTF-16
caret/selection mapping and undo/redo. Merely revealing syntax keeps the stored
paragraph's original spelling; edited fragments use the existing serializer.

Native code blocks offer a context-menu language chooser and optional logical
line numbers. Newlines, selected replacements and multiline plain-text paste
preserve fence spelling, line endings and undo/redo. Replacements crossing a
code-block boundary are currently rejected to protect hidden fences.

Multi-paragraph list items and paragraph-only nested quotes are editable natively.
Option+Return adds a paragraph inside a list item. Lists/quotes containing other
block types, footnotes and HTML/media can still require source editing. Math
preferences/export consistency and visual table/image drag handles do not yet
match Typora. Unedited blocks preserve
their original bytes; edited paragraphs may normalize Markdown spelling, including
resolved references becoming inline links while retaining their destinations and
titles. These changes do not add iCloud collaboration.

原生编辑器支持普通段落、标题、粗体/斜体/删除线/行内代码、链接、引用、
嵌套列表、多段落列表项和段落型嵌套引用。代码块隐藏围栏后直接编辑
（输入 ``` 回车创建），表格在编辑区内直接编辑并提供行列结构菜单。
公式、图片和支持的图表就地渲染，点击展开源码；公式修改后离开源码
或按 Cmd+Return 收起。“视图 → 公式编号”控制原生文稿的编号规则。
复杂混合容器、脚注、HTML/媒体和表格/图片拖拽交互仍待完善，不包含
iCloud 协作。
未编辑的块按原样保存，编辑过的普通段落可能规范化 Markdown 标记。

Use `make dev` / `make build`, or explicitly run
`CGO_ENABLED=1 go tool mygo dev` / `CGO_ENABLED=1 go tool mygo build` on macOS.
MyGo defaults to disabling cgo when the environment variable is unset; direct
commands without it use the source/preview fallback. AppKit harness tests exercise
real native text storage without Accessibility permission; interactive Chinese
input-method candidate selection still needs manual validation.
