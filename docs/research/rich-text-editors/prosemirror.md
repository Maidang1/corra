# ProseMirror 的数据建模

## 0. 范围与来源

调研对象是 ProseMirror 的核心包。对 corra 有参考价值的是数据层，所以本文不讨论 prosemirror-view 的 DOM 渲染。

可信度标签：**[源码]** 表示已在下列提交的源码里核对；**[官方]** 表示已在 prosemirror.net 官方文档核对；**[未核对]** 表示没有验证，只能当作线索。

读到的版本（2026-11 用 `git clone --depth 1` 取得，GitHub 镜像的 README 声明仓库已迁到 code.haverbeke.berlin，本文未访问新地址）：

| 包 | 版本 | commit |
| --- | --- | --- |
| prosemirror-model | 1.25.4 | 6264de0 |
| prosemirror-transform | 1.12.0 | 662b7a9 |
| prosemirror-state | 1.4.4 | ffad5d9 |
| prosemirror-history | 1.5.0 | 445409b |
| prosemirror-collab | 1.3.1 | 7736c6c |
| prosemirror-markdown | 1.13.4 | 6b95bfe |

来源 URL：
- https://prosemirror.net/docs/guide/
- https://github.com/ProseMirror/prosemirror-model（src/schema.ts、node.ts、fragment.ts、mark.ts、resolvedpos.ts、replace.ts、content.ts）
- https://github.com/ProseMirror/prosemirror-transform（src/step.ts、replace_step.ts、mark_step.ts、attr_step.ts、map.ts、transform.ts）
- https://github.com/ProseMirror/prosemirror-state（src/selection.ts、transaction.ts、state.ts）
- https://github.com/ProseMirror/prosemirror-history（src/history.ts）
- https://github.com/ProseMirror/prosemirror-collab（src/collab.ts）
- https://github.com/ProseMirror/prosemirror-markdown（src/schema.ts、from_markdown.ts、to_markdown.ts）

## 1. 文档数据结构

文档是一棵不可变的树。官方文档说节点和 Fragment 是值，更新文档会得到新值，并共享未改动的子树，调用方不应原地修改 **[官方]**。

核心类型 **[源码]**：

- `Node`：字段有 `type: NodeType`、`attrs: Attrs`、`content: Fragment`、`marks: readonly Mark[]`。
- `TextNode extends Node`：叶子文本。它多一个 `text: string`，`nodeSize` 等于字符串长度，空文本节点会抛 `RangeError`。文本是一整段字符串，不是一字符一节点。
- `Fragment`：子节点数组 `content` 加缓存的 `size`。
- `Mark`：`type: MarkType` 加 `attrs`。标记挂在文本节点（以及其他行内节点）的 `marks` 上，不形成树层级。同一段文字的标记是一个集合，按 schema 里的声明顺序排序（`MarkType.rank`）。相邻且 marks 相同的文本节点会被合并。
- `Attrs = {readonly [attr: string]: any}`。

Schema 由 `SchemaSpec { nodes, marks?, topNode? }` 声明，里面是 `NodeSpec` 和 `MarkSpec` 对象 **[源码]**：

- `NodeSpec.content` 是内容表达式字符串，例如 `"block+"`、`"inline*"`、`"(text | image)*"`。`ContentMatch.parse` 把它编译成状态机，`ContentMatch.validEnd` 表示此处能否结束 **[源码]**。
- `NodeSpec.group` 给节点分组，`NodeSpec.marks` 限定允许的标记，`inline`、`atom`、`selectable`、`code`、`defining` 是布尔开关。
- `NodeSpec.attrs` 和 `MarkSpec.attrs` 是 `{name: AttributeSpec}`，`AttributeSpec` 带 `default`，没有默认值的属性必须在创建时给出。
- `MarkSpec.inclusive` 决定光标在标记末尾时输入是否继承标记，`MarkSpec.excludes` 决定标记互斥。
- 叶子节点没有 `content` 表达式（`NodeType.isLeaf` 即 `contentMatch == ContentMatch.empty`）。

