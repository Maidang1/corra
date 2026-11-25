# Corra 文档数据模型重设计

日期：2026-11-24
输入：`docs/research/rich-text-editors/`（ProseMirror、Tiptap、Lexical、Slate、Milkdown、MDXEditor、Toast UI、Vditor/Lute、CodeMirror 6）
范围：编辑器的数据模型、编辑事务、撤销、扩展机制、Markdown 编解码和渲染层。不沿用现有设计。

## 0. 前提和结论

**前提（已确认）**：不要求对用户文件做字节级保真。要求的是**语义保真**：打开再保存，内容、结构和格式不丢；写法可以被统一（`*` 变 `-`、setext 标题变 ATX、空行数量统一）。

**结论**：

1. **文档是一棵不可变的、有 schema 的节点树，树是唯一真相。** Markdown 只是持久化格式，加载时解析一次，保存时序列化一次。
2. **所有节点类型、mark、Markdown 规则、视图和命令都通过扩展注册**，内核只认识"块容器、文本块、原子"三种结构。
3. **一个渲染层，三平台共用**：在 MyGo 的 `ui.TextInputClient`（输入法）和 `ui.TextLayout`（排版）上自绘编辑器。删掉 NSTextView 路径和 MyGo richtext 组件路径。
4. **一份撤销历史**：撤销存的是旧的根节点。因为树是不可变的，新旧版本共享没改动的节点，所以存旧根很便宜。

```
              ┌──────────────────────── Extension（每种语法一个）────────────────────────┐
              │ NodeSpec/MarkSpec · Markdown 解析与写出 · 视图 · 命令 · 输入规则 · 快捷键 │
              └───────────────────────────────────────────────────────────────────────────┘
 .md 文件 ──goldmark 解析──▶ Doc 树（不可变，NodeID） ──序列化──▶ .md 文件
                                │  ▲
                        布局/绘制 │  │ Tx（命令、输入、粘贴、源码模式回写）
                                ▼  │
                         Editor 视图（MyGo TextInputClient + TextLayout，三平台一套）
                                │
                   派生索引：目录、脚注编号、链接、字数、查找
```

## 1. 从调研里取了什么

| 来源 | 取什么 |
|---|---|
| ProseMirror | schema 声明节点类型；`content` 约束；mark 是挂在文本上的有序集合（`rank`、`inclusive`、`excludes`）；列表是真正的嵌套结构，缩进和反缩进用 lift/sink；序列化器用"前缀栈"写容器（`wrapBlock`） |
| Tiptap | 一个扩展打包 schema、Markdown 规则、命令、快捷键、输入规则；命令返回 bool；`can()` 干跑；`chain()` 合并成一个事务 |
| Lexical | 节点 key 在会话内稳定；不可变状态加结构共享；历史直接存旧状态；dirty 集合驱动增量更新 |
| Slate | 每条操作都带旧数据，反转是纯函数（用 NodeID 寻址，不用 Path）；规范化只在被改动的节点上运行，并设迭代上限，防止规则互相触发不收敛 |
| MDXEditor | 解析器不认识的语法存成原文块，原样写回，不能让整篇加载失败 |
| CodeMirror 6 | 撤销合并规则：同类输入、位置相邻、间隔很短才合并 |

## 2. 文档树

### 2.1 三种结构

内核只有三种结构。其余一切，比如标题、列表、表格、公式、callout，都是扩展注册的节点类型：

| 结构 | 内容 | 例子 |
|---|---|---|
| **块容器**（`ContentBlocks`） | 子块列表 | doc、blockquote、list、list_item、callout、table、table_row、footnote_def |
| **文本块**（`ContentInline` / `ContentText`） | 行内内容 / 纯文本 | paragraph、heading、table_cell；code_block（纯文本，没有 mark） |
| **原子**（`ContentNone`） | 无，数据都在属性里 | 块级：math_block、diagram、image、hr、toc、raw；行内：math、footnote_ref、mention、hard_break |

光标只能落在文本块里。原子块只能整体选中（NodeSelection），或者进入它自己的编辑界面（比如公式源码输入框）。

### 2.2 Go 类型

