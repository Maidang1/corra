# CodeMirror 6 数据建模调研

## 0. 版本与来源

CodeMirror 6 由多个 npm 包组成。`codemirror/dev` 仓库只放 issue 和开发脚本，不含源码，README 注明已迁移到 https://code.haverbeke.berlin/codemirror/dev。源码分别从各包的 GitHub 仓库 `git clone --depth 1` 读取，commit 日期均为 2026-04-15 前后。所有包的 `LICENSE` 为 MIT。

| 包 | 版本 | commit |
| --- | --- | --- |
| `@codemirror/state` | 6.6.0 | `9c801279cb83011e6f92af778f4443406e8f1200` |
| `@codemirror/view` | 6.41.0 | `fbff59ba004d80d8c914f64c42586387b08706ac` |
| `@codemirror/commands` | 6.10.3 | `5b9bac974f2c4af3e20b045adef949667872ecad` |
| `@codemirror/language` | 6.12.3 | `8e9700018446d46f23267f6e31da56628d5117c0` |
| `@codemirror/lang-markdown` | 6.5.0 | `8f73fd5013a1aaa3d3319f9165fff3100d159124` |
| `@lezer/markdown` | 1.6.3 | `30fe1b12a53f58ebab66e156a7847484addf610b` |
| `@lezer/common` | 1.5.2 | `d87b56cbe3d6f54edb5b8343ee794f8f96f9c86c` |

其他来源：
- https://codemirror.net/docs/ref/ （官方参考文档由源码里的 `///` 注释生成；本文直接读源码注释，没有抓取该网页）
- https://docs.obsidian.md/Plugins/Editor/Editor+extensions （Obsidian 官方开发者文档，用于第 6 节）

可信度标记：【源码】已读源码核对；【文档】已读官方文档页面；【第三方】仅来自第三方说法；【推断】由源码推断，未运行；【未核对】。

另外读了 MyGo v0.7.3（`~/go/pkg/mod/github.com/egoist/mygo@v0.7.3`，MIT）的 `ui/textbuffer.go`、`ui/richtext/document.go`、`ui/richtext/text.go`，用于第 9 节。

## 1. 数据结构

### 1.1 Text：树状文档

- `Text` 是抽象类，有两种实现：`TextLeaf` 保存一组行字符串，`TextNode` 保存若干子 `Text`，子节点之间隐含换行（`state/src/text.ts`）。【源码】
- 分支因子约 32（`Tree.BranchShift = 5`，注释写 approximate）。`replace(from, to, text)` 通过 `decompose` 把两侧拆开再 `TextNode.from(parts)` 重新平衡，所以是持久化结构，旧版本不被破坏。【源码】
- 行号从 1 起，`lineAt(pos)` 和 `line(n)` 返回 `Line`。位置是 UTF-16 码元偏移：`textLength` 对每行累加 `line.length`（JS 字符串长度）。【源码】

### 1.2 EditorState 与扩展机制

- `EditorState` 持有 `doc`、`selection` 和各 `StateField` 的值；用 `state.field(f)` 读取（`state.ts`）。状态不可变，每个 Transaction 产出新状态。【源码】
- `StateField.define({create, update(value, tr), compare, provide, toJSON, fromJSON})`：字段随每个 Transaction 更新，`compare` 为真则视为未变（`facet.ts` 第 295 行起）。【源码】
- `Facet` 是「多个扩展贡献输入，再合并成一个输出」的配置点；`Extension` 可嵌套数组，`Prec` 控制优先级，`Compartment` 支持动态重配置。【源码】

### 1.3 「纯文本为唯一真相，装饰层叠加」

这是整个设计的核心：`doc` 永远是用户的原文，没有第二份「结构化文档」。语法树（Lezer）、装饰（Decoration）、折叠、查找高亮都是从 `doc` 派生的状态，通过 StateField 或 ViewPlugin 保存，并随 ChangeSet 映射。【源码+推断】

## 2. 位置与选区

