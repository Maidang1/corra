# 成熟富文本编辑器的数据建模，以及能用到 corra 的部分

范围：ProseMirror、Tiptap、Lexical、Slate、Milkdown、MDXEditor、Toast UI Editor、Vditor（含 Lute）、CodeMirror 6。每个方案一份文档，都是读官方文档和 GitHub 源码写成，每条结论标了核对程度。本文是汇总。

读的版本：ProseMirror 各包 2026-04-01 的提交，Tiptap 3.30.3，Lexical 0.53.0，Slate 0.147.4，Milkdown 7.23.0，MDXEditor（Lexical 0.48），tui.editor 3.2.3，Vditor 4.0.1，MyGo v0.7.3，goldmark v1.8.6。

## 文档索引

| 文档 | 模型一句话 |
|---|---|
| [prosemirror.md](prosemirror.md) | 有 schema 的节点树，用 step 组成 transaction，位置是整数偏移，用 mapping 修正位置 |
| [tiptap.md](tiptap.md) | ProseMirror 之上的声明层和命令层（Node/Mark/Extension、`chain()`、`can()`），没有新的数据模型 |
| [lexical.md](lexical.md) | 以 key 为索引的不可变节点 Map，`getWritable` 写时复制并记 dirty 集合，历史存整个 EditorState |
| [slate.md](slate.md) | 纯 JSON 树，Path 定位，9 种 Operation，每种可反转，规范化靠 dirty path |
| [milkdown.md](milkdown.md) | remark/mdast 解析，映射到 ProseMirror 节点，再用 remark-stringify 写回 |
| [mdxeditor.md](mdxeditor.md) | mdast 经访问者映射到 Lexical 节点，复杂块把 mdast 原样存起来 |
| [toast-ui-editor.md](toast-ui-editor.md) | v3 里 Markdown 模式和 WYSIWYG 模式各一个 ProseMirror 文档，切换时整份互转 |
| [vditor.md](vditor.md) | 状态在 contenteditable DOM 里，每次输入对光标所在块调用 Lute 往返 |
| [codemirror6.md](codemirror6.md) | 纯文本是唯一真相，ChangeSet 描述修改，装饰层叠加显示 |

## 横向比较

| 方案 | 文档是什么 | 位置 | 变更 | 撤销 | 未知 Markdown 语法 |
|---|---|---|---|---|---|
| ProseMirror | 有 schema 的树 | 整数偏移 | Step，可 map 和反转 | 反转 step | 序列化器默认抛错 |
| Lexical | key 到节点的 Map | key 加偏移 | 写时复制加 dirty 集合 | 整个 EditorState 快照，结构共享 | 无官方表格、公式转换器，导出有损 |
| Slate | JSON 树 | Path 加偏移 | Operation，可反转 | 反转操作，栈上限 100 | 社区方案丢位置和分隔符写法 |
| CodeMirror 6 | rope 文本 | 一维偏移 | ChangeSet | 反向 ChangeSet | 不涉及，文本就是源码 |
| Milkdown | ProseMirror 树 | 同 ProseMirror | 同 ProseMirror | 同 ProseMirror | 抛 `parserMatchError`，没人捕获 |
| MDXEditor | Lexical 树 | 同 Lexical | 同 Lexical | 同 Lexical | 抛 `UnrecognizedMarkdownConstructError`，整篇加载失败 |

有三点在所有方案里一致：

1. 没有一个方案在模型里保留 Markdown 源码位置。Milkdown 只在解析时读 `position.offset`，MDXEditor 从不读，Lute 的 AST 里搜不到位置字段。
2. 没有一个 Markdown 编辑器对未知语法有逐块的原文兜底。它们要么抛错，要么静默丢弃，只有少数复杂块（表格、指令、JSX、frontmatter）以原始 mdast 存储并原样写回。
3. 它们每次导出都重写整份文档，格式会被规范化（`remark-stringify` 实测：setext 标题变 ATX，`-`/`+`/`_` 变 `*`，`~~~` 变反引号围栏，缩进代码变围栏代码）。

## corra 现状对比

corra 的做法是「按空行切块，行内用私有区字符占位，受保护块带原文，保存时还原」。这个做法在保真上比上面所有方案都强：未知语法落到受保护块，其他块不受影响，`stable()` 检查失败时整块退回原文。corra 的短板在三处：保护单位太粗，撤销有两套，私有区占位符有冲突。

## 可以用到 corra 的部分，按优先级

### 1. 修复私有区占位符冲突（已复现，是 bug）

`tokenize` 用 U+F0000 起的字符做占位，`Save` 里遇到这个范围内的字符就当占位符还原，但不检查字符是否来自占位。实测：

```
输入: "$a$ and $b$ and \U000F0001 end\n"
输出: "$a$ and $b$ and $b$ end\n"
```

源文里的 U+F0001 被替换成了 `$b$`。位置在 `internal/mgdoc/classify.go:17` 和 `codec.go:209`、`codec.go:456`。触发条件罕见（私有区 15 号平面字符），但后果是静默改写用户文件。

修法有两种：Load 时发现源块含 `>= rawBase` 的字符就把整块当受保护块；或者每次按源文里没用到的区间动态选 `rawBase`。第一种更简单。

### 2. 源码模式的撤销改成增量记录