`prosemirror-markdown` 的默认 schema 是个现成例子：`doc: {content: "block+"}`、`paragraph: {content: "inline*", group: "block"}`、`heading: {attrs: {level: {default: 1}}, content: "(text | image)*"}`、`code_block: {content: "text*", marks: "", attrs: {params}}`，标记有 `em`、`strong`、`link`、`code` **[源码]**。

JSON 形式 **[源码]**：`{type, attrs?, content?, marks?}`，文本节点多一个 `text`。`Node.fromJSON(schema, json)` 会按 schema 检查类型名。

## 2. 位置与选区

位置是一个整数，按「令牌」计数 **[官方]**：文档开头是 0；进入或离开非叶子节点各算 1；文本节点里每个字符算 1；不含内容的叶子节点（图片）算 1。源码里 `Node.nodeSize` 对叶子是 1，对非叶子是 `2 + content.size`，对文本是字符串长度 **[源码]**。

`<p>One</p><blockquote><p>Two<img></p></blockquote>` 的位置范围是 0 到 13，文档大小是 `doc.content.size`，不含根节点自己的两个边界 **[官方]**。

整数位置本身不带结构信息。`doc.resolve(pos)` 返回 `ResolvedPos`，其中有 `pos`、`depth`、`parentOffset`、`path`（每层三元组：节点、索引、起点偏移），能回答「父节点是谁、在第几个孩子、离父节点开头多远」**[源码]**。`resolve` 对同一文档有缓存（`ResolvedPos.resolveCached`）。

JS 字符串长度按 UTF-16 码元计，所以文本里的位置也是 UTF-16 偏移。这是 JS 的特征，不是模型要求。**[源码]**（`TextNode.nodeSize` 返回 `this.text.length`）

选区 **[源码]**：`Selection` 抽象类持有 `$anchor`、`$head`（`ResolvedPos`），子类有 `TextSelection`、`NodeSelection`（只有 `anchor`，选中一个节点）、`AllSelection`。每个子类实现 `map(doc, mapping)`、`eq`、`toJSON`、`getBookmark`。`TextSelection.map` 用 `mapping.map(head)` 重新解析位置，落到非行内内容里时用 `Selection.near` 找最近合法位置。

## 3. 变更表示

### 3.1 Step

`Step` 是抽象类 **[源码]**，方法是：

```
apply(doc: Node): StepResult          // {doc | null, failed | null}
getMap(): StepMap                      // 位置映射
invert(doc: Node): Step                // 需要「应用前」的文档
map(mapping: Mappable): Step | null    // 被删除时返回 null
merge(other: Step): Step | null
toJSON(): any                          // {stepType: "replace", ...}
static fromJSON(schema, json)
static jsonID(id, stepClass)           // 注册 stepType
```

具体类型：`ReplaceStep(from, to, slice, structure)`、`ReplaceAroundStep(from, to, gapFrom, gapTo, slice, insert, structure)`、`AddMarkStep`、`RemoveMarkStep`、`AddNodeMarkStep`、`RemoveNodeMarkStep`、`AttrStep(pos, attr, value)`、`DocAttrStep` **[源码]**。大多数编辑最后都归结为 `ReplaceStep`。

`Slice(content: Fragment, openStart, openEnd)` 表示一段「可能切开了几层节点」的内容，`openStart`/`openEnd` 是两端打开的深度 **[源码]**。这让「把段落从中间拆开」这类操作能写成一个替换。

`ReplaceStep.invert(doc)` 是 `new ReplaceStep(from, from + slice.size, doc.slice(from, to))`，也就是用旧文档里被换掉的那段做反向步骤 **[源码]**。所以反转需要旧文档，不能只靠步骤本身。

### 3.2 Transform 与 Transaction