- 位置是 `doc` 里的整数偏移。`SelectionRange` 有 `from`、`to`、`anchor`、`head`（由内部 flags 记录方向），`EditorSelection` 持有多个 range 和主 range 下标（`selection.ts`）。【源码】
- 位置映射的入口是 `ChangeDesc.mapPos(pos, assoc = -1, mode = MapMode.Simple)`。`assoc < 0` 表示位置贴着前一个字符，遇到该处插入时留在插入之前；`assoc >= 0` 则移到插入之后。`mode` 为 `TrackDel`、`TrackBefore`、`TrackAfter` 时，位置所依附的字符被删则返回 `null`（`change.ts` 第 110 到 142 行）。【源码】
- `Transaction.newSelection` 在没有显式设置选区时，自动把旧选区 `map` 过本次 changes。【源码】

## 3. 变更表示

- `ChangeDesc` 只记长度信息：`sections` 是成对整数 `[旧长度, 新长度或 -1]`，-1 表示未变；插入是 `(0, n)`，删除是 `(n, 0)`，替换是两个正数（`change.ts` 第 21 到 30 行）。【源码】
- `ChangeSet extends ChangeDesc`，另存 `inserted: Text[]`。它记录文档长度，只能应用到长度相同的文档（`apply` 会检查并抛 `RangeError`）。【源码】
- 关键方法：`apply(doc)`、`invert(doc)`（需要变更前的文档来取回被删内容）、`compose(other)`、`map(other, before)`、`iterChanges`、`iterChangedRanges`、`touchesRange`、`toJSON/fromJSON`。【源码】
- `map` 的注释给出合流性质：`A.compose(B.map(A))` 与 `B.compose(A.map(B, true))` 产生同一文档，并称这是一种基础的操作变换（OT），可用于协同。【源码】
- `ChangeSpec` 可以是 `{from, to?, insert?}`、`ChangeSet` 或它们的数组；多个变更的位置都指向「变更前的文档」。【源码】
- `Transaction` 含 `startState`、`changes`、`selection`、`effects`（`StateEffect`，携带非文本状态变更，如设置装饰）、`annotations`（元信息，如 `Transaction.addToHistory`、`userEvent`、`time`）、`scrollIntoView`（`transaction.ts`）。【源码】

## 4. 装饰（Decoration）

`view/src/decoration.ts`：【源码】

- `Decoration.mark(spec)`：给范围加样式，嵌套产生嵌套 DOM。
- `Decoration.widget({widget, side, block})`：在某位置插入一个控件，不改变文档。
- `Decoration.replace({widget?, inclusive, block})`：用控件替换一段范围，或者不给控件而直接隐藏。「Create a replace decoration which replaces the given range with a widget, or simply hides it.」
- `Decoration.line(spec)`：给整行加属性。
- `WidgetType` 是控件描述：`toDOM(view)`、`eq(other)`、`updateDOM`、`estimatedHeight`、`lineBreaks`、`ignoreEvent`、`coordsAt`。`eq` 用来避免重复创建 DOM。
- `DecorationSet = RangeSet<Decoration>`，`RangeSet.map(changes)` 把整组范围映射过变更（`rangeset.ts`）。

装饰通过 `EditorView.decorations` facet 提供，可以是直接的集合（来自 StateField），也可以是函数（来自 ViewPlugin）。源码注释明确规定：「Only decoration sets provided directly are allowed to influence the editor's vertical layout structure」，函数形式提供的装饰不得引入块级控件，也不得替换覆盖换行的范围。【源码】

`EditorView.atomicRanges` 让光标移动和删除把某些范围当成一个整体；它不阻止程序直接设置选区进入这些范围。【源码】

## 5. Lezer 增量语法树与 Markdown