来源：CodeMirror 6 的 ChangeSet、Slate 的 `insert_text`/`remove_text` 合并。`editor/undo.go` 现在存整份源码快照，200 步。可以在 `Push` 内部用前后缀 diff 算出 `{offset, removed, inserted}`，存记录而不是整份文本，调用点不用改。收益随文档变大而增加。

### 3. 先理清两套撤销，再决定要不要动富文本撤销

多个调研独立发现：MyGo 的 `richtext.Document` 自带撤销栈（前后 `Text` 快照，200 步，1 秒合并），共享结构，已经是增量成本。富文本编辑走 `MarkSourceDirty`，不推 `editor/undo.go`。所以富文本模式不需要 Slate 的 Path 加 Operation，也不需要 Lexical 的整份状态历史。

要先查清的两点（调研里都没核对）：

- `richtext.Document.SetText` 会清空历史，而每次 `Load` 都调它。哪些场景会触发 `Load`（切换模式、外部文件变化、保存后刷新），决定用户会不会丢撤销。
- macOS 的 NSTextView 路径用的是 AppKit 自己的撤销，和上面两套是第三套。

### 4. 把保护单位从整块缩小到行内

来源：ProseMirror 和 Tiptap。`inlineRaw` 已经支持行内原文附件。现在一个段落里出现一个不认识的语法，整个块都会退回原文。缩小后，只有那一小段变成原文附件，其余部分仍可富文本编辑。

### 5. 块类型注册表加命令表

来源：Tiptap 的 `Node.create`、`Mark.create` 和 `(…args) => (props) => boolean` 命令。corra 的块类型散在 `classify`、`blockParagraph`、`Save` 里，`Format(command string)` 是字符串分发。可以建一个注册表，每种块声明自己的分类、渲染、保存和命令。新增语法（如 callout、新图表）只需注册一次。这是扩展性上最直接的收益，但属于重构，没有 bug 驱动。

### 6. 保存和快照按块增量

来源：Lexical 的 dirty 集合。`Snapshot` 和 `Save` 每次整篇 O(n) 重写，数学和图表缓存在撤销时因 `c.math = nil` 整体丢弃。`ParagraphStyle.ID` 在编辑和撤销中保留，可以作为块标识。限制：这个 ID 不写入 Markdown，只在会话内有效。另外没核对 MyGo 能否告知哪个段落被编辑过，这决定这一项能不能做。

### 7. 可选：用 goldmark 只确定块边界

goldmark v1.8.6 已在 `go.mod`。它的叶子块有 `Lines()` 字节范围，容器块（列表、引用、表格）没有。标题范围不含 `#`，围栏代码范围不含围栏行，脚注定义被挪到末尾。建议只用它替代 `Split` 的空行启发式来找块边界，块之间的空隙保持原文。不建议改成「AST 出，AST 进」的模型，因为 goldmark 没有 Markdown 渲染器，而且第三方渲染器的往返保真没核对。

## 不建议采用

- **Path 加 Operation 树（Slate）和 step 加 mapping（ProseMirror）**：富文本撤销已有，源码模式用一维记录就够。
- **Yjs 或 CRDT 协同**：corra 是本地单人编辑，没有需求。
- **AST 出、AST 进的 Markdown 模型（Milkdown、MDXEditor）**：遇到未知语法会抛错，往返会规范化格式，保真不如现状。
- **`@lexical/markdown` 的 transformer**：没有表格、公式、图片转换器，导出有损。
- **直接引入 Lute**：AST 没有源码位置，许可证 Mulan PSL v2 要求保留版权声明并附许可证副本，`lute.go` 无构建标签就 import 了 `gopherjs/js`，三个平台能否原生编译没验证。它内置的 Mark、Sup、Sub、行内数学、脚注、提示块节点可以作为设计参考。可借用的是 Vditor 用字符 `‸` 让光标穿过块往返的做法。
- **CodeMirror 6 的「文本为真相加装饰叠加」**：和 corra「附件是文档内容」的方向相反，需要源码偏移到 richtext 偏移的映射层。调研里提出的折中（光标所在块显示源码，其他块用现有 Codec 渲染）是设计设想，没实现也没验证。

## 没能核对的

- 所有 Markdown 往返丢失项，除 Milkdown 里 `remark-stringify` 的四个实验外，都是读代码推断，没跑样例。
- Lute 的实际输出、往返差异、三平台编译。子代理的权限分类器拒绝了 `go get` 和 `go run`。
- MyGo 层面：能否知道哪个段落被编辑、富文本 Ctrl+Z 的入口、`richtext.Attributes.Tag` 能否表达隐藏的标记符、`ui.TextArea` 有没有范围样式接口。
- macOS NSTextView 路径的撤销行为。
- lexical.dev、milkdown.dev 官网没抓取，改读了仓库里的文档源文件。
- ProseMirror 的 GitHub README 说仓库已迁到 code.haverbeke.berlin，调研读的是 GitHub 克隆，没访问新地址。

## 和上一轮调研的关系

`../frontend-data-modeling/` 是状态和缓存库（Apollo、Relay、TanStack DB、Redux 等）的调研，范围偏离了「富文本编辑器」这个主题。那批文档保留作参考，但与 corra 的相关度低于本目录。