```go
package doc // 不依赖 MyGo，可以脱离窗口测试

type NodeID uint64 // 会话内递增，不落盘

// Node 创建后不可变。修改总是生成新节点，未改动的子树原样共享。
type Node struct {
	id    NodeID
	typ   *NodeType
	attrs Attrs
	kids  []*Node // ContentBlocks
	text  Inline  // ContentInline / ContentText
	size  int     // 子树在线性文本里的长度（UTF-16），见 2.4
	sizes []int   // kids 的长度前缀和；创建节点时顺手算好，用于把全局偏移定位到叶子
}

// Inline 是文本块的内容，不可变。
type Inline struct{ segs []Segment }

type Segment struct {
	Text  string  // 原子时为空
	Marks MarkSet // 按 rank 排序，没有重复
	Atom  *Node   // 行内原子（公式、脚注引用、提及、硬换行），在线性文本里占 1 个 U+FFFC
}

type MarkSet []Mark
type Mark struct {
	Type  *MarkType
	Attrs Attrs // link{href,title}、color{value}…
}

// Attrs 不可变；键、默认值和校验来自 NodeSpec / MarkSpec。
type Attrs struct{ m map[string]any }
```

各扩展为自己的属性提供带类型的访问器，例如 `heading.Level(n) int`、`list.Ordered(n) bool`、`code.Language(n) string`。业务代码不直接读写 `map`。

### 2.3 Schema

```go
type ContentKind uint8 // ContentNone | ContentInline | ContentText | ContentBlocks

type NodeSpec struct {
	Name      string
	Group     string      // "block"、"list_item"、"table_row"、"inline"…
	Content   ContentKind
	Allow     []string    // ContentBlocks：允许的子 Group 或类型名；ContentInline：允许的 mark（空表示全部）
	Attrs     map[string]AttrSpec
	Inline    bool        // 行内原子
	Isolating bool        // 光标操作不跨出边界（表格单元格、callout）
	Defining  bool        // 粘贴或替换内容时保留自身（code_block、heading）
	Normalize func(n *Node, fix *Fixer) // 可选：修正非法状态
}

type MarkSpec struct {
	Name      string
	Rank      int      // 写出 Markdown 时的嵌套顺序：link 在外，code 在内
	Inclusive bool     // 在 mark 末尾继续打字时是否延续这个 mark（bold 是，link 和 code 不是）
	Excludes  []string // code 排斥所有其他 mark
	Attrs     map[string]AttrSpec
}

type AttrSpec struct {
	Default  any
	Validate func(any) error
}
```

`Registry` 汇总所有扩展的 spec，生成 `*NodeType` 和 `*MarkType`，用指针比较判断类型。

### 2.4 线性文本和位置

MyGo 的 `TextInputClient` 和 `TextLayout` 都用 UTF-16 偏移，模型直接采用同一种单位，省去每次按键的换算。

- **线性文本**：按文档顺序把所有文本块的内容连起来，每个文本块后面跟一个分隔符；行内原子算 1 个单位。`Node.size` 是子树在线性文本里的长度。
- **模型里的位置**：`Point{Leaf NodeID, Offset int}`，即文本块 ID 加块内 UTF-16 偏移。其他块里发生的编辑不会让这个位置失效，只有同一个块里的编辑需要平移它。
- **全局偏移**只给输入法用：沿根节点往下走，每层在 `sizes` 前缀和上二分查找，复杂度是 O(深度 × log 子节点数)。

```go
type Point struct {
	Leaf   NodeID
	Offset int
}

type Selection interface{ isSelection() }
type TextSelection struct{ Anchor, Head Point }
type NodeSelection struct{ IDs []NodeID }                       // 选中整块：图片、公式、多个块
type CellSelection struct{ Table NodeID; Anchor, Head CellRef } // 表格框选
```

### 2.5 按 ID 找节点

`State` 里保存一张持久化映射 `parent: NodeID → NodeID`（HAMT，修改时只复制被改的路径）。打字不改变结构，不用更新这张表；只有插入、删除、移动块时才更新。从 ID 找节点就是沿父链求出路径，再从根往下走。

## 3. 事务、命令、撤销

### 3.1 State 和 Tx

```go
type State struct {
	Doc    *Node
	Sel    Selection
	Stored MarkSet          // 光标折叠时待用的 mark（按 ⌘B 后接着打字）
	parent persistentMap    // 2.5
}

type Tx struct {
	start   State
	doc     *Node
	sel     Selection
	stored  MarkSet
	ops     []Op // 可反转的操作日志，见 3.5
	meta    Meta // Kind："input" | "delete" | "format" | "structure" | "paste" | "source"；AddToHistory
}
```

**只有四个原语**。所有编辑最终都由它们组成：