- `Tree` 由 `NodeType`、子节点和相对位置组成；每个节点有 `from`、`to`，没有任意属性，只有 `NodeProp`（`common/src/tree.ts`）。【源码】
- 增量解析：`language.ts` 的 `ParseContext.changes(changes, newState)` 对每个变更范围调用 `changes.iterChangedRanges`，再用 `TreeFragment.applyChanges(fragments, ranges)` 得到可复用的旧树片段，交给新一轮解析。语法树存放在 StateField 里，`syntaxTree(state)` 读取，`ensureSyntaxTree(state, upto, timeout = 50)` 可在时限内补齐解析。【源码】
- Markdown 解析器 `@lezer/markdown` 在块起始处尝试 `reuseFragment`，满足 `matches(block.hash)` 就整块复用旧节点（`markdown.ts` 第 742 行）。所以复用粒度是块。【源码】
- 语法标记符是独立节点，带位置：`HeaderMark`、`EmphasisMark`、`CodeMark` 等，与 `Emphasis`、`StrongEmphasis`、`FencedCode` 并列。这使得「隐藏 `**`」可以直接对 `EmphasisMark` 的范围做 `Decoration.replace`。【源码+推断】
- `@codemirror/lang-markdown`：`markdown({base, extensions, codeLanguages, addKeymap, ...})`；`commonmarkLanguage` 是基础，`markdownLanguage` 是 `commonmark.configure([GFM, Subscript, Superscript, Emoji, ...])`，GFM 含 `Table`、`TaskList`、`Strikethrough`、`Autolink`。这些就是 corra 里 `mark`、`sup`、`sub` 对应的语法范围，但 `==高亮==` 与 `$数学$` 不在其中，需要自定义。【源码】
- 自定义语法：`MarkdownConfig` 有 `defineNodes`、`parseBlock`、`parseInline`、`remove`、`wrap`。【源码】
- 编辑命令：`insertNewlineContinueMarkup`（回车续列表和引用）、`deleteMarkupBackward`、`markdownKeymap`（`lang-markdown/src/commands.ts`）。这些都是「改文本」的 StateCommand，不是改结构。【源码】

## 6. 在源码上叠加渲染（Live Preview 思路）

- 做法：保持 `doc` 为 Markdown 源码；遍历语法树，对光标不在其中的节点，给标记符加 `Decoration.replace({})`（隐藏），给整块（如表格、公式）加 `Decoration.replace({widget, block: true})`；光标进入节点范围时撤掉这些装饰，让源码显示出来。这一做法由第 4 节的 API 可以直接组合，但「CodeMirror 官方有此示例」我没有核对。【推断】
- 约束：隐藏换行或用块级控件必须来自 StateField 直接提供，不能来自 ViewPlugin 的函数形式（第 4 节）。所以「块级渲染」要放在 StateField 里算，并在 `update` 里用 `tr.changes` 映射或重算。【源码+推断】
- Obsidian：Obsidian 官方开发者文档写道「Obsidian uses CodeMirror 6 (CM6) to power the Markdown editor」，并说想改变 Live Preview 的外观需要构建编辑器扩展。【文档】至于 Live Preview 内部是否正是 `Decoration.replace` 加控件，Obsidian 文档里没有，只来自社区文章的常见说法，本文不当作事实。【第三方，未核对】

## 7. 撤销

`commands/src/history.ts`：【源码】

- 历史是两条分支 `done` 和 `undone`，元素是 `HistEvent`：`changes`（一个 ChangeSet）、`effects`、`mapped`、`startSelection`、`selectionsAfter`。
- 入栈时用 `tr.changes.invert(tr.startState.doc)` 存反向变更，所以每个事件的内存与变更量成正比，不存快照。
- 合并：`newGroupDelay` 默认 500 毫秒，`joinToEvent` 默认「是否相邻」（`isAdjacent` 用 `iterChangedRanges` 判断范围是否接触）。合并时 `event.changes.compose(lastEvent.changes)`；输入法组合期间（`userEvent == "input.type.compose"`）总是并入上一事件。
- 深度：`minDepth` 默认 100。`updateBranch` 在超过 `maxLen + 20` 才裁剪。
- 不进历史的变更：`Transaction.addToHistory` 注解为 false 时，其 `changes.desc` 被 `addMapping` 记入历史，用来改写栈中更早事件的位置，这样撤销时位置仍然对得上。
- `isolateHistory` 注解可强制切分事件，`invertedEffects` facet 让 StateEffect 也能被撤销。
- 撤销本身也是一个 Transaction，带 `fromHistory` 注解，`userEvent` 为 `select.undo` 或 `select.redo`。

