# Corra Markdown 产品文案统一设计

于 2026-10-17 在对话中确认方向：采用“专业文档编辑器”定位，以“文档 / 文档库”替代产品语境中的“备忘录 / notes”，保留现有交互与默认存储目录。

## 背景

Corra 已具备本地文档库、Markdown 源码与原生富文本编辑、GFM、数学公式、图表、图片和多格式导出能力，但菜单、空状态、欢迎文档和 README 仍沿用 Apple Notes 式“备忘录”定位。现有文案既低估了产品能力，也存在“两栏 / 三栏 / 单栏”等布局口径不一致的问题。

本次工作只统一产品表达，不重新设计信息架构或编辑交互。

## 目标

1. 将 Corra 明确表达为纯原生 Markdown 桌面编辑器，而不是备忘录应用。
2. 统一中文和英文中单个内容实体、内容集合及根目录的称谓。
3. 让按钮、菜单、空状态和错误提示准确描述实际操作对象。
4. 同步应用内文案、欢迎文档、默认新文档文案和现行使用文档，避免不同入口定位冲突。
5. 保持文件、数据、内部 API 和 Markdown 技术语法兼容。

## 产品定位

中文产品描述采用以下语义，可按所在文档的篇幅换行，但不改变核心口径：

> Corra 是一款纯原生 Markdown 桌面编辑器，提供文档库、所见即所得与源码编辑，并支持 GFM、数学公式、图表及多格式导出。

英文 README 使用等价的 native Markdown desktop editor 定位。无需在每处反复加入“专业”等宣传性形容词；专业性通过准确、克制、一致的文案体现。

布局统一描述为“文档库与 Markdown 编辑区”或 document library and Markdown editing workspace，不再用 Apple Notes 定义产品，也不再写死两栏或三栏。

## 术语规范

| 场景 | 中文 | 英文 |
| --- | --- | --- |
| 单个内容实体 | 文档；需要强调格式时用 Markdown 文档 | document；需要强调格式时用 Markdown document |
| 内容集合及本地存储体系 | 文档库 | document library |
| 库根目录 | 文档库根目录；空间受限时用库根目录 | document library root / library root |
| 全部内容入口 | 全部文档 | All Documents |
| 当前内容的操作入口 | 文档选项 | Document Options |
| 空白既有文档的标题回退 | 未命名文档 | Untitled Document |
| 新建文档的默认标题 | 新文档 | New Document |

“文件夹”继续表示文档库内的实际目录，不改成“文档”。“最近删除”继续表示现有回收站视图，不改变其名称或 `.trash` 存储实现。“搜索”输入框已有的简短 placeholder 保持不变，辅助功能标签使用“搜索文档”。

## 界面文案

### 菜单、工具栏和侧栏

- “新建备忘录”改为“新建文档”。
- “删除所选备忘录”改为“删除所选文档”。
- “搜索备忘录”改为“搜索文档”。
- “全部备忘录”改为“全部文档”，包括由它派生的辅助功能标签。
- “N 个备忘录”改为“N 个文档”。
- “备忘录选项”改为“文档选项”。
- “在我的 Mac 上”改为“本地文档库”，避免在 Windows 和 Linux 上出现平台错误。
- “刷新备忘录”改为“刷新文档库”，因为实际行为是重新扫描整个库。

### 文档操作与状态

- 移动、恢复和删除菜单统一以“文档”为对象。
- 移动到库根目录时显示“文档库根目录”或“库根目录”，不显示容易误解为单篇文档的“备忘录”。
- 删除确认分别使用“删除文档？”和“永久删除文档”的明确表述，保留现有确认流程。
- “未选择备忘录”改为“未选择文档”。
- “无备忘录”改为“暂无文档”。
- “无匹配的备忘录”改为“没有找到匹配的文档”。
- “无附加文本”改为“暂无正文”。
- “请先保存当前备忘录”改为“请先保存当前文档”。
- “已删除的备忘录 · 恢复后可编辑”改为“已删除的文档 · 恢复后可编辑”。
- 保存失败及新建失败等提示使用“当前文档”或“文档”，库配置问题使用“文档库目录”。

辅助功能标签与可见文案同步更新，包括列表行“文档：<标题>”和编辑器“文档正文”。

## 欢迎文档与默认新文档

`app/state.go` 中的欢迎文档保留快速导览作用，但开场改为原生 Markdown 编辑器定位，不再称为 Markdown 备忘录或 Apple Notes 风格产品。布局说明使用“文档库与编辑区”，日期、列表和菜单说明统一使用“文档”。

`app/library.go` 中：

- 新建文件默认名称改为“文档 <时间>.md”，只影响以后新建的文件。
- 新建文件初始标题改为“# 新文档”。
- 无法从既有空白文件提取标题时，列表回退为“未命名文档”。
- 不重命名或改写用户已有文件。

## 错误文案

