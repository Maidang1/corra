# 文档模型重写：分步实现计划

设计：`docs/superpowers/specs/2026-11-24-document-model-design.md`（下称"设计"，§ 编号指设计文档的章节）
目标：用"不可变树 + 扩展注册 + 自绘编辑器"替换现在的两套富文本模型和两个渲染后端，三平台共用一套代码。

## 总原则

- **每个阶段都能单独合并**，合并后应用照常可用。新编辑器在阶段 5 之前一直放在设置开关后面，默认仍然是现有编辑器。
- **先写测试。** `doc`、`mdcodec` 的每一步都先写失败的测试，再写实现。
- **每一步的通用验收**：`go vet ./...`、`go test ./...` 通过；CI 的三平台矩阵（ubuntu、macos、windows）全部变绿。下文"验收"只列这一步额外要满足的条件。
- **依赖方向**：`internal/doc`、`internal/mdcodec` 不导入 MyGo；`internal/editor` 依赖 MyGo `ui`；`internal/ext/*` 依赖前三者；`views` 只通过 `RichEditor` 接口接入新编辑器。
- **一个阶段一个 PR。** 阶段内步骤多时，可以按"步骤"拆成多个 PR，但每个 PR 都要满足总原则。
- 工作量用 S（一两天）、M（一周内）、L（一周以上）粗估，只用于排序，不是承诺。

## 进度

- [x] 阶段 0：MyGo 升级与编辑器原型（**闸门**：不通过就停下来重新评估设计）—— 无窗口部分完成，真实平台手动检查见 `2026-11-24-editor-spike.md`
- [x] 阶段 1：`internal/doc` 文档模型
- [x] 阶段 2：`internal/mdcodec` 与基础扩展（任务列表并入 `ext/list`：checked 属性只能由 list_item 自己声明）
- [x] 阶段 3：`internal/editor` 编辑器 MVP，接入设置开关（**闸门**：性能和输入法）—— 代码完成；1000 段文档按键+绘制 0.71 ms（无窗口）；三平台手动清单未做
- [x] 阶段 4：功能迁移成扩展 —— 完成。块手柄、图片粘贴/拖入与图床上传、代码行号与公式编号已随编辑器落地；4.12 拼写检查于 2026-11-25 补上三平台系统接口（见下方进度记录）。已知限制：`goat` 在 oksvg 下栅格化失败（旧实现同样如此）
- [x] 阶段 5：5.1 切换默认与回退开关已做；5.2 删除旧实现于 2026-11-25 由用户明确指示提前执行（跳过手动验收闸门），见下方进度记录
- [ ] 阶段 6（可选）：导出直接从树渲染

---

## 阶段 0：MyGo 升级与编辑器原型

### 0.1 升级 MyGo（M）

- `go.mod` 里的 `github.com/egoist/mygo` 从 v0.7.3 升到当前最新版本，同时更新 `go tool mygo`。
- 逐个修复编译错误和行为变化。重点检查：`views/`、`views/rich_mygo.go`、`internal/mgdoc`、`main.go` 里的窗口和菜单、`nativerich` 的窗口句柄。
- 读新版 MyGo 的变更日志，记下和自绘编辑器相关的接口：`ui.TextInputClient`、`HandleTextInput`、`ui.TextLayout`、`Painter.TextLayout`、无障碍（`access.go`）、剪贴板、拖放。

验收：

- 现有全部测试通过，包括 macOS 上的 `nativerich` 测试。
- 三平台手动冒烟：打开文档、富文本编辑、源码模式、导出 PDF。
- 在 PR 描述里列出 MyGo 的破坏性变更和对应修改。

### 0.2 编辑器原型（M）

位置：`internal/editor/spike/`，加一个调试入口 `cmd/editorspike`。原型代码用完即删，不进主路径。

只做最难的部分，不接任何文档模型。数据就是 `[]string` 段落：

- 用 `ui.TextLayout` 排版多个段落，垂直堆叠，支持滚动。
- 实现 `ui.TextInputClient`：线性 UTF-16 文本 = 段落用 `\n` 相连；`ReplaceText`、`SetMarkedText`、`UnmarkText`、`BoundsForRange`、`IndexForPoint`。
- 光标绘制和闪烁；点击定位；拖动选择（跨段落）；Shift+方向键扩展选区。
- 左右按字素移动，上下移动时保持列位置；⌥/Ctrl+方向键按词移动。
- 输入法组字：组字文本叠加显示并带下划线，候选框跟随光标。
- 复制选中的纯文本。