```go
func (tx *Tx) ReplaceText(leaf NodeID, from, to int, ins Inline)             // 文本块内替换，含 mark
func (tx *Tx) SetAttrs(id NodeID, a Attrs)
func (tx *Tx) SetType(id NodeID, t *NodeType, a Attrs)                         // 段落 ↔ 标题，内容不变
func (tx *Tx) ReplaceChildren(parent NodeID, from, to int, nodes ...*Node)    // 插入、删除、移动块
```

选区和待用 mark 不属于文档内容，单独用 `SetSelection(Selection)` 和 `SetStoredMarks(MarkSet)` 设置。

原语的语义：

- **按顺序生效**：同一个 Tx 里，后一个原语看到的是前一个改完之后的树，下标和偏移都按当前状态算。
- **ID 唯一**：`ReplaceChildren` 插入的节点可以是树里已有的子树（拖动、包进引用时复用同一个指针，ID 不变），但不能让同一个节点在树里出现两次。复制、创建副本、粘贴自己的内容时，必须先 `doc.CloneFresh` 换一批新 ID。测试模式下，提交时检查 ID 唯一性。
- **事务内只复制一次**：Tx 内新建的节点归这个 Tx 所有，后续原语直接原地修改，不再复制。所以给 300 个段落加粗，根节点也只复制一次。例外是已经写进操作日志的节点：它们被冻结，之后再改要重新复制（见 3.5）。提交之后，所有节点都变成不可变。
- **每个原语都记一条操作**：执行时把旧值和新值写进 Tx 的操作日志（3.5）。

在它们之上是一组结构化 helper，也是写命令时最常用的那一层：`InsertText`、`DeleteSelection`、`SplitBlock`（回车）、`JoinBackward`（行首退格）、`AddMark`、`RemoveMark`、`Wrap`（包进引用或列表）、`Lift`（移出一层）、`Sink`（列表项缩进）、`MoveBlock`、`InsertBlocks`。

每次修改都沿路径复制祖先节点，其余节点共享。一个有几千个顶层块的文档，打一个字只复制 doc 节点的子节点切片（几十 KB 以内）和被改的那个段落。如果实测有压力，再把子节点切片换成分块的持久化序列；`Node.kids` 不对外暴露，换实现不影响调用方。

### 3.2 规范化

`Commit` 之前，只对操作日志涉及的节点及其祖先跑规范化。规范化做的修改同样通过原语执行，也记进日志（`Origin = Normalize`）：

- **内核规则**：合并相邻且 mark 相同的文本段；删掉空文本段；检查容器的子节点是否符合 `Allow`，不符合的交给 spec 的 `Normalize` 修正，没有 `Normalize` 就把子节点包进默认块。
- **扩展规则**：比如"列表的子节点只能是 list_item"、"表格每行列数相同"、"文档最后必须是一个可输入的段落"。
- **迭代上限**是被改节点数乘以 10。超过上限时，测试里直接 panic，线上记日志并停止规范化。

规范化之后，树一定符合 schema。这是给扩展作者的保证：拿到的永远是合法的树。

### 3.3 命令

```go
// tx 总是非空。命令返回 false 时不能改动过 tx。
type Command func(s State, tx *Tx) bool

func Can(s State, c Command) bool         // 在一个用完即弃的事务上运行 c；工具栏、格式条、菜单的可用状态
func Chain(cs ...Command) Command         // 多个命令共用一个 Tx，一次撤销
func FirstOf(cs ...Command) Command       // 依次尝试，第一个成功的生效（回车、退格、Tab 的分派）
```

例如 Tab 键 = `FirstOf(table.NextCell, list.Sink, code.Indent, insertTab)`。每个扩展往这个链里加自己的那一项，不需要中央 `switch`。

实现时的修正：最初设计是"`tx` 为 nil 时只判断能否执行"，实现改成了 `Can` 在一个用完即弃的事务上真的执行一次。树是不可变的，丢弃的事务不影响任何状态，命令作者也不用为每个命令写判断和执行两套逻辑。`Chain` 和 `FirstOf` 用保存点回滚没有成功的尝试。

**例子：选中文字按 ⌘B**

