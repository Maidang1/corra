# Agent 交接文档：Corra 编辑器

面向接手的下一位 agent。读完本文应能：跑起来、改得动、测得过，并知道下一步该做什么。

## 0. 一分钟上手

- 项目：Corra，Go + MyGo 的单栏 Markdown 桌面编辑器（macOS/Windows/Linux），**没有 WebView**。编辑区是自绘文档编辑器（`internal/editor`），三平台同一实现。
- 当前分支：`document-model`；设计文档 `docs/superpowers/specs/2026-11-24-document-model-design.md`，分步计划 `docs/superpowers/plans/2026-11-24-document-model.md`（含进度记录，先读它）。
- 环境：Go 1.27+；macOS 需要 cgo（数学排版 `render/mathtext_darwin.go` 与拼写检查 `internal/spell` 都用 cgo），Windows/Linux 走 `CGO_ENABLED=0`。
- 跑应用：`make dev`；打包：`make build`。
- 跑测试（提交前必跑）：

```bash
CGO_ENABLED=1 go test ./... -count=1 -timeout 600s   # macOS
CGO_ENABLED=1 go vet ./...
CGO_ENABLED=0 GOOS=windows go build ./...             # 交叉编译检查
CGO_ENABLED=0 GOOS=linux go build ./...
```

- gopls 对 ObjC cgo 前导码（`internal/spell/darwin_cgo.go`、`render/mathtext_darwin.go`）会报假错（"use of undeclared identifier"），以 `go build` 为准。

## 1. 代码地图

### 文档模型与编辑器

| 路径 | 职责 |
| --- | --- |
| `internal/doc` | 不可变文档树：`NodeID`、schema/`Registry`、四个原语（ReplaceText/SetAttrs/SetType/ReplaceChildren）、可逆 op 日志、撤销历史存旧根 |
| `internal/mdcodec` | goldmark ⇄ 树的双向编解码（读一次、编辑后写回，写法收敛到定点） |
| `internal/ext` + `internal/ext/std` | 扩展接口与默认套件（`Kit`：Registry、Codec、EditorConfig）；各扩展语法在 `internal/mdext` 等包 |
| `internal/editor` | 自绘编辑器：基于 MyGo `ui.TextInputClient`/`ui.TextLayout`；选区、撤销、查找替换、补全、粘贴、拖拽、拼写、行号 |
| `internal/spell` | 三平台拼写检查：macOS NSSpellChecker（cgo）、Windows Spell Checking API（syscall COM，专用 STA 线程）、Linux hunspell（purego，无 cgo） |
| `internal/htmlmd` | 粘贴 HTML → Markdown 的保真转换器 |
| `views/rich_doc.go` | `DocRich`：把编辑器接进应用（工具栏、查找条、图片加载、页面卡片、`[[` 补全、附件保存） |

### 外壳（Go / MyGo）

| 路径 | 职责 |
| --- | --- |
| `main.go` | 创建窗口、`DocRich` 编辑器并接线回调、菜单初始化；无平台分支 |
| `menu.go` | 应用菜单与快捷键 |
| `views/` | MyGo 界面：侧栏、标题栏、`RichEditor` 接口与 `richPane`、状态栏、模态框、快速打开、文档统计 |
| `app/` | 状态、文档库扫描、文件打开/保存、autosave、设置持久化、图床上传 |
| `render/` | 导出管线：`render.Parse(Markdown)` → HTML/PDF（不经过文档树） |
| `editor/` | 源码模式编辑器（MyGo `ui/editor`） |

### 数据流

用户键入 → `internal/editor` 应用一条 `doc.Command` → `OnChange` 置脏 → autosave 时 `Snapshot()` 把树写回 Markdown（未编辑的文档永不重写）。源码模式改动经 `Reload` 回到树，保留未变块。

## 2. 关键约束与踩过的坑

- **UTF-16 偏移**：编辑器选区、拼写范围都用 UTF-16 码元（`ui.TextInputRange`）；rune 与 UTF-16 的换算见 `internal/spell/spell.go` 的 `words()`。代理对（emoji）占 2。
- **AppKit 主线程**：所有 NSSpellChecker 调用必须经 `mygo.RunOnMain`。
- **累计计数器断言**：测试里的 `renders`/`loads` 等计数器是累计值，断言用前后差值，不要写 `!= 0` / `== 0`（改写旧断言时最容易踩）。
- **ObjC JSON 布尔**：`NSJSONSerialization` 会把 `@(a||b)` 序列化成 `0/1`，Go 端解析失败；要写 `cond ? @YES : @NO`。
- **Linux 无 cgo**：CI 在 Linux/Windows 用 `CGO_ENABLED=0`；需要系统库时用 purego dlopen（参考 `internal/spell/linux.go` 的懒加载 + 静默降级模式）。
- **新增 `RichEditor` 方法**：同步更新 `views/rich_editor_test.go` 的 `fakeRichEditor` 与 `fakeInlineRichEditor`，以及 `DocRich`。
- **测试窗口**：`ui.NewTester` + `tt.Click("标签")`、`tt.Type`、`tt.Key`；悬浮面板类 UI 在测试窗口里不进视图层级，用特征函数验证。

## 3. 测试策略

- 全量 `go test ./...`（macOS 约 40–60 秒）。
- `internal/mdcodec/corpus_test.go`：`CORRA_CORPUS` 环境变量 + 仓库自带 Markdown 做编解码往返与定点测试。
- `internal/spell/darwin_cgo_test.go`：真机 NSSpellChecker，无词典环境自动跳过。
- 截图：`CORRA_SCREENSHOTS=<dir>`（外壳各模态）；生成产物写 `.cache/`（已 gitignore）。
- 提交前：全量测试全绿、`go vet` 干净、三平台 `go build` 通过。

## 4. 当前状态（2026-11-25）

- 文档模型编辑器已上线为**唯一**编辑器：阶段 1–5 全部完成。旧的 `nativerich/`（AppKit NSTextView 桥）、`richtext/`、`internal/mgdoc/`、`views/rich_mygo.go` 已删除（净删约 2.77 万行），`LegacyEditor` 回退开关一并移除（设置 JSON 里的未知字段会被忽略，老配置可直接升级）。
- 拼写检查三平台已接系统接口（见计划文档 2026-11-25 进度记录）；**尚未在 Windows/Linux 真机实测**。
- 已知限制：`goat` 图表在 oksvg 下栅格化失败（预览显示错误信息）；导出走 `render.Parse` 独立管线，未经文档树（阶段 6，可选）。

## 5. 建议的下一步

1. **三平台手动验收**：Windows/Linux 真机跑一遍编辑、粘贴图片、拼写检查，补上验收记录后即可发版。
2. **阶段 6（可选）**：导出直接从 `*doc.Node` 渲染（各扩展实现 HTML 渲染，`render` 增加从树构造 `render.Document` 的入口）。
3. **`goat` 栅格化**：排查 oksvg 对 goat 输出的兼容问题，或换栅格化器。

## 6. 协作约定

- UI 文案中文；代码注释英文。
- 每轮一个可演示的改动：实现 → 单元测试 → 截图目检 → 更新 README（中英）→ 全量测试 → 提交。
- 提交信息英文、`feat:`/`docs:` 前缀；文档用中文。
- 如果用户要求「继续 loop」，不要停在中途：每轮给出实际进展与验证证据，直到用户叫停。