验收（三平台手动，结果写进 `docs/superpowers/plans/` 下的原型记录）：

- [ ] 中文拼音输入法：组字、选候选、在段落中间输入、跨段落选中后用输入法替换
- [ ] 日文输入法（假名转汉字、分节）
- [ ] Emoji 和代理对：光标不会停在半个字符上
- [ ] 1000 段、每段 200 字的文本：按键到绘制的延迟，和滚动是否流畅（记录数字）
- [ ] 读屏器能读出当前段落（macOS VoiceOver、Windows 讲述人，至少一个）

**闸门**：输入法或选区在某个平台上做不出来，就停下来，回到设计讨论（备选方案是：在那个平台保留原生文本控件作为输入层）。

---

## 阶段 1：`internal/doc` 文档模型

本阶段全部是纯 Go，不导入 MyGo，不接 UI。

### 1.1 行内值类型（S）

文件：`internal/doc/inline.go`、`attrs.go`、`mark.go`

- `Attrs`：不可变；`Get`、`With`（返回新值）、`Equal`。
- `Mark`、`MarkSet`：按 `Rank` 排序且无重复；`Add`（处理 `Excludes`）、`Remove`、`Has`、`Equal`。
- `Segment`、`Inline`：
  - `Len()`：UTF-16 长度，原子算 1；
  - `Slice(from, to)`：在 UTF-16 偏移处切开，不能切在代理对中间（落在中间时向前取整）；
  - `Concat`；`AddMark`、`RemoveMark`；`MarksAt(offset)`；
  - 构造时自动合并相邻的同 mark 片段，丢掉空片段。
- 这一步先用一个测试用的最小 `MarkType`，正式的 `Registry` 在 1.2。

测试：表驱动，覆盖中英文、emoji、代理对边界、`code` 和其他 mark 互斥、相邻片段合并。

### 1.2 Schema 与节点（M）

文件：`internal/doc/schema.go`、`node.go`、`id.go`

- `NodeSpec`、`MarkSpec`、`AttrSpec`、`ContentKind`（设计 §2.3）。
- `Registry`：注册 spec，生成 `*NodeType`、`*MarkType`；重名、`Allow` 引用了不存在的组或类型时报错。
- `Node`：私有字段，加上只读访问器（`ID`、`Type`、`Attrs`、`ChildCount`、`Child(i)`、`Inline`、`Size`）。
  - 构造函数 `NewNode(typ, attrs, kids...)`、`NewText(typ, attrs, inline)`；构造时计算 `size` 和 `sizes` 前缀和，并检查内容种类和 `Allow`。
  - 属性按 `AttrSpec` 补默认值并校验。
- `NodeID`：进程内原子递增。`CloneFresh(n)`：深复制并全部换新 ID。

测试：用一个测试 schema（doc、paragraph、heading、blockquote、list、list_item、image、math）构造合法树和非法树；验证 `size` 和 `sizes`；验证 `CloneFresh` 后的 ID 全部不同。

### 1.3 State、位置与查找（M）

文件：`internal/doc/state.go`、`position.go`、`index.go`

- 持久化映射 `NodeID → 父 NodeID`。先评估第三方 HAMT 库（许可证必须是 MIT 或 BSD）；不合适就自己写一个最小 HAMT，只需要 `Get`、`Set`、`Delete`。
- `State{Doc, Sel, Stored}`，`NewState(doc)` 时建好父节点映射。
- `Point`、`TextSelection`、`NodeSelection`、`CellSelection`（设计 §2.4）。
- 查找：
  - `Find(id) (*Node, Path)`；
  - `LeafRanges(sel)`：按文档顺序把选区切成 `[{leaf, from, to}]`；
  - `GlobalOffset(Point)` 和 `PointAt(global)`：通过 `sizes` 二分查找；
  - `NextLeaf`、`PrevLeaf`（光标跨块移动用）。

测试：随机树上 `PointAt(GlobalOffset(p)) == p`；`LeafRanges` 的总长度等于选区的全局长度。

### 1.4 事务、原语与操作日志（L）

文件：`internal/doc/tx.go`、`op.go`、`commit.go`