```go
func ToggleMark(t *doc.MarkType) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		sel, ok := s.Sel.(doc.TextSelection)
		if !ok || !s.MarkAllowed(sel, t) { // 代码块里不能加粗，按钮置灰
			return false
		}
		if sel.Empty() { // 光标折叠：只切换待用 mark
			tx.SetStoredMarks(s.CurrentMarks().Toggle(t))
			return true
		}
		add := !s.Doc.RangeHasMark(sel, t) // 全是粗体就去掉，否则全加上
		for _, r := range s.Doc.LeafRanges(sel) { // 按文本块切段：[{leaf, from, to}…]
			in := s.Doc.Leaf(r.Leaf).Inline().Slice(r.From, r.To)
			if add {
				in = in.AddMark(doc.Mark{Type: t}) // 内部按 Excludes 处理 code 等互斥 mark
			} else {
				in = in.RemoveMark(t)
			}
			tx.ReplaceText(r.Leaf, r.From, r.To, in) // 长度不变
		}
		tx.SetSelection(sel)
		return true
	}
}
```

工具栏按钮的状态也来自这里：`Can` 决定能不能点，`RangeHasMark` 决定是否显示为按下。

**常见操作拆成原语**

| 操作 | 原语 |
|---|---|
| 设为标题 | `SetType(段落, heading, {level})` |
| 回车拆段 | `ReplaceText` 截掉后半段 + `ReplaceChildren` 在后面插入新块 |
| 行首退格合并 | `ReplaceText` 把当前块内容接到上一块末尾 + `ReplaceChildren` 删除当前块 |
| 跨块删除 | 首块截尾、接上末块剩余部分，再删除中间块；首尾块父节点不同时，在各自层级做 `ReplaceChildren` |
| 列表缩进 / 反缩进 | `ReplaceChildren` 把列表项移进上一项的子列表 / 移出 |
| 包进引用、callout | `ReplaceChildren(父, i, j+1, 容器(原块…))` |
| 拖动块 | 从旧位置删除，把同一个节点指针插到新位置，ID 不变 |
| 表格加行、加列 | `ReplaceChildren` 作用于表格或每一行 |
| 勾选任务、改代码语言、改公式 | `SetAttrs` |
| 链接、颜色 | 带属性的 mark，同加粗 |
| 粘贴多块 | 拆开当前块 + `ReplaceChildren` 插入片段 + 合并两端 |
| 全部替换 | 每处匹配一次 `ReplaceText` |
| frontmatter 等文档属性 | `SetAttrs(根节点, …)` |

### 3.4 撤销

```go
type entry struct {
	before, after State
	kind          string
	leaf          NodeID
	at            time.Time
}
```

- 撤销把 `before` 换回来，重做把 `after` 换回来。旧根和新根共享未改动的节点，一个条目的成本和改动量成正比。
- **合并**：同为 `input` 或同为 `delete`、发生在同一个文本块、位置首尾相接、间隔小于 1 秒，就只更新上一条的 `after`。`format`、`structure`、`paste`、`source` 总是开新条目。
- 上限 200 条，或者按内存预算截断。
- NodeID 在旧根里原样保存，所以撤销后块 ID 不变。悬停控件、书签、渲染缓存都不会失效。
- 全应用一份历史，源码模式的整次编辑算一条（第 6 节）。
- 撤销不使用操作日志。快照更简单，也不会因为某种操作的反转写错而出 bug。

### 3.5 操作日志

每个原语执行时都记录一条带新旧数据的操作，所以 Tx 有一份完整、可以反转的修改记录。这借鉴了 Slate 的做法（每条 Operation 都带旧数据，`inverse` 是纯函数），但寻址用 NodeID，不用 Path。

```go
type OpKind uint8

const (
	OpReplaceText OpKind = iota
	OpSetAttrs
	OpSetType
	OpReplaceChildren
)

type OpOrigin uint8 // Command | Normalize | Source | External

type Op struct {
	Kind   OpKind
	Origin OpOrigin
	Target NodeID // ReplaceText：文本块；SetAttrs、SetType：节点本身；ReplaceChildren：父节点
	From   int    // ReplaceText：UTF-16 偏移；ReplaceChildren：子节点下标
	To     int    // 被替换范围在执行前的终点

	OldText, NewText   Inline    // OpReplaceText
	OldAttrs, NewAttrs Attrs     // OpSetAttrs、OpSetType
	OldType, NewType   *NodeType // OpSetType
	OldKids, NewKids   []*Node   // OpReplaceChildren
}

// Invert 返回反向操作，是纯函数，不需要读文档。
func (o Op) Invert() Op
```

**反转规则**：

| 操作 | 反向操作 |
|---|---|
| `ReplaceText(t, from, to, Old→New)` | `ReplaceText(t, from, from+len(New), New→Old)` |
| `SetAttrs(id, Old→New)` | `SetAttrs(id, New→Old)` |
| `SetType(id, OldType/OldAttrs → NewType/NewAttrs)` | 交换新旧 |
| `ReplaceChildren(p, from, to, OldKids→NewKids)` | `ReplaceChildren(p, from, from+len(NewKids), NewKids→OldKids)` |