## 8. Markdown 往返

- 唯一真相是文本，所以往返是恒等的，保存时输出 `doc.toString()`，不经过序列化。【推断，由模型本身得出】
- 不存在「解析后再写回」的步骤，也就不存在规范化、占位字符或 `stable()` 检查。代价是所有格式命令（加粗、列表、表格）都必须写成「对文本的 ChangeSpec」，不能改结构。`lang-markdown` 的 `insertNewlineContinueMarkup` 就是这类命令的样例。【源码】

## 9. 在 MyGo / Go 里能否实现

先说 MyGo 现有的相关基础（均已读 v0.7.3 源码）：【源码】

- `ui.TextBuffer`：持久化的 rope，单位是 rune，`Snapshot()` 便宜，`Replace(start, end, s)`，方法并发安全（`ui/textbuffer.go`、`ui/text_buffer.go`）。
- `richtext.Text`：一棵 rope 加一棵持久化段落序列 `seq`，`Slice`、`Replace` 都有，偏移也是 rune（`richtext/text.go`、`seq.go`）。
- `richtext.Document`：自带撤销栈，上限 200，1 秒合并，每步保存前后两份 `Text`，二者共享未变部分；`SetText` 会清空历史（`richtext/document.go`）。
- 内部有 `change{version, start, oldEnd, newEnd}` 日志，但未导出，应用层不能用。
- `ui.TextArea` 的方法名里我没有找到按范围样式或装饰的接口，但没有穷尽检查：【未核对】。

逐项判断：

| 项 | 判断 | 改动点 | 风险 |
| --- | --- | --- | --- |
| Text 树状文档 | 已有 | 源码用 `ui.TextBuffer`，无需自写 | 偏移是 rune，与 corra 的 `search.go`（rune）一致 |
| ChangeSet / ChangeDesc / mapPos | 可做，约 500 行 | 新包 `editor/change`：`sections []int`、`inserted []string`，实现 `Apply`、`Invert`、`Compose`、`Map`、`MapPos`。`state/src/change.ts` 共 576 行，可对照移植 | 中。边界语义（`assoc`、`before`）容易出错，需要表驱动测试，并用随机变更验证 `A.Compose(B.Map(A))` 与 `B.Compose(A.Map(B,true))` 一致 |
| State / Transaction | 可做且更简单 | 一个 `type State struct{Doc, Sel, History}` 和纯函数 `Apply(State, Transaction) State`；Facet、Extension、Compartment 不需要 | 低。照搬 Facet 会过度设计 |
| StateField | 不需要 | 用结构体字段，在 `Apply` 里按顺序更新（块列表、装饰、搜索匹配） | 低 |
| 撤销（反向 ChangeSet） | 可做 | 替换 `editor/undo.go`：存 `invert(doc)` 得到的 ChangeSet，合并用 `Compose`，保留 1 秒窗口。内存从「200 份整文快照」变为与变更量成正比 | 中。文稿模式下 `richtext.Document` 自带撤销，需要决定用哪一套（见下） |
| Lezer 增量解析 | 不能直接用 | JS 库。需用 Go 解析器替代；corra 已有 `Split` 的块级切分，可记录每块的源码范围，用 `ChangeDesc.MapPos` 平移未变块，只重解析被触及的块，等价于 Lezer 的块级复用 | 中。跨块语法（链接引用定义、脚注）会让「只重解析一块」不成立；Go 解析器是否输出标记符位置：【未核对】 |
| Decoration.mark | 可做 | 对应 `richtext.Run` 的 `Attributes` 加 `Tag`，已在 `rich_mygo.go` 用 `TagStyle` | 低 |
| Decoration.replace + widget | 部分可做，是核心难点 | 见下一段 | 高 |
| 光标在节点内展开源码 | 可做，块级粒度 | 见下一段 | 中 |
| 搜索 | 简化 | 在源码文本上查找，`editor/search.go` 的 `Find` 即可；文稿模式不再有 `views/rich_mygo.go` 里「附件字符参与匹配」的问题 | 低 |

