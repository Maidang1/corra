# Agent 交接文档：Corra 编辑器

面向接手的下一位 agent。读完本文应能：跑起来、改得动、测得过，并知道下一步该做什么。

## 0. 一分钟上手

- 项目：Corra，Go + MyGo 的 macOS 原生 Markdown 桌面编辑器；富文本编辑区是 AppKit `NSTextView`，经 cgo 嵌入（`nativerich/`），**没有 WebView**。
- 当前分支：`codex/native-rich-editor`；最新提交 `afc5c34`（功能：导出面板）；工作区干净。
- 环境：macOS + Xcode Command Line Tools、Go 1.27.1、cgo 必须开启。
- 跑应用：`make dev`（macOS 自动开启 cgo）；打包：`make build`。
- 跑测试（提交前必跑）：

```bash
CGO_ENABLED=1 go test ./... -count=1 -timeout 600s
CGO_ENABLED=1 go vet ./...
```

- `ld: warning: ignoring duplicate libraries: '-lobjc'` 是正常告警，可忽略。
- 先读：`README.md`（功能与用法，中英双语）、`docs/work-summary.md`（本轮系列功能总结）、`docs/typora-parity.md`（产品对齐方向）。

当前长期目标（用户原话）：「loop 一直打磨 UI 和功能，调研其他 Markdown 编辑器有而 Corra 没有的功能，都加进来，注意界面的美观。」每轮交接应保持：**一个具体功能 + 测试 + 截图 + README 更新 + 全量测试通过**。

## 1. 代码地图

### 外壳（Go / MyGo）

| 路径 | 职责 |
| --- | --- |
| `main.go` | 创建窗口、原生编辑器并接线回调（`OnLinkCandidates`、`OnOpenRecent`、`OnExportPDF` 等）、菜单初始化 |
| `menu.go` | 应用菜单与快捷键（`CmdOrCtrl+P` 打印、`CmdOrCtrl+Shift+P` 快速打开） |
| `views/` | MyGo 界面：侧栏、标题栏、编辑区、状态栏、模态框、快速打开、文档统计 |
| `views/quickopen.go` | 快速打开 / 命令面板：文档搜索、最近分组、`>` 命令模式、↑↓ 选择 |
| `views/stats.go` | 文档统计扫描（`inspectDocument`）与状态栏统计条 |
| `views/link.go` | 链接插入弹窗 + `LinkCandidates`（`[[` 补全的 Go 数据源） |
| `views/editorpane.go` | 编辑区列布局；富文本视图与源码视图切换、状态栏 |
| `views/rich_editor.go` | `RichEditor` 接口、`richVisible()`（模态打开时隐藏原生编辑器） |
| `views/settings.go` / `views/modal.go` | 设置面板、冲突/图床等模态框 |
| `app/` | 状态、文档库扫描、文件打开/保存、设置持久化、最近文件（`App.RecentPaths()`） |
| `editor/` | 源码模式编辑器 |
| `richtext/`、`render/`、`internal/mdext/` | Markdown 解析、渲染、扩展语法（高亮/上下标/表情 shortcode 等） |

### 原生富文本编辑器（`nativerich/`）

| 文件 | 职责 |
| --- | --- |
| `bridge_darwin.m` | `CREditor` 与 `CRTextView` 的实现；include 各功能头文件；输入/选区/生命周期钩子；`cr_*` C 桥；测试桥 |
| `bridge.h` | 测试桥的 C 声明（`cr_test_*`） |
| `harness_darwin.go` | Go 测试辅助（`nativeTestMain`、模拟输入、悬停、截图） |
| `slash_menu_darwin.h` + `slash_menu_view_darwin.h` | `/` 斜杠菜单：23 项、分组、过滤高亮、键盘导航 |
| `format_bar_darwin.h` + `format_bar_view_darwin.h` | 选区格式浮动条（粗/斜/下划线/删除线/代码/高亮/链接/清除） |
| `block_actions_darwin.h` | 块级操作：移动、复制、删除、块矩形、拖动排序 |
| `hover_controls_darwin.h` | 悬停控件：块手柄、代码复制/语言、图片尺寸/复制/删除、表格加行/加列、脚注卡片触发 |
| `code_controls_darwin.h` | 代码块扫描、语言读取/写入 |
| `footnote_darwin.h` | 脚注映射、引用定位、跳转、tooltip 数据 |
| `footnote_card_view_darwin.h` | 脚注悬停卡片视图（无焦点面板） |
| `emoji_menu_darwin.h` + `emoji_menu_view_darwin.h` | `:` 表情 shortcode 选择器 |
| `link_menu_darwin.h` + `link_menu_view_darwin.h` | `[[` 文档补全与 `[[#` 标题锚点补全 |
| `toc_darwin.h` | 目录图、标题条目（`tocEntriesFromText`）、`CRSlug`、锚点跳转 |
| `structure_controls_darwin.h` | 键盘处理（块快捷键、面板按键路由）与右键菜单（表格/列表/块操作） |
| `table_controls_darwin_test.go` 等 | 各功能测试；`zz_*_shot_darwin_test.go` 为截图入口 |