- `Tx`：`ReplaceText`、`SetAttrs`、`SetType`、`ReplaceChildren`，以及 `SetSelection`、`SetStoredMarks`（设计 §3.1）。
- **所有权**：Tx 内复制出来的节点打上"归本 Tx 所有"标记，后续原语原地修改；写进日志的节点冻结（设计 §3.5）。
- 每个原语记一条 `Op`；`Op.Invert()`。
- `ReplaceChildren` 检查 ID 唯一：插入的子树里不能有已经在树里的 ID，除非是同一个 Tx 里刚从别处移除的（移动）。
- `ReplaceChildren` 和 `SetType` 后增量更新父节点映射。
- 选区映射：`ReplaceText` 平移同一个块里的 `Point`；块被删除时，选区落到最近的文本块（命令应当自己设选区，这只是兜底）。
- `Commit() (State, Commit)`：结束所有权，产生新 State 和 `Commit{Before, After, Ops, Meta}`。

测试（属性测试，各跑几千次随机序列）：

- 正向：在 `Before.Doc` 上重放日志，等于 `After.Doc`，包括 ID；
- 反向：在 `After.Doc` 上倒序执行 `Invert()`，等于 `Before.Doc`；
- `Before.Doc` 里的所有节点在提交后都没有被修改（提交前对 `Before` 做深快照，提交后比较）；
- 一个 Tx 里对同一路径做 N 次修改，根节点只复制一次（用计数钩子检查）。

### 1.5 规范化（M）

文件：`internal/doc/normalize.go`

- 内核规则：文本块的行内内容合并或去空（多数已在 `Inline` 构造时完成）；容器子节点不符合 `Allow` 时，交给 spec 的 `Normalize`，没有就包进 `Group` 的默认块；`doc` 不能为空。
- 只处理日志涉及的节点及其祖先；规范化做的修改通过原语执行，`Origin = Normalize`。
- 迭代上限是涉及节点数乘以 10；超过时测试模式 panic，正式模式记日志并停止。

测试：故意注册一个互相触发的规则，验证上限生效；随机命令序列后，`Validate(doc)` 总是通过。

### 1.6 撤销历史（S）

文件：`internal/doc/history.go`

- `History{Push(Commit), Undo(State) (State, bool), Redo(...)}`，存 `before`、`after` 两个 State（设计 §3.4）。
- 合并规则：同为 `input` 或同为 `delete`、同一文本块、位置首尾相接、间隔 1 秒内；其他类型总是开新条目。`AddToHistory=false` 的提交不入栈。
- 上限 200 条。

测试：连续输入合并成一条；输入后格式化是两条；撤销后根指针等于之前的根；新提交清空重做栈。

### 1.7 命令与通用 helper（M）

文件：`internal/doc/command.go`、`helpers.go`

- `Command`、`Can`、`Chain`、`FirstOf`（设计 §3.3）。
- 通用 helper（只依赖内核结构，不认识具体节点类型）：
  - `InsertText`（套用待用 mark）、`DeleteSelection`（跨块）、`SplitBlock`、`JoinBackward`、`JoinForward`；
  - `AddMark`、`RemoveMark`、`ToggleMark`；
  - `SetBlockType`、`Wrap`、`Lift`、`MoveBlock`、`InsertBlocks`、`InsertFragment`（粘贴，含两端合并）。
- 列表的 sink 和 lift、表格的导航之类的类型相关逻辑，放到阶段 2 和 4 的扩展里。

测试：每个 helper 都写表驱动测试，用一种紧凑的树文本表示输入和期望输出，例如 `doc(p("a|b"), p("c"))`，`|` 是光标，`[` 和 `]` 是选区。测试工具放在 `internal/doc/doctest`。

### 1.8 随机测试框架（M）

文件：`internal/doc/doctest/gen.go`、`run.go`

- 按 schema 随机生成合法的树。
- 随机命令序列执行器：从命令表里随机挑命令，随机设选区。
- 每一步检查：schema 合法（设计 §7.3 第 3 条）、撤销回到同一个根（第 4 条）、日志正反重放（第 5 条）。
- 提供 `doctest.CheckExtension(ext)`，阶段 2 和 4 的每个扩展都要调用它。

阶段 1 验收：`go test ./internal/doc/... -race` 通过；随机测试在 CI 上每次跑固定种子，本地可以用环境变量 `CORRA_FUZZ_ITERS` 加大迭代次数。

---

## 阶段 2：`internal/mdcodec` 与基础扩展