`Transform` 持有 `doc`、`steps: Step[]`、`docs: Node[]`（每步之前的文档）、`mapping: Mapping` **[源码]**。`step()` 失败抛 `TransformError`，`maybeStep()` 返回 `StepResult`。`Transaction extends Transform`，另加 `selection`、`storedMarks`、`time` 和元数据 `setMeta/getMeta` **[源码]**。`Command = (state, dispatch?, view?) => boolean`，不传 `dispatch` 时只检查能否执行 **[源码]**。

`EditorState.apply(tr)` 得到新状态，状态里的 `doc`、`selection`、`storedMarks` 和各插件字段都是值 **[源码]**。

### 3.3 位置映射

`StepMap(ranges, inverted)` 的 `ranges` 是扁平数组，每三个数一组 `[start, oldSize, newSize]` **[源码]**。`map(pos, assoc)` 的 `assoc` 为 -1 或 1，决定插入点上的位置偏向前还是偏向后。`mapResult` 另返回 `MapResult`，带 `deleted`、`deletedBefore`、`deletedAfter`、`deletedAcross`，用来判断位置所在内容是否被删。

`Mapping` 是 `StepMap[]` 的序列，可以 `slice(from, to)`，还有 `mirror`（互为镜像的两个映射，用于「撤销再重做」时恢复被删位置）**[源码]**。一个 `ReplaceStep` 的映射只需要 `[from, to-from, slice.size]` 一组数，足以映射所有位置。

## 4. 撤销与协同

### 4.1 撤销

prosemirror-history 不存文档快照，存的是反向步骤 **[源码]**：

- `HistoryState { done: Branch, undone: Branch, prevRanges, prevTime, prevComposition }`。
- `Branch { items: RopeSequence<Item>, eventCount }`。
- `Item { map: StepMap, step?: Step, selection?: SelectionBookmark, mirrorOffset? }`。有 `step` 的 item 存的是反向步骤，没有 `step` 的 item 只存映射。
- 一个撤销事件是若干连续 item，以带 `selection` 的那个为边界。
- 选项默认值：`depth` 100，`newGroupDelay` 500 毫秒。新事务在延迟之后，或与上一次改动的范围不相邻时，才开新事件 **[源码]**（`isAdjacentTo`）。
- 撤销时 `popEvent` 把反向步骤依次应用。其间若有别的事务（比如协同来的远端步骤）插进来，会用 `remapping` 把反向步骤重新映射到当前文档 **[源码]**。
- 新的编辑会清空 `undone`（重做栈）。`Item.merge` 能把相邻的 `ReplaceStep` 合并，所以连续输入是一个 item。

### 4.2 协同

prosemirror-collab 采用中心权威 + 变基（rebase），不是 OT 的变换矩阵，也不是 CRDT **[官方][源码]**：

- 中心服务器给每批被接受的步骤一个递增版本号；客户端提交时带上自己看到的版本，版本不一致就被拒，客户端得先拉取新步骤再重试 **[官方]**。
- `CollabState { version, unconfirmed: Rebaseable[] }`；`Rebaseable { step, inverted, origin }` **[源码]**。
- `sendableSteps(state)` 取出未确认步骤。`receiveTransaction(state, steps, clientIDs)` 收到远端步骤后调用 `rebaseSteps`：先把本地未确认步骤逐个反向应用，再应用远端步骤，最后把本地步骤用 `step.map(mapping)` 映射后重放，映射失败（返回 null）的本地步骤被丢弃 **[源码]**。
- 协同时历史插件靠 `tr.getMeta("rebased")` 得知步骤被变基，并重写自己的 item **[源码]**（`Branch.rebased`）。

这个设计依赖两点：每个 Step 可以 `map`，并且服务器能给出全序。

## 5. 序列化与 Markdown 往返

官方包 prosemirror-markdown 1.13.4 **[源码]**：