### 输入数据流

1. 用户键入 → `CRTextView insertText:` → 自动配对/表格单元处理 → `applyInputRules`（`:` 表情、输入规则）→ 段落规范化。
2. 插入/删除/选区变化都会触发 `slashQueryChanged` / `emojiMenuQueryChanged` / `linkMenuQueryChanged`（在 `bridge_darwin.m` 钩子中）。
3. 面板打开后由 `structure_controls_darwin.h` 先路由按键（斜杠 → 表情 → 链接），其余交给文本视图。
4. 文档始终以 `NSTextStorage` 属性 + `CRParagraph` 元数据存在；保存时由序列化器生成 Markdown，**不要直接往文本里塞 Markdown 字面量**（链接会变成 `\[...\]`），应构造对应属性的 run（参考 `linkMenuChoose:`）。

## 2. 新增一个面板的配方

以表情/链接面板为模板，按顺序做：

1. **视图头文件（文件作用域）**：`XxxPanel : NSPanel`（`NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel`，`canBecomeKey/Main = NO`，`releasedWhenClosed = NO`，透明背景、阴影、`alphaValue = 0` 淡入）；`XxxView : NSView`（`isFlipped`、绘制、悬停、滚动）。
2. **编辑器半部头文件**（include 进 `@implementation CREditor`）：
   - 查询范围检测：必须检查 `editable`、`hasMarkedText`、`suppress`、无选区，并排除受保护块（`code/source/opaque/math/diagram/toc`、表格单元、脚注定义）。
   - 拉取候选、开/关、定位（跟随光标、越界翻转）、键盘（↑↓/回车/Esc）、选择/插入。
3. **`bridge_darwin.m` 接线**：
   - `@class` 前置声明、`CREditor` ivar、方法声明；
   - 视图头文件放文件作用域 include，编辑器半部放 `@implementation` 内的 include 列表（顺序依赖：`toc` 等在 link 之前）；
   - 钩子：`insertText`/`deleteBackward`/`textViewDidChangeSelection`/`textDidEndEditing`、`handleStructureKey`、右键菜单、`cr_load`/`cr_close`/`cr_style`/`cr_visible`；
   - 面板互斥：打开任一文本触发面板时关闭另外两个。
4. **测试**：加 `cr_test_xxx` 桥（返回 JSON 状态、可选截图），在 `harness_darwin.go` 包一层，写确定性测试；需要视觉检查时加 `zz_xxx_shot_darwin_test.go` + `CORRA_XXX_SHOT`。
5. 更新 `README.md` 中英功能列表与 `docs/work-summary.md`（或相应文档）。

## 3. 关键约束与踩过的坑

- **测试死锁**：`nativeTestMain` 在主线程跑 AppKit，内部禁用 `t.Fatalf`（Goexit 会跳过 dispatch 返回导致死锁）。用 `t.Errorf` + `return`。
- **面板不能抢焦点**：全部用无焦点子 `NSPanel`；点击面板不应改变文本选区/第一响应者。
- **绘制顺序**：文本容器之外的控件（左侧块手柄、表格加行/加列按钮）必须在 `[super drawRect:]` **之前**画，否则被 `NSTextView` 裁剪；覆盖在字形上的控件（图片尺寸条、拖动插入线）在之后画。
- **JSON 布尔**：`NSJSONSerialization` 会把 `@(BOOL)` 变成 `1`，Go 解析失败；用 `cond ? @YES : @NO`。
- **C 字符字面量**：不要写全角字符字面量（`'（'` 编译报错），用 `0xFF08` 之类码点。
- **char 参数**：`CRCellAt`/`CRTableRange` 接收 `CREditor *`，在 `CRTextView` 里传 `self.owner`。
- **撤销**：结构性替换用 `beginStructuralCommand` + `[_undo beginUndoGrouping]`；测试里连续输入与插入可能落在同一撤销组（harness 无事件循环），断言时不要假设能单独撤销一次。
- **自动配对**：输入 `[[` 会自动补 `]]`，链接插入时需消费末尾最多两个 `]`（`linkMenuChoose:` 已处理）。
- **过期状态**：`cr_load`/`cr_close` 必须清理悬停、拖动、脚注卡片与所有面板，否则切换文档后残留。
- **截图**：`cr_test_screenshot` 会先 `[e.text display]` 再抓取，避免取到旧图层；截图产物写 `.cache/`（已 gitignore）。
- **跨平台**：原生实现仅 darwin+cgo；`nativerich/editor_stub.go` 必须同步新增字段/方法，否则非 macOS 构建或 `main.go` 赋值会失败。

## 4. 测试策略