### 2.1 扩展骨架（S）

文件：`internal/ext/ext.go`、`internal/ext/std/std.go`

- 按设计 §7.1 定义 `Extension`。阶段 2 先不带 `Views`，阶段 3 再加，避免提前依赖 `editor`。
- `std.Extensions()` 返回内置扩展清单；`ext.Build(exts) (*doc.Registry, *mdcodec.Rules, ...)` 按 `Priority` 合并。

### 2.2 解析（M）

文件：`internal/mdcodec/parse.go`

- 用各扩展的 goldmark `Extender` 搭 goldmark 实例。复用 `internal/mdext` 里现有的扩展：高亮、上下标、数学、alert、frontmatter、TOC、图片尺寸。
- `Parser` 的建树 API：`Open`、`Close`、`AddText`、`AddAtom`、`WithMark`、`Children`。
- 兜底规则（设计 §4.1）：块级没有处理函数的节点 → `raw` 块，内容是原文行；行内 → 带 `raw_inline` mark 的文本。
  - 容器节点的原文范围由第一个和最后一个后代叶子的 `Lines()` 推出；推不出来时退回整个父块的原文。
- 输出 `*doc.Node`（doc 根），节点 ID 全新。

### 2.3 写出（M）

文件：`internal/mdcodec/write.go`、`style.go`

- `Writer`：前缀栈、`WithPrefix`、`Inline`、`Text`（按上下文转义）、`Raw`、`Close`、紧凑和松散列表（设计 §4.2）。
- mark 按 `Rank` 开合；交叉的 mark 拆成合法嵌套；mark 两端的空格移到 mark 外面（`** a**` 不是合法强调）。
- `Style`：无序列表符号、强调符号、加粗符号、代码围栏、有序列表分隔符；默认值放在 `style.go`，阶段 3 接入设置。
- 写出时可选记录源码映射：`[]{Leaf, LeafOffset, SourceOffset}`，供阶段 3 的源码模式换算光标。

### 2.4 基础扩展（M）

每个扩展一个包，包含 schema 和 Markdown 解析、写出：

| 包 | 节点和 mark |
|---|---|
| `ext/core` | doc、paragraph、text、hard_break、raw、raw_inline |
| `ext/heading` | heading{level} |
| `ext/blockquote` | blockquote |
| `ext/list` | bullet_list、ordered_list{start, tight}、list_item；命令 sink、lift、拆分列表项 |
| `ext/task` | list_item 的 checked 属性和对应命令 |
| `ext/code` | code_block{language} |
| `ext/hr` | horizontal_rule |
| `ext/emphasis` | bold、italic、strike、code、link{href, title} |
| `ext/image` | image（行内原子）{src, alt, title, width, height} |

每个包都要调用 `doctest.CheckExtension`，覆盖设计 §7.3 的第 1、2 条（语义往返、写出不动点）。

### 2.5 语料测试和对比工具（S）

- `internal/mdcodec/corpus_test.go`：读 `CORRA_CORPUS` 环境变量（沿用 `richtext/zz_corpus_test.go` 的约定）和内置样例：仓库里的 `README.md`、`docs/**/*.md`、`internal/mgdoc/codec_test.go` 里的 `corpus`，以及从 `richtext` 和 `nativerich` 测试里提取的 Markdown 片段。
  - 断言：能解析；写出是不动点；语义往返相等。
  - 记录：第一次写出和原文的差异，按设计 §4.3 的已知写法统一归类。出现不在清单里的差异，测试失败。
- `cmd/mdroundtrip`：`go run ./cmd/mdroundtrip file.md` 打印原文和写出结果的 diff，开发时人工检查用。

阶段 2 验收：语料测试通过；人工抽查 10 篇真实笔记的 diff，确认都属于已接受的写法统一。

---

## 阶段 3：编辑器 MVP

位置：`internal/editor`。从阶段 0 的原型里挑能用的代码搬过来，原型目录删掉。

### 3.1 视图规格与主题（S）

- `ViewSpec`、`TextLook`、`ContainerLook`、`Box`、`Env`（设计 §5.2）。
- `Extension` 加上 `Views` 字段；阶段 2 的基础扩展补上视图规格。
- `Env` 从 `views/theme.go` 取颜色和字体，包括深色模式、缩放、正文和等宽字体设置。

### 3.2 布局（L）

文件：`layout.go`、`layout_cache.go`

