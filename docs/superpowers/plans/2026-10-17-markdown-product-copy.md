# Markdown Product Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Corra 的现行产品表达从“备忘录 / notes app”统一为准确、克制的“Markdown 文档编辑器 / document library”语义，同时保持数据、交互和内部 API 兼容。

**Architecture:** 沿用项目现有的内联文案，不引入 i18n 或文案 catalog。按默认新文档、MyGo 界面、欢迎文档、应用错误和 Markdown 文档分层修改；内部 `Note` API、`.trash`、默认 `<config>/备忘录` 路径及 Markdown 技术语法保持不变。

**Tech Stack:** Go、MyGo UI、Go 标准库 `testing`、Markdown 文档。

**Spec:** `docs/superpowers/specs/2026-10-17-markdown-product-copy-design.md`

---

## File Structure

- `app/library.go`：既有空白文档的标题回退、新文档文件名与初始正文。
- `app/library_test.go`：标题、文件名和初始正文的回归测试。
- `menu.go`：全局菜单中的新建、删除和搜索文档。
- `views/header.go`：文档数量、新建按钮和文档选项。
- `views/notes.go`：文档库、搜索、列表、空状态和新建错误。
- `views/commands.go`：文档移动、恢复、删除和文档库刷新。
- `views/editorpane.go`：编辑器空状态、正文辅助功能、删除和保存状态。
- `views/interactions_test.go`、`views/zz_click_probe_test.go`：通过可见标签执行的真实 UI 交互测试。
- `app/state.go`：首次启动欢迎文档。
- `main_test.go`：欢迎文档定位和既有解析/往返验证。
- `app/notes.go`、`app/folders.go`、`app/library_paths.go`、`app/attachment_paths.go`、`app/save.go`、`app/attach.go`：生产错误和提示。
- `app/product_copy_test.go`：关键生产错误文案的聚焦回归测试，不替代现有文件系统测试。
- `README.md`、`docs/typora-parity.md`：现行产品定位与使用说明。
- `views/zz_layout_shot_test.go`、`nativerich/zz_typo_shot_darwin_test.go`：仍用于视觉验收的截图夹具。

不改 `main.go` 的默认库路径或 stderr 诊断日志，不改历史规格、历史计划和 `verifier/runs/`。

### Task 1: 默认新文档与标题回退

**Files:**
- Modify: `app/library.go:289-329`
- Test: `app/library_test.go:217-259`

- [ ] **Step 1: 先修改测试，明确新文档契约**

在 `app/library_test.go` 的 import 中加入 `regexp`。

将 `TestNoteMeta` 的空内容断言改为：

```go
title, _ = NoteMeta("")
if title != "未命名文档" {
	t.Fatalf("empty title = %q", title)
}
```

在 `TestCreateAndDeleteNote` 创建首个文件并成功 `os.Stat` 后，加入文件名和正文断言：

```go
namePattern := regexp.MustCompile(`^文档 \d{4}-\d{2}-\d{2} \d{2}\.\d{2}\.\d{2}\.md$`)
if got := filepath.Base(p); !namePattern.MatchString(got) {
	t.Fatalf("new document name = %q", got)
}
data, err := os.ReadFile(p)
if err != nil {
	t.Fatal(err)
}
if got, want := string(data), "# 新文档\n\n"; got != want {
	t.Fatalf("new document content = %q, want %q", got, want)
}
```

将同一测试中的扫描标题断言改为：

```go
if len(lib.Notes) != 1 || lib.Notes[0].Title != "新文档" {
	t.Fatalf("notes = %+v", lib.Notes)
}
```

不要修改测试中的 `Note`、`Notes` 等内部 API 名称。

- [ ] **Step 2: 运行测试并确认先失败**

Run:

```bash
go test ./app -run 'TestNoteMeta|TestCreateAndDeleteNote'
```

Expected: FAIL；至少出现旧的 `新备忘录` 标题或旧的 `备忘录 ...md` 文件名不符合新断言。

- [ ] **Step 3: 做最小实现**