生产代码中会进入保存错误、文件夹错误或系统对话框的英文错误，使用语法完整、语义准确的 document / document library 表述，并保留原始路径、错误包装、动态上下文和 `%s` 占位符。复合词按完整文案修改，不做机械单词替换：

- `not a notes folder: %s` → `not a document library folder: %s`
- `restore requires a deleted note; move requires an active note` → `restore requires a deleted document; move requires an active document`
- `notes library is not configured` → `document library is not configured`
- `symbolic links are not managed notes paths: %s` → `symbolic links are not managed document library paths: %s`
- `not a Markdown note: %s` → `not a Markdown document: %s`
- `note is outside the library: %s` → `document is outside the library: %s`
- `not a regular note file: %s` → `not a regular document file: %s`
- `note changed during move: %s` → `document changed during move: %s`
- `cannot save a non-regular note: %s` → `cannot save a non-regular document: %s`
- `note changed during save: %s` → `document changed during save: %s`

不借此次文案工作重构错误体系或新增本地化框架。生产代码中尚未启用但明确面向用户的附件保存提示也从“先保存备忘录，再插入附件”改为“先保存文档，再插入附件”，避免未来重新出现旧术语。

## README 与现行使用文档

中英文 README 同步调整产品简介、功能、存储和操作说明：

- Markdown notes app / Markdown 备忘录应用改为 native Markdown desktop editor / 原生 Markdown 桌面编辑器。
- Apple Notes 式布局改为文档库与编辑工作区的描述。
- notes、note menu、new note 等操作说明改为 documents、document menu、new document 等。
- “附件保存到笔记旁”改为“附件保存到当前文档旁”。
- 存储章节说明文档就是普通 Markdown 文件，不再以 Apple Notes 作为产品定义。
- 保留默认路径字面量 `<config>/备忘录`，因为它仍是实际兼容路径，不是需要隐藏的旧定位。
- `docs/typora-parity.md` 中现行功能的“备忘录”描述同步改为“文档”。

历史规格、历史实施计划和 `verifier/runs/` 归档保持原样。

## 明确边界

- 不修改 `<config>/备忘录` 默认目录，不增加迁移、回退或旧目录探测逻辑。
- 不修改 `.trash`、文件关联、存储格式、自动保存和任何交互行为。
- 不重命名 `Note`、`Library.Notes`、`CreateNote`、`views/notes.go` 等内部 API、类型、函数或文件。
- 不修改 GitHub `[!NOTE]` / Note 提示块、sequence.js 的 `Note over/left of/right of`、release notes 参数及其他技术语境中的 note。
- `main.go` 中 `seed note:` 和 `open note:` 是启动过程的 stderr 诊断日志，不是 GUI 产品文案，本次保持不变。
- 不重写 `docs/superpowers/` 中的历史规格和计划，也不修改历史验收日志。
- 不为了本次修改引入统一文案 catalog 或 i18n 架构；沿用项目当前的内联文案方式。

## 主要影响位置

- `menu.go`：全局文件、编辑和视图菜单。
- `views/header.go`：数量、新建提示和文档选项。
- `views/notes.go`：侧栏、搜索、列表、空状态和新建错误。
- `views/commands.go`：文档/文件夹菜单及删除确认。
- `views/editorpane.go`：编辑区空状态、只读提示、辅助功能和保存错误。
- `app/state.go`：首次启动欢迎文档。
- `app/library.go`：标题回退、新文件名和初始正文。
- `app/notes.go`、`app/folders.go`、`app/library_paths.go`、`app/attachment_paths.go`、`app/save.go`、`app/attach.go`：用户可见错误和提示。
- `README.md`、`docs/typora-parity.md`：现行产品和使用说明。

## 验证

1. 更新通过旧可见文本或辅助功能标签定位控件的测试，尤其是 `views/interactions_test.go` 和 `views/zz_click_probe_test.go`。
2. 更新 `app/library_test.go` 的标题断言：既有空白文档回退为“未命名文档”，新建设置显示“新文档”；`main_test.go` 中默认目录断言保持“备忘录”不变。
3. 为新建文件增加直接验收：断言 basename 以“文档 ”开头、保留现有时间格式并以 `.md` 结尾；直接读取文件并断言初始标题为“# 新文档”，避免只检查扫描后的列表标题而遗漏旧文件名或旧正文。
4. 如截图测试夹具继续用于视觉验收，同步其中的现行产品文案，但不改历史归档。
5. 对修改的 Go 文件执行 gofmt。
6. 运行 `go test ./...` 和 `go build ./...`。
7. 分范围检查欢迎文档、生产用户可见字符串、README 和 `docs/typora-parity.md` 中的“备忘录”、“笔记”、`Apple Notes` 以及英文单词 `note` / `notes` 残留。只允许默认存储路径字面量、技术语法、release notes、内部标识符、注释、诊断日志和明确的历史记录保留，不做跨仓库全局替换。
8. 检查 Note/note 修改没有触及 GitHub 提示块、sequence.js、内部 Go API 或 release 工作流。