**记录时值必须是不可变的**，否则日志会被同一个 Tx 里后面的修改悄悄改掉：

- `Inline`、`Attrs` 本身是不可变值，直接记录。
- 子节点切片在记录时复制切片头，不和父节点共用底层数组。
- 写进日志的节点指针（`OldKids`、`NewKids` 里的节点）会被**冻结**：即使它归这个 Tx 所有，之后再改也要先复制。这只影响"同一个事务里先插入、再修改同一个块"这种少见情况，多复制一次。

记录的成本只是几个指针，旧值本来就被旧根引用着，不额外占内存。

**保证**（7.3 的测试会检查）：

- 正向：在 `Before.Doc` 上按顺序执行日志，结果等于 `After.Doc`，包括 ID。
- 反向：在 `After.Doc` 上按倒序执行每条操作的 `Invert()`，结果等于 `Before.Doc`。

**用途**：

```go
type Commit struct {
	Before, After State
	Ops           []Op
	Meta          Meta
}
```

`Commit` 交给所有订阅者：

- **派生索引和扩展的 `OnCommit` 钩子**按操作增量更新，不用自己比较新旧树。比如只有标题文本变了才更新目录，只有 `footnote_ref` 增删才重新编号。
- **调试**：开发模式下可以把每次提交的日志打出来，看一个命令到底改了什么，规范化又补了什么。
- **外部修改和源码模式回写**：先把新树和旧树的差异转成操作，再走同一条提交路径，`Origin` 分别标为 `External`、`Source`。
- **将来做协同**：日志是翻译成 Yjs 等协同层操作的输入（见第 12 节）。

渲染层不依赖日志，仍然靠节点指针判断布局缓存是否有效（5.3）。日志是给"需要知道改了什么"的消费者用的，渲染只需要知道"变没变"。

## 4. Markdown 编解码

### 4.1 解析：goldmark → 树

继续用 goldmark：Corra 已经依赖它，`internal/mdext` 里也有现成的扩展（高亮、上下标、数学、alert、frontmatter、TOC、图片尺寸）。不要求字节保真以后，goldmark 缺少源码位置就不再是问题。

```go
package mdcodec

type ParseFunc func(p *Parser, n ast.Node) // 用 p.Open/Close/AddText/AddAtom/WithMark/Children 建树

type Rules struct {
	Extenders []goldmark.Extender         // 语法扩展
	Nodes     map[ast.NodeKind]ParseFunc  // goldmark 节点 → doc 节点
}
```

**兜底规则**（内核提供，扩展不能绕过）：

- 块级 goldmark 节点没有对应的处理函数 → 生成 `raw` 块，内容是这个节点覆盖的原文行。显示为等宽文本，可以直接编辑，写回时原样输出。
- 行内节点没有处理函数 → 原文作为文本段，加 `raw_inline` mark，写出时不转义。

所以任何文件都能打开，不认识的内容不会丢失，也不会让整篇加载失败。

### 4.2 写出：树 → Markdown

```go
type Writer struct{ /* 前缀栈、当前行、紧凑/松散、风格配置 */ }

func (w *Writer) Block(n *doc.Node)                      // 按节点类型分派给 spec 的 Write
func (w *Writer) WithPrefix(first, rest string, fn func()) // 容器："> "、"- "、"1. " + 续行缩进
func (w *Writer) Inline(in doc.Inline)                    // 按 rank 开合 mark，必要时转义
func (w *Writer) Text(s string)                           // 按上下文转义
func (w *Writer) Raw(s string)                            // 原样输出
func (w *Writer) Close()                                  // 结束一个块，处理块间空行
```

- **风格配置**（设置里可改）：无序列表符号、强调和加粗的符号、代码围栏字符、有序列表分隔符、表格是否对齐。默认值是最常见的 GFM 写法。
- **mark 交叉**：`**a _b** c_` 这种交叉嵌套，按 `Rank` 拆成合法的嵌套后再写出。
- **Markdown 写不出的格式**（文字颜色、合并单元格）：由扩展自己决定写法，通常是行内或块级 HTML。读入时由同一个扩展识别回来。

### 4.3 什么时候写文件