- **原生**：`go test ./nativerich -run TestXxx -v`。测试通过 `nativeTestMain` 在主线程执行；用 `nativeTestAction`（`type`/`key`/`backspace`/`undo`/`slash-pick`/`emoji-pick`/`link-pick` 等）驱动，悬停/拖拽有专用桥。不要用原生弹出菜单（会阻塞事件循环）。
- **外壳**：`ui.NewTester` + `tt.Click("标签")`（按钮 `Label` 同时用于可访问性）、`tt.Type`、`tt.Key(0, ui.KeyEnter)`、`tt.HasText`、`waitFor`/`waitTransition`。
- **截图**：
  - 原生：`CORRA_SLASH_SHOT`、`CORRA_FORMAT_BAR_SHOT`、`CORRA_HOVER_SHOT`、`CORRA_EMOJI_SHOT`、`CORRA_LINK_SHOT`、`CORRA_TABLE_SHOT`
  - 外壳：`CORRA_SCREENSHOTS=<dir>`（快速打开、命令面板、文档统计、设置等）
  - 其他历史截图变量见 `grep -rho "CORRA_[A-Z_]*" nativerich views`。
- 全量测试约 40–60 秒；提交前必须全绿并 `go vet` 干净。

## 5. 当前状态

已完成并验证（`66b64d7` 提交）：

- 斜杠菜单、选区格式条、块移动/复制/删除/拖动、代码块复制与语言标签。
- 脚注引用悬停卡片；`: ` 表情选择器；`[[` 文档补全与 `[[#` 标题锚点。
- 表格悬停加行/加列；图片尺寸/复制/删除。
- 快速打开（文档搜索、最近打开分组、↑↓ 选择）与 `>` 命令面板（17 个动作）。
- 脚注返回导航（`e0b8d95`）：`⌘⌥←` 回到跳转起点（toggle）、右键「返回跳转前位置」、命令面板入口；`cr_load`/`cr_close` 清除返回点。
- 导出面板（`afc5c34`）：`views/export.go` 格式卡片模态；`OnExportHTML`/`OnExportPandoc` 回调；`View.PandocPath` 可注入（测试确定性）；截图 `TestExportShot`。
- 状态栏文档统计条（字数/字符/行/标题/链接/图片/任务/代码块/阅读时间）。
- 测试与截图脚本；`docs/work-summary.md` 有完整功能与架构总结。

已知限制：

- 快速打开结果上限 8 条，仅子串匹配（无模糊评分）。
- 表情目录是约 120 个精选 shortcode，非全量 Unicode。
- 链接补全只检索「可见文档」（受侧栏筛选/回收站影响）。
- 表格行列移动目前只通过右键菜单，没有拖拽手柄。
- 脚注卡片仅展示（忽略鼠标事件）；返回引用走 `⌘⌥←`、右键「返回跳转前位置」或命令面板。
- 代码行号是全局开关，没有每块独立开关。

## 6. 建议的下一步（可直接开工）

1. **表格行列拖拽排序**：表格矩形与悬停状态已在 `hover_controls_darwin.h`（`_hoverTable`/`_tableRect`）；行列移动命令已存在（`table:row-up`/`table:row-down`/`table:column-leftward`/`table:column-rightward`，见 `structure_controls_darwin.h`）。可仿照块拖动（`beginBlockDragAtPoint:`/`updateDragTargetAtPoint:`/`commitBlockDrag`）实现行/列手柄与插入线，测试用直接桥驱动。
2. ~~导出面板~~（已完成）：格式卡片模态在 `views/export.go`，回调 `OnExportHTML`/`OnExportPandoc` 已在 `main.go` 接线；截图测试 `TestExportShot`。
3. **图片替换**：图片 pill 已有复制/删除（`_imageCopyRect`/`_imageDeleteRect`、`copyImageAtHover`/`deleteImageAtHover`）；附件是 `NSTextAttachment`（`.image`，属性 `CRImageURL`/`CRImageSource`），可加替换按钮 → `NSOpenPanel` → 更新 URL 并用 `imageAttachmentForURL:` 重载；测试用「给定路径」桥绕过面板。
4. **侧栏键盘导航**：`views/notes.go` 的列表与 `App.SelNote`；用 `OverlayShortcut` 处理 ↑↓/回车，注意不要从原生编辑器抢焦点。
5. **每代码块行号**：需要可持久化的 Markdown 扩展（fence info 或文档元数据），涉及 `internal/mdext` 解析与 `nativerich` 代码块渲染；若不想扩展语法，保持全局开关即可。
6. **查找替换核查**：`ShowFindBar` 使用 `NSTextView` 自带查找条；确认是否提供替换，若缺失再考虑自绘面板。
7. **脚注卡片加交互入口**：卡片目前忽略鼠标事件；可考虑让卡片可点击（「跳到定义」链接样式文本），需把卡片面板改为可接受鼠标移动/点击且不抢焦点。

## 7. 协作约定

- UI 文案中文；代码注释英文；Objective-C 4 空格缩进。
- 每轮一个可演示的改动：实现 → 单元/原生测试 → 截图目检 → 更新 README（中英）→ 全量测试 → 提交。
- 提交信息英文、`feat:`/`docs:` 前缀；文档用中文。
- 涉及视觉时优先复用现有面板样式（圆角、阴影、`CRTintedSymbol`、`fieldFill`、强调色），保持明暗一致。
- 如果用户要求「继续 loop」，不要停在中途：每轮给出实际进展与验证证据，直到用户叫停。