- 解析：`MarkdownParser(schema, tokenizer: MarkdownIt, tokens: {[name]: ParseSpec})`，用 markdown-it 做分词，再按 `ParseSpec`（`{block}`、`{node}`、`{mark}`、`getAttrs`、`noCloseToken`、`ignore`）建 `Node`。默认实例是 `defaultMarkdownParser`，配置为 `MarkdownIt("commonmark", {html: false})`。
- 序列化：`MarkdownSerializer(nodes, marks, options)`，每个节点是函数 `(state, node, parent, index)`，`MarkdownSerializerState` 提供 `write`、`wrapBlock`、`renderInline`、`text`、`esc`、`closeBlock` 等方法。
- 遇到 schema 里没有序列化函数的节点或标记，默认抛 `Error`；设 `strict: false` 才会只输出内容 **[源码]**。解析时遇到没有 `ParseSpec` 的 markdown-it token 同样抛错。

会丢的信息（由代码结构推断，**[源码]** 可见处理路径，但未做往返实验）：

- 默认配置 `html: false`，原始 HTML 不会被解析成节点。
- 默认 schema 只有 `em`、`strong`、`link`、`code`。表格、数学、脚注、删除线、任务列表等要自己加 schema 和两侧的规则。
- 排版细节不在模型里：列表的缩进、`*` 与 `-` 的选择、强调用 `*` 还是 `_`、空行数量、链接是引用式还是内联式、自动链接。序列化时按序列化器固定风格输出。`tight` 是少数保留下来的列表属性。
- 反斜杠转义由 `esc` 统一重新生成，原文里不必要的转义会消失。

没有「原文保真」机制：schema 之外的内容要么报错要么丢失，社区常见做法是把它们建成带原文属性的 atom 节点。后一句 **[未核对]**。

## 6. 在 MyGo / Go 里能否实现

以下按概念逐项判断。corra 现状见各行的「现状」。

| 概念 | 依赖浏览器吗 | Go 里可行吗 | 说明 |
| --- | --- | --- | --- |
| Node/Fragment/Mark 不可变树 | 否 | 可行 | model/state/transform/history/collab 的逻辑与 DOM 无关。DOM 只出现在 `dom.ts`、`from_dom.ts`、`to_dom.ts`；`prosemirror-state` 对 `prosemirror-view` 只有类型导入 **[源码]**。 |
| Schema 与内容表达式 | 否 | 可行 | `ContentMatch` 是纯状态机。corra 只有一种 Markdown 方言，用 Go 结构体声明允许的子类型就够，不必实现表达式语言。 |
| 整数令牌位置 | 否 | 可行 | 但 corra 的 MyGo `richtext` 用 rune 偏移，段落边界算一个换行符 rune，与 ProseMirror 的 2 令牌边界不同。 |
| Step / StepMap / Mapping | 否 | 可行 | 数据结构简单，几百行 Go。 |
| history 反向步骤 | 否 | 可行 | 纯数据。 |
| collab 变基 | 否 | 可行但 corra 暂不需要 | corra 是单机编辑器。 |
| DOMParser、DOMSerializer、EditorView | 是 | 不适用 | MyGo 没有 DOM，需要自己写绘制和输入层，MyGo 的 `richtext.EditorBase` 已经提供。 |

### 6.1 corra 当前做法与 ProseMirror 的对应

读过 `internal/mgdoc/codec.go`、`blocks.go`、`split.go`、`views/rich_mygo.go`、`editor/undo.go`，并读了 MyGo v0.7.3 的 `ui/richtext/document.go`、`text.go`：