- 只有提交过事务才写盘。只打开不编辑的文件，内容一个字节都不变。
- 第一次编辑后保存时，整篇按风格配置统一写法，这是已经确认可以接受的。之后每次保存都是稳定的（见 7.3 的不动点测试），不会每存一次就变一次。
- 已知的写法统一：setext 标题变 ATX、列表符号统一、强调符号统一、`~~~` 变反引号、引用式链接变成内联链接、块间空行统一、多余的转义被去掉。

## 5. 渲染层：一个自绘编辑器

### 5.1 选型

用 MyGo 的两组公开接口自绘编辑器：

- `ui.TextInputClient`：接系统输入法（组字、候选框位置、查询文本），文本、选区和撤销都由应用自己管；用 `element.HandleTextInput(client)` 挂上。
- `ui.TextLayout`：带样式的文本排版、光标位置、命中测试，偏移单位是 UTF-16。

macOS 不再用 NSTextView，Windows 和 Linux 不再用 MyGo 的 richtext 组件。三个平台跑同一份 Go 代码，`nativerich/` 下的 ObjC 全部删掉。

### 5.2 视图规格

视图依赖 MyGo，放在 `editor` 包里，不进 `doc` 包。扩展按类型名注册：

```go
package editor

type ViewSpec struct {
	// 文本块：字体、行距、段前段后间距、背景、左侧装饰（项目符号、序号、复选框）
	Text func(n *doc.Node, env *Env) TextLook
	// 容器：内边距、装饰（引用竖线、callout 底色和图标）；子块由引擎排版
	Container func(n *doc.Node, env *Env) ContainerLook
	// 原子和需要自定义布局的节点（表格、公式、图表、图片）
	Box func(n *doc.Node, env *Env) Box
	// mark 的样式
	Mark func(m doc.Mark, env *Env, s *ui.Span)
}

type Box interface {
	Measure(width float32) ui.Size
	Paint(p *ui.Painter, r ui.Rect, st PaintState) // st：是否选中、是否悬停、当前主题
	Hit(pt ui.Point) Hit                           // → 文本位置、整块选择或控件
}
```

### 5.3 布局缓存：用节点指针做 key

节点不可变，所以**节点指针没变，它的布局就一定有效**。这是不可变树带来的最大好处：

- 布局树和文档树结构对应。每个布局节点缓存 `(节点指针, 宽度, 主题版本) → 尺寸和子布局`。
- 提交一个事务后，只有从根到被改节点这条路径上的指针变了。只重新排版被改的文本块，祖先只重新累加子节点高度，其余全部命中缓存。不需要单独维护 dirty 标记。
- 每个容器保存子块高度的前缀和，滚动定位和命中测试都用二分查找。
- 只排版可见区域加上下各一屏，其余按估计高度占位，滚动到时再排版。

### 5.4 输入

- 输入法组字的内容放在编辑器状态里（`preedit{leaf, range, text}`），绘制时叠加在文本块上，不进文档、不进历史。组字结束时一次提交，`Kind = "input"`。
- 键盘事件先查扩展的快捷键表，再查 `FirstOf` 链（回车、退格、Tab），剩下的才作为文本输入。
- **输入规则**（Markdown 快捷方式）：行首输入 `# `、`- `、`1. `、`> `、`` ``` ``、`$$`，行内输入 `**x**`、`` `x` ``，触发对应命令。触发后紧接着按一次退格，撤回转换、恢复字面文本。由扩展注册：

```go
type InputRule struct {
	Match  *regexp.Regexp // 匹配光标前的文本
	Handle func(s doc.State, tx *doc.Tx, m []string) bool
}
```

- **剪贴板**：同时写入纯文本、Markdown 和 HTML 三种格式，以及应用内部格式（一段 doc 片段，带开放深度，粘回 Corra 时结构不丢）。粘贴的优先级：内部格式 → HTML（转成树）→ Markdown → 纯文本。

### 5.5 macOS 失去的东西

拼写检查、听写、系统文本替换、"查询"、Services、系统查找条，这些都来自 NSTextView。查找替换用 Go 自己实现（在线性文本上搜，结果按 `Point` 存）。拼写检查放到后面单独做，按平台调系统接口：macOS NSSpellChecker、Windows ISpellChecker、Linux hunspell。

## 6. 源码模式

源码模式就是 Markdown 的一个编辑视图，不是第二份真相：

1. **进入**：把树序列化成 Markdown，序列化时顺便记一张源码映射（文本块 ID 加块内偏移 ↔ 源码偏移），用它把光标换算过去。用 MyGo 的 `ui/editor` 显示（Corra 现在的源码模式就是它）。
2. **编辑期间**用 `ui/editor` 自带的撤销。
3. **退出**：解析源码得到新树，和旧树做 ID 协调，然后作为一个 `Kind = "source"` 的事务提交，在文档历史里算一条撤销。

**ID 协调**：先按 `(类型, 内容哈希)` 配对新旧子节点，剩下的按顺序和类型配对，配上的沿用旧 ID，配不上的分配新 ID。这样用户在源码模式里只改了一段，回到富文本后其他块的 ID 和布局缓存都还在。

## 7. 扩展机制

### 7.1 一个扩展的组成

```go
package ext