- 布局树和文档树结构对应；文本块用 `ui.TextLayout`，mark 通过 `ViewSpec.Mark` 转成 `ui.Span` 样式。
- 缓存 key 是 `(节点指针, 宽度, 主题版本)`；提交后只重新排版指针变了的路径（设计 §5.3）。
- 每个容器保存子块高度前缀和；只排版可见区域加上下各一屏，其余用估计高度。

测试：用 MyGo 的无窗口测试（`ui.Tester` 或等价物）检查：改一个段落后其余段落的布局对象复用（指针相同）；宽度变化后全部重排。

### 3.3 绘制（M）

- 文本、选区底色（跨块连成一片）、光标、组字下划线。
- 装饰：项目符号、序号（按 `ordered_list.start` 递增）、复选框、引用竖线、代码块底色、分隔线。

### 3.4 命中测试与光标移动（L）

- 点击、双击选词、三击选段、拖动选择、Shift+点击扩展选区。
- 方向键（字素和词）、上下移动时保持列、Home/End、行首行尾、文首文末、翻页、全选。
- 原子块：点击得到 `NodeSelection`；方向键可以越过原子块，也可以选中它。

### 3.5 输入（M）

- `TextInputClient` 实现（设计 §5.4）：线性文本用 `GlobalOffset` 和 `PointAt` 换算；组字状态不进文档，提交时产生一个 `input` 事务。
- 快捷键表；`Enter`、`Backspace`、`Delete`、`Tab`、`Shift+Tab` 的 `FirstOf` 链；⌘/Ctrl + B、I、U、K、Z、Shift+Z。
- 输入规则：`# `、`- `、`1. `、`> `、`` ``` ``、`**x**`、`` `x` ``；触发后紧接着按一次退格，撤回成字面文本。

### 3.6 剪贴板（M）

- 复制：内部格式（doc 片段）、Markdown、HTML、纯文本同时写入。
- 粘贴：内部格式 → HTML（复用 `nativerich/htmlimport.go` 的 `HTMLToMarkdown`，再走 `mdcodec`；`nativerich` 包在阶段 5 会被删除，所以先把这个文件移到独立的包，例如 `internal/htmlmd`）→ Markdown → 纯文本。
- 剪切和拖放文本。

### 3.7 撤销、查找、滚动（S）

- `History` 接到 ⌘Z 和 ⇧⌘Z；撤销后把光标滚到可见区域。
- 查找替换：在线性文本上搜索，结果存为 `Point`，按提交的日志平移；全部替换是一个事务。

### 3.8 接入应用（M）

- 新类型 `views.DocEditor` 实现 `views.RichEditor`：
  - `Load` = 解析；
  - `Snapshot` = 自加载后有过提交才写出，否则原样返回 `source`（设计 §4.3）；
  - `Format(command)` 按命令名调用扩展的 `Commands`。
- 设置项"新编辑器（预览）"，默认关闭。打开后 `main.go` 在所有平台都用 `DocEditor`，不再创建 `nativerich` 或 `MygoRich`。
- **源码模式切换**（设计 §6）：进入时写出并带源码映射，光标换算过去；退出时解析、做 ID 协调，作为一个 `source` 事务提交。
- **外部文件变化**：沿用现有的冲突处理；没有本地修改时重新解析，用和源码模式相同的 ID 协调，作为 `External` 事务提交，不进撤销历史。

**阶段 3 验收（闸门）**：

- [ ] 把 `views/rich_mygo_test.go`、`views/interactions_test.go` 里富文本相关的行为测试改写成针对 `DocEditor` 的版本，全部通过
- [ ] 三平台手动清单：输入、输入法、选择、格式、列表缩进、撤销重做、复制粘贴、源码模式往返、外部修改
- [ ] 性能：1 MB 的混合文档，按键到绘制的 P50 和 P99 延迟（记录数字，和现有编辑器对比，不能更差）；打开文档的时间
- [ ] 内存：连续输入 10 分钟后，历史和布局缓存的内存占用

---

## 阶段 4：功能迁移成扩展

每个扩展的完成标准都一样：

- 包含 schema、Markdown 规则、视图、命令、快捷键、斜杠菜单项；
- 通过 `doctest.CheckExtension`；
- 有截图测试（沿用现有 `zz_*_shot` 测试的做法）；
- 行为对照现有 macOS 实现（`nativerich` 里对应的测试用例）。

建议顺序，前面的项目是后面的依赖或者最常用：

| # | 扩展 | 说明 | 量 |
|---|---|---|---|
| 4.1 | 斜杠菜单、格式条、块手柄（拖动、菜单） | 编辑器层的通用 UI，后面的扩展往里注册项目 | L |
| 4.2 | `table` | table、table_row、table_cell；Tab 切换单元格；加行加列；对齐；`CellSelection` | L |
| 4.3 | `image` 增强 | 块级图片、尺寸拖拽、粘贴和拖入时保存到 `assets/`、图床上传（复用 `app/images.go`、`app/hosting.go`） | M |
| 4.4 | `math` | 行内和块级；编辑用源码输入框；渲染复用 `internal/mathjax`、`mtex`；位图缓存按内容哈希 | M |
| 4.5 | `diagram`、`chart` | mermaid、d2、goat、chart 代码块渲染成预览，复用 `render/diagram.go`、`chart.go`；可切换到源码 | M |
| 4.6 | `footnote` | footnote_ref（行内原子）、footnote_def（容器）；编号作为派生索引；悬停卡片；跳转 | M |
| 4.7 | `callout` | 设计 §7.2 的例子 | S |
| 4.8 | `highlight`、`supsub`、`kbd`、`color` | mark 扩展；颜色调色板 UI | S |
| 4.9 | `toc`、`frontmatter` | 原子块；frontmatter 显示为可折叠的属性块 | S |
| 4.10 | `page_card`、`mention`、`[[` 补全、表情补全 | 复用 `views/link.go` 的 `LinkCandidates`、`nativerich/page_info.go` 的 `PageInfo` | M |
| 4.11 | 编辑器设置 | 字体、缩放、专注模式、打字机模式、代码行号、公式编号 | S |
| 4.12 | 拼写检查 | 按平台调系统接口；可以放到阶段 5 之后 | M |

阶段 4 验收：对照 `README.md` 的功能列表和 `docs/typora-parity.md`，列出每一项在新编辑器里的状态；除 4.12 外全部完成。

---

## 阶段 5：切换默认并删除旧实现

### 5.1 切换默认（S）

- 设置开关默认打开；保留一个版本周期，让用户可以切回旧编辑器。
- 发布说明写明：
  - macOS 失去的系统功能（设计 §5.5）及替代方案；
  - 第一次编辑后保存时会统一 Markdown 写法（设计 §4.3）。

### 5.2 删除（M）

- 删除 `nativerich/`、`richtext/`、`internal/mgdoc/`、`views/rich_mygo.go`，以及 `views` 里只为它们存在的代码（`inlineRich`、`NativeRich()` 分支等）。
- `RichEditor` 接口精简成新编辑器真正需要的方法；`main.go` 去掉平台分支。
- `editor/undo.go`：确认源码模式只用 MyGo `ui/editor` 自带的撤销后删除；`editor/` 包里的格式化动作和搜索如果还在源码模式里用，就保留。
- `Makefile` 和 CI：`render/mathtext_darwin.go` 仍然用 cgo，所以 macOS 上的 `CGO_ENABLED=1` 要保留，只更新注释。
- 更新 `README.md`、`docs/handover.md` 的代码地图。

验收：全量测试通过；`go build` 在三平台通过；代码行数的变化写进 PR 描述。

---

## 阶段 6（可选）：导出直接从树渲染

现在的导出走的是 `render.Parse(Markdown)` → HTML / PDF。新编辑器写出 Markdown 以后，这条路径不用改也能工作，所以这一阶段不急。

要做的话：

- 每个扩展实现 `HTML` 渲染；
- `render` 增加从 `*doc.Node` 构造 `render.Document` 的入口，PDF 管线不变；
- 对比现有导出结果的截图，保证一致。

---

## 风险跟踪

| 风险 | 在哪一步暴露 | 应对 |
|---|---|---|
| MyGo 升级有大量破坏性变更 | 0.1 | 升级单独一个 PR；必要时先升到中间版本 |
| 某个平台的输入法或无障碍做不出来 | 0.2 | 闸门；备选是那个平台保留原生文本控件作为输入层 |
| 大文档的按键延迟 | 0.2 记数字，3.8 闸门 | 根节点子节点切片换成分块持久化序列（设计 §3.1）；布局只排可见区域 |
| goldmark 的解析差异让写出的结果不是不动点 | 2.5 | 语料测试会暴露；在对应扩展的写出里修正 |
| 功能迁移的工作量被低估 | 阶段 4 | 每个扩展单独排期；旧编辑器在阶段 5 之前一直可用 |
| macOS 用户感知到的退化 | 5.1 | 保留一个版本周期的回退开关；拼写检查优先补 |

---

## 进度记录（2026-11-24）

分支 `document-model`，未提交。全部测试在 Linux arm64 上通过（含 `-race`），Windows 和 macOS（不带 cgo）交叉编译通过。

### 已完成

| 阶段 | 内容 | 主要位置 |
|---|---|---|
| 0.1 | MyGo v0.7.3 → v0.10.0，无需改代码 | `go.mod` |
| 0.2 | 编辑器原型（已删除），结论见 `2026-11-24-editor-spike.md` | — |
| 1 | 文档模型：不可变树、schema、四个原语、可反转操作日志、规范化（含祖先）、撤销、命令、片段复制、ID 协调 | `internal/doc` |
| 2 | Markdown 编解码与基础扩展：core、heading、blockquote、list（含任务）、code、hr、emphasis、image | `internal/mdcodec`、`internal/ext/*` |
| 3 | 自绘编辑器：布局缓存、绘制、输入法、键盘与鼠标、剪贴板、撤销、查找替换、输入规则、源码模式往返（一次撤销）；`DocRich` 接入设置开关"新编辑器（预览）" | `internal/editor`、`views/rich_doc.go`、`main.go` |
| 4 | callout、math（行内与块，MathJax 渲染）、marks（高亮、上下标、下划线、kbd、颜色）、footnote、frontmatter、toc、table（含编辑命令与横向布局）、图片加载与 `=尺寸`、图表与 chart 预览、斜杠菜单、格式条、`[[` 笔记链接、表情补全、页面卡片、笔记提及、Cmd+点击打开链接、专注模式、打字机模式 | `internal/ext/*`、`internal/editor` |

测试覆盖：
- `internal/doc`：随机命令序列（默认 6000 条，`CORRA_FUZZ_ITERS` 可加大，已跑过 30 万条），每步检查 schema、旧文档不变、操作日志正反重放、撤销重做。
- `internal/ext/std`：仓库全部 Markdown 的语料往返；随机文档往返（已跑过每个种子 3000 篇）；随机命令；扩展语法的逐例测试。
- `internal/editor`：用 MyGo `Tester` 走真实输入路径的行为测试（输入、输入法、回车退格、输入规则、列表、表格、复制粘贴、查找替换、源码往返、斜杠菜单、格式条、图表预览）。

性能（Linux arm64，无窗口）：1000 段文档中按键并完成一帧绘制约 0.71 ms。

### 未完成

**阶段 4 剩余**
- 已知限制：`goat` 图表在 oksvg 下栅格化失败（旧实现同样如此），预览显示错误信息。其余项目（块手柄、图片保存与上传、代码行号、公式编号、拼写检查）均已完成，见 2026-11-25 的进度记录。

**阶段 0 / 3 的手动验收（这台机器做不了）**
- 三平台：中文、日文输入法；读屏器；macOS 带 cgo 构建与 `nativerich` 测试；`go tool mygo build` 打包。
- 三平台手动清单（计划阶段 3 验收表）。
- 拼写检查的三平台实测：macOS 已在本机验证（NSSpellChecker 真实调用，测试通过）；Windows ISpellChecker、Linux hunspell 只做了交叉编译与 vet，需要在真机上确认。

**阶段 5.2（删除旧实现）** 于 2026-11-25 执行完毕：用户明确指示「直接移除旧版实现」，跳过了原定的三平台手动验收闸门（闸门本意是防止删除不可逆实现后无法回退；接受该风险）。删除内容与善后见下方进度记录。5.1 的 `LegacyEditor` 回退开关随之移除。

**阶段 6（可选）** 没做：新编辑器写出 Markdown 后，导出仍走现有 `render` 管线，可以正常工作。

### 实现中对设计的修正

已同步到设计文档的：
- `Can` 用丢弃的事务干跑，不用 `tx == nil`（§3.3）。

其他（设计文档未细写的实现决定）：
- 任务列表并入 `ext/list`：属性只能由节点类型自己声明。
- 规范化会检查改动节点的祖先（表格的行列对齐依赖整张表）。
- 公式块是纯文本块（文本即 TeX），预览画在源码下方：为此 `TextLook` 增加了 `Below` 区域，图表预览复用同一机制。
- 补全弹层通用化为 `Completion`（`/`、`[[`、`:` 都是它）。
- 视图的后台工作（图片、图表）通过 `Env.Refresh` 通知编辑器重新排版。
- 写出器的若干 Markdown 规则都是随机测试发现后加的，例如：强调符号两侧字符编码、相邻同字符分隔符换写法、紧凑列表只在安全时使用、空列表项后不空行、换行延迟写出、文档首行的 `---` 改写为 `***`。

---

## 进度记录（2026-11-25）

### 4.12 拼写检查：三平台系统接口

新包 `internal/spell`，`spell.New()` 在 `views/rich_doc.go` 接到编辑器的 `cfg.Spell`，设置里的"拼写检查"开关从此对新编辑器生效。

| 平台 | 实现 | 说明 |
|---|---|---|
| macOS | cgo/ObjC 调 `NSSpellChecker` | 整段一次检查，`checkSpellingOfString:startingAt:` 循环；返回的 NSRange 就是 UTF-16 偏移，与编辑器同单位。经 `mygo.RunOnMain` 保证 AppKit 主线程 |
| Windows | syscall COM：`ISpellCheckerFactory` → `ISpellChecker::Check` | GUID 与 vtable 顺序对照 mingw-w64 `spellcheck.h` 核实；专用 goroutine `LockOSThread` + STA 公寓串行服务；用户 locale 的语言，退回 en-US |
| Linux | purego dlopen `libhunspell`（CGO_ENABLED=0 可用） | Go 侧分词（`internal/spell/spell.go`：字母成词、撇号连词、CJK 等无空格文字整段跳过）；词典按 locale 优先在常见发行版路径里找，找不到静默降级 |

- 无后端的平台（含 macOS 无 cgo 交叉编译）`New()` 返回 nil，行为与从前一致。
- 测试：`spell_test.go` 分词表驱动（UTF-16 偏移、代理对、CJK）；`darwin_cgo_test.go` 真机 NSSpellChecker（无词典环境自动跳过）。`go test ./...` 804 个测试全部通过；三平台 `go build`、`go vet` 通过。
- 尚未在 Windows/Linux 真机上实测（见上方"未完成"）。

### 5.2 删除旧实现（2026-11-25，用户明确指示执行）

用户指示「直接移除旧版实现」，跳过原定的三平台手动验收闸门（删除不可逆，接受风险）。删除后以 git 历史为准可随时找回。

删除：

- `nativerich/`（整个 AppKit NSTextView 桥，含全部 .h/.m/测试）、`richtext/`、`internal/mgdoc/`、`views/rich_mygo.go`、`views/rich_mygo_test.go`。
- `LegacyEditor` 设置项与回退分支：`app/settings.go`、`app/state.go`、`views/settings.go`（实验区整节删除）、`menu.go` 的 `NativeRich()` 分支、`main.go` 的双路径初始化。
- `RichEditor` 接口瘦身：移除 `SetFrame`/`SetVisible`/`Print`/`NativeRich`；可见性改为 `richPane` 内由 `richVisible()` 派生（`inlineRich` 接口保留）。

搬迁（旧包里仍被新编辑器复用的部分）：

- `nativerich/htmlimport.go` → `internal/htmlmd/htmlmd.go`（原样保留，仅改包名）。
- 侧栏页面卡片的标题/摘要读取改为新树：`views/rich_doc.go` 的 `docPages`（走 `kit.Codec.Parse`，首标题+首段落，140 rune 截断）。
- `richFontSize` 常量与 `mygoCorpus` 测试语料移入 `views/rich_doc.go` / `rich_doc_test.go`。

验证：

- `go test ./...` 387 个测试全绿；`go vet` 干净；三平台 `go build` 通过（Windows/Linux 为 CGO_ENABLED=0 交叉编译）。
- 规模：128 个文件改动，删除 28,479 行，新增 178 行 + `internal/spell` 585 行，**净删约 2.77 万行**（PR 描述引用此数）。
- `Makefile` 注释更新为「数学排版需要 cgo」；CI 本就只在 macOS 开 cgo，无需改动；README 无旧实现引用。
- `docs/handover.md` 整体重写为新架构版本。