- corra 没有自己的文档树。`Codec.Load` 用 `Split` 按空行切块，每块要么交给 `richtext.ParseMarkdown`，要么变成只含一个 `ObjectReplacement` 的段落，附件的 `Data` 是 `rawRef{raw, block}`。这相当于「只有一层段落、受保护块是 atom 叶子」的最小 schema。
- 行内的 mark、sup、sub 用 `Attributes.Tag` 字符串携带，`Codec.Save` 保存时用 `markOpen` 等私用区字符（U+E001 起）和 `rawBase = 0xF0000` 起的占位符还原原文。ProseMirror 里同样的信息是 `Mark` 与 atom 节点的 `attrs`，不需要占位符字符。
- `stable(t)` 检查 `ParseMarkdown` → `Markdown()` → `ParseMarkdown` 是否不变，不稳定就整块降级为 `blockOther`。这是 corra 自己的「往返保真」办法，ProseMirror 没有对应物。
- MyGo 的 `richtext.Document` 已经有变更日志和撤销：`change{version, start, oldEnd, newEnd}`、`undoStep{before, after Text, selBefore, selAfter, start, oldEnd, newEnd}`、`mapPosition(p, start, removed, inserted, upper)`，上限 200 步、1 秒合并 **[源码，读的是 MyGo 模块缓存]**。`Text` 底层是 rope 加持久化段落序列，`before`/`after` 共享未改动部分，所以存「前后两个 Text」成本不高。这相当于 `StepMap` 的单区间版本，`upper` 参数对应 `assoc`。
- `editor/undo.go` 的快照栈（`snapshot{src, caret}`，最多 200 条，1 秒合并）只在 App 层用于源码模式，文稿模式的撤销走 MyGo 的 `Document`。`Document.SetText` 会清空历史 **[源码]**，所以每次 `Load` 后文稿内的撤销栈为空。App 层如何在两种模式切换时衔接，本次没有逐行核对 **[未核对]**。

### 6.2 建议与改动点

1. **不要照搬 ProseMirror 全套。** corra 的编辑内核在 MyGo 里，位置、选区、撤销已有；缺的是「受保护块」的结构化身份。可借鉴的是 schema 思想：用一个 Go 结构体列出块类型（`blockTable`、`blockMath` 等，已在 `classify.go`），而不是靠 `Attachment.Data` 的类型断言。
2. **给受保护块稳定 ID。** MyGo 的 `ParagraphStyle.ID` 与 `IdentifyParagraphs` 可以做这件事。`text.go` 的注释说 ID 会随文本编辑保留、撤销时带回，但不会写进 Markdown、HTML 和剪贴板，所以它只在会话内有效 **[源码]**。ProseMirror 靠整数位置加映射，corra 可以用段落 ID 表达「某个附件块」，在撤销、重排后仍能找到它。
3. **改 `Save` 的占位符机制要谨慎。** 私用区占位符（`rawBase` 起）有数量上限和与正文冲突的风险；ProseMirror 的做法是 atom 节点直接携带 `raw` 属性。在 MyGo 里对应物就是 `Attachment.Data`，已经是这样，占位符只在 `Save` 内部经过 `richtext.Markdown()` 时才用。若要消除它，需要自己写序列化循环而不是借用 `Text.Markdown()`，改动较大，收益有限。
4. **位置映射可以直接复用 `mapPosition` 的思路。** 查找高亮 `findState.matches [][2]int` 存的是 rune 偏移，文档变化后会失效，现在靠 `lastVersion` 重新搜索。若要保留，需要 MyGo 暴露变更日志；目前 `since` 和 `log` 是包内私有的 **[源码]**，要么向 MyGo 提需求，要么自己在 `Document.Version()` 变化时重搜。
5. **Markdown 往返的教训。** ProseMirror 的做法是 schema 外就报错或丢，corra 的「不稳定就整块保护」更保守，这个方向是对的。问题在于判定粒度是「块」，一个段落里有一个不支持的行内语法，整段就变成不可编辑的源码块。可以考虑把粒度降到行内：只把那一小段变成行内 raw 附件（`inlineRaw` 已经有这个能力，用于行内数学和脚注）。
6. **Go 实现 Step 与 history 的成本。** 若将来想脱离 MyGo 的撤销，自己做 `Step`+`StepMap`+反向步骤，约几百行；难点是 `Slice` 的 open depth，只在有嵌套结构（列表套引用）时才需要。corra 目前是扁平段落加段落样式，不需要。

## 7. 未核对项

- ProseMirror 往返丢失的具体清单没有做实验，只读了解析与序列化代码。
- prosemirror-view 的位置与 DOM 映射（`posAtCoords`、`domAtPos`）没有读。
- app 层在文稿与源码模式切换时如何同步撤销栈，没有逐行核对。