type Extension struct {
	Name     string
	Priority int

	Nodes []doc.NodeSpec
	Marks []doc.MarkSpec

	Markdown mdcodec.Rules            // goldmark 扩展 + 解析处理函数
	Write    map[string]mdcodec.WriteFunc
	HTML     map[string]HTMLFunc      // 导出

	Views map[string]editor.ViewSpec

	Commands   map[string]doc.Command // 命名命令：命令面板、菜单、快捷键都按名字调用
	Keymap     []KeyBinding
	Chains     map[string][]doc.Command // 往 "Enter"、"Backspace"、"Tab" 等链上挂处理函数
	InputRules []editor.InputRule
	Slash      []SlashItem
	Toolbar    []ToolItem
	Indexers   []doc.Indexer // 派生索引：目录、脚注编号、链接引用、字数
	OnCommit   func(c doc.Commit) // 可选：按操作日志响应修改（3.5）
}
```

内置功能全部用同一个接口写，不留内核特权：`core`（doc、paragraph、text、hard_break、raw）、`heading`、`list`、`task`、`blockquote`、`code`、`hr`、`emphasis`（bold、italic、strike、code、link）、`table`、`image`、`math`、`diagram`、`chart`、`footnote`、`callout`、`toc`、`frontmatter`、`highlight`、`supsub`、`color`、`kbd`、`page_card`、`mention`。

### 7.2 例子：callout

```go
var Callout = ext.Extension{
	Name: "callout",
	Nodes: []doc.NodeSpec{{
		Name: "callout", Group: "block", Content: doc.ContentBlocks, Allow: []string{"block"},
		Isolating: true,
		Attrs: map[string]doc.AttrSpec{"kind": {Default: "note", Validate: oneOf("note", "tip", "important", "warning", "caution")}},
	}},
	Markdown: mdcodec.Rules{
		Nodes: map[ast.NodeKind]mdcodec.ParseFunc{mdext.KindAlert: parseAlert}, // 复用 internal/mdext 现有的 alert 解析
	},
	Write: map[string]mdcodec.WriteFunc{"callout": func(w *mdcodec.Writer, n *doc.Node) {
		w.WithPrefix("> ", "> ", func() {
			w.Raw("[!" + strings.ToUpper(kind(n)) + "]")
			w.Close()
			w.Children(n)
		})
	}},
	Views:    map[string]editor.ViewSpec{"callout": {Container: calloutLook}}, // 底色、左边框、图标
	Commands: map[string]doc.Command{"callout.set": setCallout, "callout.unwrap": unwrap},
	Chains:   map[string][]doc.Command{"Backspace": {exitCalloutAtStart}},
	Slash:    []ext.SlashItem{{Title: "提示块", Keywords: "callout note tip", Run: "callout.set"}},
}
```

不用改内核、编解码器、渲染层，也不用写任何平台代码。

### 7.3 注册时强制跑的测试

框架为每个扩展自动生成以下测试：

1. **语义往返**：`parse(write(doc)) ≅ doc`（忽略 ID）。用扩展自带的样例，加上随机生成、符合 schema 的树。
2. **写出不动点**：`write(parse(write(parse(md)))) == write(parse(md))`。保证第一次保存之后，每次保存的结果都相同。
3. **命令后 schema 合法**：在随机文档上随机执行注册的命令，每一步之后树都符合 schema，规范化在上限内收敛。
4. **撤销**：执行后撤销，根节点和执行前是同一个指针。
5. **操作日志**：对每次随机命令的提交，正向重放日志得到 `After.Doc`，倒序执行反向操作得到 `Before.Doc`（3.5）。

## 8. 派生索引

不放进树、由树算出来的数据，在每次提交后根据操作日志（3.5）增量更新：

- 标题列表和 slug（目录、`[[#` 补全、锚点跳转）
- 脚注定义、引用和编号
- 链接、页面提及、图片资源
- 字数统计和阅读时间
- 查找结果（按 `Point` 存，同一个块里有编辑时平移，块被删除时作废）

扩展通过 `Indexer` 注册自己的索引。

## 9. 包结构

```
internal/doc        树、schema、Tx、命令、历史、规范化、索引接口   不依赖 MyGo
internal/mdcodec    goldmark ↔ doc、Writer、兜底规则                  不依赖 MyGo
internal/editor     布局、绘制、输入、剪贴板、查找                    依赖 MyGo ui
internal/ext/...    每个扩展一个包：schema + Markdown + 视图 + 命令
internal/ext/std    内置扩展清单
```

`doc` 和 `mdcodec` 不导入 MyGo，所以模型、编解码、命令都可以不开窗口测试；将来做命令行工具（比如现有的 `aitool`）也能直接复用。

## 10. 迁移计划

分步实现计划见 `docs/superpowers/plans/2026-11-24-document-model.md`。计划把下表的 P1 拆成了"文档模型"和"Markdown 编解码"两个阶段，阶段编号以计划为准。

| 阶段 | 内容 | 验收 | 删除 |
|---|---|---|---|
| P0 原型 | 把 MyGo 从 v0.7.3 升到当前版本；用 `TextInputClient` + `TextLayout` 做最小编辑器：多段落、光标、跨段选区、中文输入法组字、滚动 | macOS 和 Windows 上手动验证；这一步不通过，后面都不做 | — |
| P1 模型 | `doc`、`mdcodec`，加上 core、heading、list、blockquote、code、emphasis、link 扩展 | 7.3 的四类测试；`richtext/zz_corpus_test.go` 和 `mgdoc` 的语料语义往返通过 | — |
| P2 编辑器 MVP | `editor` 包：文本块、列表、引用、代码块、mark、输入规则、剪贴板、撤销、查找、源码模式切换；放在设置开关后面，三平台同时可用 | 现有富文本行为测试改写后通过 | — |
| P3 功能迁移 | 表格、图片、数学、图表、脚注、callout、TOC、页面卡片、提及、表情、`[[` 补全、斜杠菜单、格式条、悬停手柄，逐个写成扩展 | 每个扩展过 7.3 测试，并有截图对比 | — |
| P4 切换 | 新编辑器设为默认 | 全量测试 | `nativerich/`、`richtext/`、`internal/mgdoc`、`views/rich_mygo.go`、`editor/undo.go` |
| P5 导出 | HTML/PDF 从树渲染（各扩展的 `HTML`） | 导出截图对比 | `render/` 里重复的 Markdown 解析 |

## 11. 风险

| 风险 | 影响 | 应对 |
|---|---|---|
| 自绘编辑器的工作量：光标移动（按词、按行、上下移动时保持列）、双向文本、跨块选区、拖放、无障碍 | 最大的一块工作 | P0 先验证最难的输入法和选区；行为对照现有 NSTextView 的测试用例 |
| MyGo 从 v0.7.3 升级有破坏性变更 | P0 的前提 | 升级单独成一个 PR，先让现有功能跑通 |
| macOS 失去 NSTextView 的系统功能（5.5） | 用户可感知的退化 | 拼写检查单独排期；切换前在发布说明里写清楚 |
| goldmark 的语义差异：脚注定义会被挪到文末，引用式链接会被解析掉 | 保存后位置和写法变化 | 属于已接受的写法统一；脚注定义在写出时按引用顺序放回文末 |
| 顶层块很多时，每次按键都复制根节点的子节点切片 | 大文档的输入延迟 | 先实测；必要时换成分块持久化序列（3.1） |
| 规范化规则冲突，不收敛 | 卡死 | 迭代上限加测试里的随机命令序列 |

## 12. 明确不做

- **字节级保真、无损语法树、ChangeSet 映射**：已确认不需要。
- **ProseMirror 的整数令牌位置和 step mapping**：位置用"块 ID + 块内偏移"，除了同一块内的编辑，其他编辑都不会让位置失效，不需要映射。
- **JSON 作为存储格式**：Markdown 就是存储格式，NodeID 不落盘。
- **CRDT 和协同**：单机应用。将来要做的话，可反转的操作日志（3.5）是翻译成 Yjs 等协同层操作的输入，不用改原语本身。但还需要两部分工作：一是把远端操作应用到本地树的规则，二是协同下的撤销（只撤销本地用户的修改，通常交给协同层的 UndoManager）。这两部分都不在本设计范围内。
- **保留 NSTextView 作为 macOS 后端**：会把两套渲染、两套输入、两套撤销带回来。