只修改 `app/library.go` 的三个字符串：

```go
if title == "" {
	title = "未命名文档"
}
```

```go
name := fmt.Sprintf("文档 %s.md", time.Now().Format("2006-01-02 15.04.05"))
```

```go
_, writeErr := f.WriteString("# 新文档\n\n")
```

保留 `numberedName`、时间格式、权限、排他创建和已有文件；不要重命名旧文件。

- [ ] **Step 4: 运行聚焦测试和 app 包测试**

Run:

```bash
gofmt -w app/library.go app/library_test.go
go test ./app -run 'TestNoteMeta|TestCreateAndDeleteNote'
go test ./app
```

Expected: 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add app/library.go app/library_test.go
git commit -m "feat: name new Markdown documents consistently"
```

### Task 2: 应用内菜单、列表和编辑器文案

**Files:**
- Modify: `menu.go:59-162`
- Modify: `views/header.go:48-80`
- Modify: `views/notes.go:13-264`
- Modify: `views/commands.go:55-213`
- Modify: `views/editorpane.go:21-245`
- Test: `views/interactions_test.go`
- Test: `views/zz_click_probe_test.go`

- [ ] **Step 1: 先更新交互测试使用的目标标签**

在两个测试文件中只替换下列精确标签，保留测试函数名和内部 note 标识：

| 旧测试标签 | 新测试标签 |
| --- | --- |
| `全部备忘录` | `全部文档` |
| `文件夹：全部备忘录` | `文件夹：全部文档` |
| `搜索备忘录` | `搜索文档` |
| `新建备忘录 (Cmd+N)` | `新建文档 (Cmd+N)` |
| `未选择备忘录` | `未选择文档` |
| `备忘录选项` | `文档选项` |
| `恢复备忘录` | `恢复文档` |
| `备忘录正文` | `文档正文` |
| `删除备忘录…` | `删除文档…` |

`views/interactions_test.go` 中折叠侧栏的负向断言和恢复、删除、搜索、自动保存等流程继续测试原有行为，只更换定位标签。

- [ ] **Step 2: 运行 views 测试并确认旧实现无法通过**

Run:

```bash
go test ./views
```

Expected: FAIL；测试无法找到 `新建文档 (Cmd+N)`、`文档选项`、`全部文档` 等新标签。

- [ ] **Step 3: 修改全局菜单和工具栏**

`menu.go` 使用以下精确替换：

```go
item("新建文档", "CmdOrCtrl+N", func() { view.NewNoteAction() })
item("删除所选文档", "", func() { view.DeleteNoteAction() })
item("搜索文档", "CmdOrCtrl+Shift+F", func() { view.SearchAction() })
```

`views/header.go` 使用：

```go
ui.Text(c, itoa(len(v.App.VisibleNotes()))+" 个文档")
iconButton(c, iconCompose, "新建文档 (Cmd+N)", v.newNote)
iconButton(c, iconEllipsis, "文档选项", nil)
```

保留原有回调、快捷键和布局逻辑。

- [ ] **Step 4: 修改文档库、列表和空状态**

`views/notes.go` 按以下清单精确替换：

| 当前文案 | 目标文案 |
| --- | --- |
| `全部备忘录` | `全部文档` |
| `在我的 Mac 上` | `本地文档库` |
| `搜索备忘录` | `搜索文档` |
| `无备忘录` | `暂无文档` |
| `无匹配的备忘录` | `没有找到匹配的文档` |
| `备忘录：` | `文档：` |
| `无附加文本` | `暂无正文` |
| `请先保存当前备忘录` | `请先保存当前文档` |
| `新建备忘录失败` | `新建文档失败` |

保留 placeholder `搜索`、垃圾桶空状态 `最近删除为空`、内部 `noteRow` / `newNote` 名称及所有注释。

- [ ] **Step 5: 修改文档操作与编辑器状态**

`views/commands.go` 使用：

| 当前文案 | 目标文案 |
| --- | --- |
| `刷新备忘录` | `刷新文档库` |
| `恢复备忘录` | `恢复文档` |
| 移动到子菜单中的 `备忘录` | `文档库根目录` |
| `删除备忘录…` | `删除文档…` |
| `删除备忘录？` | `删除文档？` |
| `备忘录将移到最近删除，你可以在那里恢复。` | `文档将移到最近删除，你可以在那里恢复。` |
| `这会永久删除备忘录，无法恢复。` | `这会永久删除文档，无法恢复。` |

`views/editorpane.go` 使用：

| 当前文案 | 目标文案 |
| --- | --- |
| `未选择备忘录` | `未选择文档` |
| `新建备忘录` | `新建文档` |
| `已删除的备忘录 · 恢复后可编辑` | `已删除的文档 · 恢复后可编辑` |
| `备忘录正文` | `文档正文` |
| `当前备忘录无法保存` | `当前文档无法保存` |

不要修改 GitHub 提示块菜单中的 `Note`。

- [ ] **Step 6: 运行 UI 回归测试**

Run:

```bash
gofmt -w menu.go views/header.go views/notes.go views/commands.go views/editorpane.go views/interactions_test.go views/zz_click_probe_test.go
go test ./views
go test . -run TestBuildMenuSmoke
```

Expected: 全部 PASS；既有布局、点击、删除/恢复、搜索和自动保存行为不变。

- [ ] **Step 7: Commit**

```bash
git add menu.go views/header.go views/notes.go views/commands.go views/editorpane.go views/interactions_test.go views/zz_click_probe_test.go
git commit -m "feat: align UI copy with Markdown documents"
```

### Task 3: 首次启动欢迎文档

**Files:**
- Modify: `app/state.go:123-161`
- Test: `main_test.go:53-68`

- [ ] **Step 1: 添加欢迎文档定位回归测试**

在 `main_test.go` 的 import 中加入 `strings`，保留现有 `TestWelcomeNoteParsesAndRoundTrips`，新增：

```go
func TestWelcomeUsesMarkdownEditorPositioning(t *testing.T) {
	for _, want := range []string{
		"原生 Markdown 桌面编辑器",
		"## 文档库与编辑区",
		"全部文档",
		"文档修改时间",
	} {
		if !strings.Contains(app.Welcome, want) {
			t.Errorf("welcome does not contain %q", want)
		}
	}
	for _, legacy := range []string{"备忘录", "笔记", "Apple Notes", "## 两栏布局"} {
		if strings.Contains(app.Welcome, legacy) {
			t.Errorf("welcome retains legacy positioning %q", legacy)
		}
	}
}
```

- [ ] **Step 2: 运行测试并确认旧欢迎文档失败**

Run:

```bash
go test . -run TestWelcomeUsesMarkdownEditorPositioning
```

Expected: FAIL，提示缺少新定位并残留 `备忘录`、`Apple Notes` 等旧文案。

- [ ] **Step 3: 重写欢迎文档的产品开场与布局说明**

在 `app/state.go` 的 `Welcome` 常量中，用以下内容替换开头到“专注写作”段落；后续“编辑与预览”“格式一览”及图表示例保持原样：

````markdown
# 欢迎使用 Corra

Corra 是一款纯原生 Markdown 桌面编辑器，提供文档库、所见即所得与源码编辑，并支持 GFM、数学公式、图表及多格式导出。

## 文档库与编辑区

- **文档库**：「全部文档」汇集库中的 Markdown 文件，各文件夹与「最近删除」用于组织文档；文档列表按修改日期分组，搜索框可检索完整 Markdown 源码
- **编辑区**：直接写作、编辑与预览，顶部显示文档修改时间；「视图」菜单可开启大纲面板与字数统计

边栏左上角按钮或「视图」菜单（Cmd+Alt+S）可收起边栏，专注写作。
````

- [ ] **Step 4: 验证新文案及原有解析/导出能力**

Run:

```bash
gofmt -w app/state.go main_test.go
go test . -run 'TestWelcome'
```

Expected: `TestWelcomeUsesMarkdownEditorPositioning` 和 `TestWelcomeNoteParsesAndRoundTrips` 均 PASS，富文本往返与 HTML/PDF 导出无回归。

- [ ] **Step 5: Commit**

```bash
git add app/state.go main_test.go
git commit -m "docs: position welcome content as a Markdown editor"
```

### Task 4: 应用层错误与提示

**Files:**
- Create: `app/product_copy_test.go`
- Modify: `app/notes.go:56-68`
- Modify: `app/folders.go:50-180`
- Modify: `app/library_paths.go:10-83`
- Modify: `app/attachment_paths.go:161-167`
- Modify: `app/save.go:17-74`
- Modify: `app/attach.go:35-53`

- [ ] **Step 1: 新建关键错误文案的失败测试**

创建 `app/product_copy_test.go`：

```go
package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProductErrorCopy(t *testing.T) {
	t.Run("library is not configured", func(t *testing.T) {
		_, err := openLibraryRoot("", false)
		if err == nil || err.Error() != "document library is not configured" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("not a Markdown document", func(t *testing.T) {
		_, err := libraryNotePath(nil, "", "plain.bin")
		if err == nil || err.Error() != "not a Markdown document: plain.bin" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("new document needs a library", func(t *testing.T) {
		a := New()
		err := a.NewNote()
		if err == nil || err.Error() != "未设置文档库目录" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("non-regular document", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "folder")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		err := SaveExistingNote(path, "content")
		want := "cannot save a non-regular document: " + path
		if err == nil || err.Error() != want {
			t.Fatalf("error = %v, want %q", err, want)
		}
	})
}
```

该测试只覆盖无需并发构造的关键入口；其余精确字符串由现有文件系统测试和下面的残留检查保护，不新增脆弱的并发测试。

- [ ] **Step 2: 运行新测试并确认失败**

Run:

```bash
go test ./app -run TestProductErrorCopy
```

Expected: FAIL，实际错误仍包含 `notes library`、`Markdown note`、`备忘录文件夹` 或 `non-regular note`。

- [ ] **Step 3: 应用精确错误文案**

`app/notes.go`：

```go
return fmt.Errorf("未设置文档库目录")
```

`app/attach.go`：

```go
return "", fmt.Errorf("先保存文档，再插入附件")
```

`app/folders.go` 的三处 `not a notes folder: %s` 均改为：

```go
fmt.Errorf("not a document library folder: %s", folder)
```

状态错误改为：

```go
fmt.Errorf("restore requires a deleted document; move requires an active document")
```

`app/library_paths.go` 使用：

```go
fmt.Errorf("document library is not configured")
fmt.Errorf("symbolic links are not managed document library paths: %s", rel)
fmt.Errorf("not a Markdown document: %s", path)
fmt.Errorf("document is outside the library: %s", path)
fmt.Errorf("not a regular document file: %s", path)
```

`app/attachment_paths.go` 使用：

```go
fmt.Errorf("document changed during move: %s", source)
```

`app/save.go` 使用：

```go
fmt.Errorf("cannot save a non-regular document: %s", path)
fmt.Errorf("document changed during save: %s", path)
```

保留所有 `%s` 参数、错误产生条件和文件系统逻辑；不要修改 `main.go` 的 `seed note:` / `open note:` stderr 日志。

- [ ] **Step 4: 运行 app 包完整回归**

Run:

```bash
gofmt -w app/product_copy_test.go app/notes.go app/folders.go app/library_paths.go app/attachment_paths.go app/save.go app/attach.go
go test ./app -run TestProductErrorCopy
go test ./app
```

Expected: 全部 PASS，既有路径安全、移动、保存和附件测试无回归。

- [ ] **Step 5: Commit**

```bash
git add app/product_copy_test.go app/notes.go app/folders.go app/library_paths.go app/attachment_paths.go app/save.go app/attach.go
git commit -m "fix: use document terminology in user errors"
```

### Task 5: README、能力说明和视觉夹具

**Files:**
- Modify: `README.md`
- Modify: `docs/typora-parity.md:62-85`
- Modify: `views/zz_layout_shot_test.go:23-29`
- Modify: `nativerich/zz_typo_shot_darwin_test.go:17-23`

- [ ] **Step 1: 更新英文 README 的定位和文档术语**

将英文简介改为以下语义，保留 MyGo 链接及后续技术说明：

```markdown
Corra is a **Markdown desktop editor** with a **fully native UI**, written in
Go with the [MyGo](https://mygo.egoist.dev) toolkit. No webview, no HTML,
no JavaScript. Go draws the shell on the GPU; macOS uses an AppKit
`NSTextView` for rich-text editing.
```

将首条 Highlights 改为：

```markdown
- **Document library and editing workspace** — folders and a
  modification-date-grouped document list in the sidebar, with a dated editor
  and optional outline
```

在英文 README 中按语境完成以下精确调整，不做跨文件全局替换：

| 当前表达 | 目标表达 |
| --- | --- |
| `Notes still autosave as Markdown files` | `Documents still autosave as Markdown files` |
| `the note menu` | `the document menu` |
| `next to the note` | `next to the document` |
| `Local Notes workflow` | `Local document workflow` |
| `new note` | `new document` |
| `search notes` | `search documents` |
| `Deleted notes` | `Deleted documents` |
| `places the note` | `places the document` |
| `above the note` | `above the document` |

将英文存储段的完整开头替换为以下内容，而不是只替换 `Notes` 或 `like Apple Notes' folders` 片段：

```markdown
Documents live in a plain folder of Markdown files
(`<config>/备忘录` by default; each subfolder appears as a folder in the sidebar).
```

保留后续关于磁盘同步和自动保存的说明。存储段保留 `<config>/备忘录` 字面量；它仍是实际默认路径。

- [ ] **Step 2: 更新中文 README 的定位和操作说明**

中文简介使用：

```markdown
Corra 是一个**纯原生 UI** 的 Markdown 桌面编辑器，用 Go 基于
[MyGo](https://mygo.egoist.dev) 工具包编写：无 webview、无 HTML、无
JavaScript。Go 在 GPU 上绘制外壳，macOS 编辑区使用 AppKit 原生富文本控件。
```

首条功能使用：

```markdown
- **文档库与编辑工作区**：侧栏提供文件夹和按修改日期分组的文档列表，
  编辑区显示文档修改时间，并可开启大纲
```

按语境完成以下调整：

| 当前表达 | 目标表达 |
| --- | --- |
| `备忘录菜单` | `文档菜单` |
| `附件保存到笔记旁` | `附件保存到当前文档旁` |
| `备忘录就是一个普通文件夹里的 Markdown 文件` | `文档以普通 Markdown 文件存放在文档库中` |
| `与 Apple Notes 一致` | 删除该比较，直接说明库子文件夹对应侧栏文件夹 |
| `本地备忘录操作` | `本地文档操作` |
| `恢复备忘录` | `恢复文档` |
| `最近删除`中的`备忘录` | `最近删除`中的`文档` |

其余 `note` / `notes` 操作说明逐项改为 `document` / `documents`。保留 `[!NOTE]`、release notes 等技术语义；本文件没有需要改名的内部 Go API。

- [ ] **Step 3: 更新 Typora 对照文档**

`docs/typora-parity.md` 只做两处产品语境调整：

```markdown
| 拖放图片进编辑器 | 🆕 拖到窗口即插入当前文档（Markdown 文件拖入仍是打开） |
```

```markdown
| 文件树 + 文章列表侧栏 | ✅ 文件夹 + 文档列表 |
```

保留同行其他单元格以及 `[!NOTE]` 和 `Note over/left of/right of` 技术语法。

- [ ] **Step 4: 更新仍用于视觉验收的夹具**

`views/zz_layout_shot_test.go`：

- `一个用 Go 写的原生 Markdown 备忘录，采用 Apple Notes 风格。` 改为 `一个纯原生 Markdown 桌面编辑器，提供文档库、所见即所得与源码编辑。`
- `# 空白备忘录` 改为 `# 空白文档`。

`nativerich/zz_typo_shot_darwin_test.go`：

- 开场改为 `一个纯原生 Markdown 桌面编辑器，提供文档库、所见即所得与源码编辑。`，保留后续用于观察长行的说明。
- 将 `## 三栏布局` 改为 `## 文档库与编辑区`，不保留固定栏数定位。
- 将三个列表项完整改为：

```markdown
- **文档库**：管理库根目录下的 Markdown 文件夹
- **文档列表**：按修改日期分组，顶部可全文搜索
- **编辑区**：顶部显示文档修改时间；「视图」菜单可开启大纲面板与字数统计
```

不修改历史验收截图或日志。

- [ ] **Step 5: 验证文档和截图相关测试**

Run:

```bash
gofmt -w views/zz_layout_shot_test.go nativerich/zz_typo_shot_darwin_test.go
go test ./views
go test ./nativerich
```

Expected: PASS；如平台或环境跳过原生截图测试，应在输出中保持 skip，而不是失败。

- [ ] **Step 6: Commit**

```bash
git add README.md docs/typora-parity.md views/zz_layout_shot_test.go nativerich/zz_typo_shot_darwin_test.go docs/superpowers/specs/2026-10-17-markdown-product-copy-design.md docs/superpowers/plans/2026-10-17-markdown-product-copy.md
git commit -m "docs: describe Corra as a Markdown editor"
```

### Task 6: 全量验证与兼容边界检查

**Files:**
- Verify all files changed in Tasks 1–5.
- Verify unchanged: `main.go`, `.github/workflows/release.yml`, `setup/workflows/release.yml`, internal Go API names, `.trash` behavior.

- [ ] **Step 1: 检查旧产品语境残留**

使用 Grep 工具分别搜索，不做全局替换：

1. 在 `app/`、`views/`、`menu.go`、`README.md`、`docs/typora-parity.md` 搜索 `备忘录|笔记|Apple Notes`。
2. 在 `README.md` 搜索 `(?i)\bnotes?\b`。
3. 在生产 Go 文件搜索 `\bnotes?\b`，逐条判断是否只是内部标识、注释或明确保留的诊断日志。

Expected: 没有面向用户的旧定位。`<config>/备忘录`、内部 Note API/注释、`main.go` 的 stderr 日志、历史记录可以保留；`main_test.go` 的默认目录断言仍是 `备忘录`。

- [ ] **Step 2: 检查技术语法未被误改**

使用 Grep 工具检查：

- `render/`、`internal/mdext/`、`richtext/`、`nativerich/` 和 `docs/typora-parity.md` 中的 `[!NOTE]` / GitHub `Note` 提示块仍存在。
- `render/sequence.go` 和 `docs/typora-parity.md` 的 `Note over/left of/right of` 仍存在。
- `.github/workflows/release.yml` 与 `setup/workflows/release.yml` 的 `--generate-notes` 仍存在。
- `main.go` 的默认目录仍为 `<config>/备忘录`，`.trash` 仍为回收站存储目录。

- [ ] **Step 3: 运行项目标准验证**

Run:

```bash
gofmt -w app/library.go app/library_test.go app/state.go app/notes.go app/folders.go app/library_paths.go app/attachment_paths.go app/save.go app/attach.go app/product_copy_test.go menu.go main_test.go views/header.go views/notes.go views/commands.go views/editorpane.go views/interactions_test.go views/zz_click_probe_test.go views/zz_layout_shot_test.go nativerich/zz_typo_shot_darwin_test.go
go test ./...
go build ./...
git diff --check
```

Expected: 所有命令退出码为 0；没有测试失败、构建错误或空白错误。

- [ ] **Step 4: 审核最终差异**

Run:

```bash
git status --short
git diff --stat HEAD
```

Expected: 差异只覆盖本计划列出的文案、测试和现行文档；不包含用户已有的 `.claude/` 未跟踪内容，不包含默认目录迁移、内部 API 重命名或交互逻辑改动。若 Tasks 1–5 已分别提交，使用 `git log` 和逐提交差异完成同样核对。