### Decoration.replace + widget 与 attachment 的对应

- CodeMirror：文档里有 N 个字符的源码，装饰在视图层把它们显示成一个控件；文档本身不变。
- corra 现状：`richtext.Text` 里只有 1 个 U+FFFC，原文挂在 `Attachment.Data` 的 `rawRef`，保存时用私有区字符占位再替换回原文（`mgdoc/codec.go` 的 `Save`）。也就是说「显示物」才是文档内容，「源码」是派生物。
- 两者的差别是真相的方向相反。CodeMirror 模型下，文稿模式需要一个「源码偏移到 richtext 偏移」的映射层，每次变更后重建；richtext 里的编辑要翻译回源码的 ChangeSet。MyGo 的 `richtext.Editor` 直接改自己的 `Text`，没有钩子在中间插入翻译，是否可以拦截：【未核对】。
- 因此不建议在文稿模式里照搬 CodeMirror 的「整文叠加」。可行的折中是块级方案（我的设计，未实现、未验证）：
  1. 真相是源码（`TextBuffer` 加 `ChangeSet` 加反向撤销）。
  2. 用 `Split` 得到块列表，每块有源码范围，用 `MapPos` 随编辑平移。
  3. 光标所在块显示为源码文本（可编辑），其余块用现有 `Codec` 渲染成 `richtext` 段落或附件（只读）。
  4. 光标离开块时，该块重新渲染；编辑只发生在源码块内，所以不需要把 richtext 编辑翻译回来。
  这与 Obsidian 的「光标所在处显示源码」体验相近，但粒度是块而不是行内节点。

### 与现有保存机制的关系

- 采用上述方案后，`Save` 不再需要，`rawBase` 私有区协议与 `stable()` 可以退出主路径（只用于渲染块时决定是否降级为源码块）。
- macOS 走 AppKit `NSTextView` 的路径不受影响，但会出现两套模型：macOS 仍是「富文本为真相」，Windows/Linux 变成「源码为真相」。这会增加测试面。

## 10. 对 corra 的启发

1. 最值得借鉴的是 ChangeSet 与反向撤销。它与 `richtext.Document` 现有的 before/after `Text` 共享结构的做法同类，但 corra 的源码模式撤销（`editor/undo.go`）目前是整份字符串快照，换成 ChangeSet 是局部、低风险的改动，可以先做，不依赖任何渲染改动。
2. `mapPos` 的 `assoc` 概念可以直接解决「文本变动后搜索高亮、折叠、书签位置如何平移」的问题。
3. 块级复用解析结果是 Lezer 的粒度，与 corra 的 `Split` 一致，说明按块缓存是对的；差别只在 Lezer 带位置，corra 要自己补。
4. 装饰层叠加模型依赖视图能「隐藏范围、替换范围」。MyGo 没有现成接口时，不要先动数据模型，先确认视图能力。

## 11. 没能核对的点

- CodeMirror 官方是否有 Live Preview 式示例，以及 Obsidian 的 Live Preview 内部实现（仅第三方说法）。
- MyGo `ui.TextArea` 是否有范围样式接口，`richtext.Editor` 的撤销快捷键能否被应用层拦截。
- Go 解析器（goldmark、`richtext.ParseMarkdown`）是否提供标记符的源码位置。
- 未运行 CodeMirror 或任何移植代码，所有行为都来自源码阅读。
