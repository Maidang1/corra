# Lexical 的数据建模

## 0. 版本与来源

- 源码：`git clone --depth 1 https://github.com/facebook/lexical`，commit `f8344ad59742d43ae9532a6ea08eaf1843bc3560`，`packages/lexical/package.json` 版本 0.53.0，CHANGELOG 标注 v0.53.0 于 2026-10-28 发布。
- 官方概念文档读的是同一 commit 里的 `packages/lexical-website/docs/concepts/`，对应网站 https://lexical.dev/docs/concepts/ 。没有单独抓取网站页面。
- 主要源码路径（均在上述 commit）：`packages/lexical/src/`（`LexicalNode`、`LexicalEditorState`、`LexicalUpdates`、`LexicalUtils`、`LexicalReconciler`、`LexicalSelection`、`LexicalNodeState`、`LexicalConstants`、`nodes/`）、`packages/lexical-{history,yjs,markdown,table}/src/`、`packages/lexical-playground/src/plugins/MarkdownTransformers/index.ts`。
- 可信度标记：**[源码]** 已在源码核对；**[文档]** 已在官方文档源文件核对；**[未核对]** 没有读到或没有验证。本文没有使用第三方文章。

## 1. 数据结构

### 1.1 EditorState [源码]

`LexicalEditorState.ts`，类 `EditorState`，主要字段：

- `_nodeMap: NodeMap`，即 `Map<NodeKey, LexicalNode>`，`NodeKey` 是 `string`（`LexicalNode.ts`）。
- `_selection: null | BaseSelection`。
- `_readOnly`、`_flushSync`、`_parsed`、`_slotsUsed`。
- `_movedNodes`、`_splitNodes`、`_textEdits`：给 Yjs 绑定用的旁路记录，以前一个 nodeMap 为弱引用键。

整个文档是一张扁平的 key 到节点的表。树结构由节点自身的指针字段表达，不是嵌套对象。根节点的 key 固定为 `'root'`。

### 1.2 LexicalNode 公共字段 [源码]

`LexicalNode` 有：`__type`（只读）、`__key`、`__parent`、`__prev`、`__next`、`__state?: NodeState`。父子和兄弟关系都用 key 引用。

### 1.3 子类与其字段 [源码]

| 类 | 额外字段 |
| --- | --- |
| `ElementNode` | `__first`、`__last`、`__size`、`__format`、`__style`、`__indent`、`__dir`、`__textFormat`、`__textStyle`、`__slotHost`、`__slots` |
| `RootNode`（继承 ElementNode） | `__cachedText` |
| `ParagraphNode`（继承 ElementNode） | 无新增结构字段 |
| `TextNode` | `__text: string`、`__format: number`、`__style: string`、`__mode: 0\|1\|2\|3`、`__detail: number` |
| `TabNode`、`CodeHighlightNode` | 继承 `TextNode` |
| `LineBreakNode` | 无 |
| `DecoratorNode<T>` | `__slotHost`、`__slots`。内容由 `decorate(editor, config)` 返回 |
| 表格 | `TableNode`、`TableRowNode`、`TableCellNode`（`__colSpan`、`__rowSpan`、`__headerState`、`__backgroundColor`） |

`ElementNode` 只存第一个和最后一个子节点的 key，子节点靠 `__next`/`__prev` 串成链表。块级元素没有「行」的概念，一个段落内的换行是 `LineBreakNode` 子节点。

`DecoratorNode` 默认是行内节点，`isInline()` 返回 `false` 才是块级 [文档 `decorators.mdx`，源码 `isInline`/`isIsolated`/`isKeyboardSelectable`]。

### 1.4 节点 key [源码]

`generateRandomKey()` 在 `LexicalUtils.ts`，实现是 `'' + keyCounter++`，一个进程内递增的计数器，不是随机串。key 在节点构造时分配并写入 `_nodeMap`。key 不进入 JSON（见第 6 节），所以重新加载后 key 会变。文档规定业务代码只在构造函数和 `clone` 里碰 `__key` [文档 `key-management.md`]。

### 1.5 NodeState [源码，文档 `node-state.md`]

`createState(key, {parse, ...})` 定义带类型的键值，`$getState` / `$setState` 读写。`NodeState` 类持有 `knownState`、`unknownState`，以及一份在同类节点间共享的配置。克隆节点时新版本先引用旧版本的 NodeState，第一次 `$setState` 才复制（写时复制）。未知的键会原样保留并在导出时写回，用于新旧版本代码并存。序列化时位于 JSON 的 `$` 键下（`NODE_STATE_KEY = '$'`），也可配置为平铺存储。

### 1.6 TextNode 的 format 与 mode [源码，`LexicalConstants.ts`]

`format` 是位掩码：

| 常量 | 值 |
| --- | --- |
| `IS_BOLD` | 1 |
| `IS_ITALIC` | 2 |
| `IS_STRIKETHROUGH` | 4 |
| `IS_UNDERLINE` | 8 |
| `IS_CODE` | 16 |
| `IS_SUBSCRIPT` | 32 |
| `IS_SUPERSCRIPT` | 64 |
| `IS_HIGHLIGHT` | 128 |
| `IS_LOWERCASE` / `IS_UPPERCASE` / `IS_CAPITALIZE` | 256 / 512 / 1024 |

`mode` 有三种：`'normal'`、`'token'`、`'segmented'`（`TextModeType`）。`token` 表示整体增删、不能在内部编辑；`segmented` 按词整体删除。`detail` 也是位掩码：`IS_DIRECTIONLESS = 1`、`IS_UNMERGEABLE = 2`。注意类型 `TextDetailType` 里写的是 `'unmergable'`（少一个 e），与常量名不一致，这是源码现状。

相邻且 format、style、mode 相同的文本节点会被规范化合并（`LexicalNormalization.ts`，合并条件 [未核对]）。

## 2. 双缓冲与更新

### 2.1 流程 [文档 `updates.mdx`，源码 `LexicalUpdates.ts`]

- `editor._editorState` 是已提交的当前状态，`editor._pendingEditorState` 是待提交状态。
- `editor.update(fn, options)` 进入 `$beginUpdate`：如果没有 pending 状态，就调用 `cloneEditorState(current)`，实现是 `new EditorState(cloneMap(current._nodeMap), ...)`，即只复制 key 到节点的 Map，节点对象本身共享。
- 同一个 tick 内的多次 update 共用一个 pending 状态，提交通过 microtask 调度。
- `options.discrete: true` 让 pending 状态的 `_flushSync = true`，提交不等 microtask，同步完成。
- `editor.read(fn)` 默认模式是 `'force-commit'`，先把 pending 更新提交再读；还有 `'pending'`、`'latest'` 模式，以及 `editor.read(editorState, fn)` 读任意一个历史状态。读取环境下 `isReadOnlyMode = true`，写入会抛错。
- 提交后，pending 变成新的 current。文档称已提交状态是冻结的。源码里 `LexicalReconciler.ts` 在协调时对节点调用 `Object.freeze`（第 928、2062 行）[源码]。

### 2.2 getLatest 与 getWritable [源码，`LexicalNode.ts`]

- 节点对象本身是不可变的快照。持有旧引用的代码读到的是旧版本。
- `getLatest()` 用 `__key` 去当前活动状态的 `_nodeMap` 里取最新版本。找不到会抛错，提示引用可能来自别的 read 或 update。
- `getWritable()` 只能在 update 中调用。它先查 `editor._cloneNotNeeded`（本次更新内已经克隆过的节点），命中直接返回；否则从 pending nodeMap 取最新节点，`$cloneWithProperties` 克隆，写回 nodeMap，加入 `_cloneNotNeeded`，并调用 `internalMarkNodeAsDirty`。
- 结果是写时复制：一次更新只复制被改的节点，未改节点在新旧两个 EditorState 里是同一个对象。

### 2.3 dirty 标记 [源码]

`internalMarkNodeAsDirty`（`LexicalUtils.ts`）：

- 叶子节点（非 ElementNode）key 进 `editor._dirtyLeaves: Set<NodeKey>`。
- 元素节点 key 进 `editor._dirtyElements: Map<NodeKey, boolean>`，值为 `true` 表示这个节点本身被改，`false` 表示只因子孙被改而被连带标记（`internalMarkParentElementsAsDirty` 沿 `__parent` 一路向上标 `false`，遇到已标记的就停）。
- `editor._dirtyType` 取 `NO_DIRTY_NODES = 0`、`HAS_DIRTY_NODES = 1`、`FULL_RECONCILE = 2`。

### 2.4 协调 [源码，`LexicalReconciler.ts`]

`$reconcileRoot(prev, next, editor, dirtyType, dirtyElements, dirtyLeaves)` 从 `'root'` 开始递归 `$reconcileNode`。对每个 key：如果新旧 nodeMap 里是同一个对象，且不在 dirty 集合里（或不是 `FULL_RECONCILE`），就复用已有 DOM，只更新文本缓存，不再下探。dirty 的节点才会创建或更新 DOM，子节点用 `$reconcileChildren` 对比。DecoratorNode 在协调时调用 `decorate()`，结果放进 `editor._pendingDecorators`，由框架层（如 React）渲染。

提交前还有两步：节点 transform 反复执行到没有新 dirty 为止（`$applyAllTransforms`），然后 `$garbageCollectDetachedNodes` 把已脱离树的节点从 nodeMap 删掉（`LexicalGC.ts`）。

## 3. 选区 [源码，`LexicalSelection.ts`]

- `RangeSelection`：`anchor: PointType`、`focus: PointType`、`format: number`、`style: string`、`dirty`，以及缓存 `_cachedNodes`。
- `Point`：`key: NodeKey`、`offset: number`、`type: 'text' | 'element'`。`type` 为 `text` 时 `offset` 是该文本节点内的字符偏移，为 `element` 时是子节点下标。
- `NodeSelection`：`_nodes: Set<NodeKey>`，用于选中一个或多个 DecoratorNode 之类的整节点。
- `TableSelection`（`lexical-table/src/LexicalTableSelection.ts`）：`tableKey`、`anchor`、`focus`，同样实现 `BaseSelection`。0.53.0 全仓库搜不到 `GridSelection`，旧名称已不存在。
- 选区属于 `EditorState._selection`，随状态一起冻结和切换，所以历史记录里每一项都带着当时的选区。选区只存 key 加偏移，不存对象引用。

## 4. 历史与协同

### 4.1 lexical-history [源码]

- `HistoryState = {current, undoStack, redoStack}`，每项是 `HistoryStateEntry = {editor, editorState}`，直接存整个 `EditorState` 对象。
- 因为未改节点是共享的，一个历史条目只多占一个 nodeMap 和被改节点，成本和被改节点数成正比（结构共享）。这一推断来自 2.1、2.2 的实现，history 包本身没有声明内存复杂度 [推断]。
- 撤销调用 `editor.setEditorState(entry.editorState, {tag: HISTORIC_TAG})`，直接换回整份状态，随后协调只涉及与当前状态不同的节点。
- 合并策略由 `createMergeActionGetter` 决定，返回 `HISTORY_MERGE`、`HISTORY_PUSH`、`DISCARD_HISTORY_CANDIDATE`。依据是：更新标签（`history-merge`、`history-push`、`historic`、组合输入标签）、`getChangeType` 判出的输入类型（逐字符插入、向前删除、向后删除、其他）、以及距上次变更是否小于 `delay`。判断用到 `dirtyLeaves` 与 `dirtyElements`，例如只有一个 dirty 叶子且 `exportJSON()` 前后相同则合并。
- 条数上限：有 `cap` 参数，超过时 `undoStack.splice(0, ...)` 截掉最旧的。

### 4.2 lexical-yjs [源码]

- 每个 Lexical 节点对应一个 `Collab*` 对象：`CollabElementNode`（持有 `_xmlText: XmlText`、`_children`）、`CollabTextNode`（`_map: YMap`、`_text`）、`CollabLineBreakNode`（`_map`）、`CollabDecoratorNode`（`_xmlElem: XmlElement`）。绑定里有 `collabNodeMap` 保存 `NodeKey` 到 Collab 对象的映射。
- V1 的做法：元素节点是一个 `XmlText`，子节点作为嵌入对象（`insertEmbed`）放进去，文本内容插入父元素的 `XmlText`，所以一个段落的文字是 CRDT 序列，文字并发编辑可以合并。节点属性（`__format` 等）存在 YMap 里。同步时排除 `__key`、`__parent`、`__next`、`__prev`、`__state`、`__slotHost`、`__slots`（`baseExcludedProperties`），即所有 key 和链表指针都不进 CRDT，结构由 Yjs 的序列位置决定。
- 另有实验性的 V2（`SyncV2.ts`，导出名带 `__EXPERIMENTAL`）：相邻文本节点合并成一个 `XmlText`，用 `applyDelta` 把 format 之类属性作为区间属性；非文本节点各对应一个 `XmlElement`，属性存在元素的 attribute 里，只同步与默认值不同的值。
- 本地更新到 Yjs 走 `syncLexicalUpdateToYjs`，入参正好是 `prevEditorState`、`currEditorState`、`dirtyElements`、`dirtyLeaves`，所以增量同步依赖 dirty 标记。反方向 `syncYjsChangesToLexical` 把远端变更套到 Lexical 状态上，带 `COLLABORATION_TAG`，避免回声。
- 协同下的撤销用 `Y.UndoManager`（`createUndoManager`），不用 lexical-history [源码 `index.ts`]。

## 5. Markdown（@lexical/markdown）

### 5.1 Transformer 类型 [源码，`MarkdownTransformers.ts`]

- `ElementTransformer`：`dependencies`、`export(node, traverseChildren, ...)`、`regExp`、`replace(parentNode, children, match, isImport)`、`type: 'element'`，可选 `triggerOnEnter`。用于标题、引用、列表。
- `MultilineElementTransformer`：用于跨多行的块，代码块 `CODE` 属于这一类，有 `regExpStart`、`regExpEnd`、`handleImportAfterStartMatch`。
- `TextFormatTransformer`：`format: TextFormatType[]`、`tag: string`、`intraword?`，如 `**`、`*`、`` ` ``、`~~`、`==`。
- `TextMatchTransformer`：`regExp`、`importRegExp`、`replace`、`export`、`trigger`，用于链接这类需要整段匹配的行内结构。

默认集合 `TRANSFORMERS` 由 `ELEMENT_TRANSFORMERS`（标题、引用、无序列表、有序列表）、`MULTILINE_ELEMENT_TRANSFORMERS`（代码）、`TEXT_FORMAT_TRANSFORMERS`（行内代码、粗斜体、粗体、高亮、斜体、删除线）、`TEXT_MATCH_TRANSFORMERS`（只有 `LINK`）拼成。

### 5.2 导入与导出 [源码]

- 导入 `$convertFromMarkdownString(markdown, transformers, node?, ...)` 先 `root.clear()`，再按行匹配块级和多行转换器，文本内再依次应用行内转换器。
- 导出 `$convertToMarkdownString` 遍历 root 的子节点，对每个节点依次尝试各块转换器的 `export`，返回非空即采用；没有转换器接管的 ElementNode 只导出子节点内容，DecoratorNode 导出 `getTextContent()`（默认为空串）。文本格式导出时只用单一格式的转换器，所以粗斜体会导出为嵌套的 `**` 与 `*`。

### 5.3 会丢什么 [源码 + 推断]

- 导入时空行语义被改写：「Markdown 没有空段落」，相邻非空行默认按 CommonMark 规则合并，除非传 `shouldPreserveNewLines`（`MarkdownImport.ts` 与 `MarkdownTransformers.ts` 中的行预处理）。原文的空行数量、行尾空格、缩进风格、列表标记（`-`/`*`/`+`）在往返后不保证一致 [推断，由导出器只用固定标记得出，未逐项跑测试]。
- 包本身不含表格、公式、图片、水平线。这些在 playground 里另写：`TABLE`（元素转换器）、`HR`、`IMAGE`（text-match）、`BLOCK_EQUATION`（`$$` 多行）、`EQUATION`（行内） [源码，`lexical-playground/.../MarkdownTransformers/index.ts`]。表格导出时每个单元格递归调用 `$convertToMarkdownString`，换行替换成 `\n` 字面量，只在存在表头行时输出分隔行，对齐信息（`:---:`）没有写出 [源码，`TABLE.export`]。
- 代码块：`CODE` 导出时若内容含围栏就加长围栏，语言存在 `language`，info 串里语言之后的部分存在 NodeState（`codeMetaState`、`codeFenceState`）[源码]。
- 没有对应转换器的节点，往返后内容丢失或退化为纯文本。Lexical 的 Markdown 是「状态到 Markdown 的有损投影」，不是源码级无损格式 [推断]。

## 6. 序列化 [源码]

- `EditorState.toJSON()` 返回 `SerializedEditorState = { root: SerializedRootNode }`。选区不在其中。
- 每个节点是 `SerializedLexicalNode`：`type: string`、`version: number`，可选 `$`（NodeState）和 `$slots`。`SerializedElementNode` 加 `children`、`direction`、`format`、`indent`、`textFormat?`、`textStyle?`。`SerializedTextNode` 加 `detail`、`format`、`mode`、`style`、`text`。
- `version` 在 0.53.0 里已标 `@deprecated`：注释写明「没有任何东西读它，解析时完全忽略」，`exportJSON()` 仍写 `1` 只是为了让旧版本可读。所以 Lexical 没有逐节点的版本迁移机制；兼容靠 `importJSON`/`updateFromJSON` 自己容错。
- 0.53.0 新增紧凑形式：`toJSON({compact: true})` 返回 `CompactSerializedEditorState`，省略解析时能自行恢复的默认值。旧版 Lexical 读不了紧凑形式。
- 解析 `editor.parseEditorState(json)` 对每个节点按 `type` 查已注册类，查不到直接抛错（`parseEditorState: type "%s" + not found`）。依次调用 `importJSON`、再对子节点递归、`append`。key 在这个过程里重新分配。

## 7. 在 MyGo / Go 里能否实现

### 7.1 corra 现状（已读代码）

- `internal/mgdoc/split.go` 的 `Split` 按空行切块，围栏代码和 `$$` 数学块保留内部空行。
- `internal/mgdoc/codec.go` 的 `Load` 对每块 `classify`；普通文本走 `readText`（先 `tokenize` 再 `richtext.ParseMarkdown`，用 `stable` 比较往返形状确认无损），受保护块走 `blockParagraph`，生成一个只含 `richtext.ObjectReplacement` 的段落，`Attributes.Attachment.Data = rawRef{raw, block}`。
- `Save` 遍历所有段落，把 `rawRef` 换成私有区字符（`rawBase = rune(0xF0000)`，`classify.go:17`）占位后交给 `richtext` 生成 Markdown，再把占位替换回原文。
- `views/rich_mygo.go` 的 `Snapshot` 用 `doc.Version()` 判断是否变化，变化就对整个文档调用 `codec.Save`，是整篇 O(n)。
- `editor/undo.go` 的 `Undo` 存 `snapshot{src string, caret int}`，上限 200，1 秒合并。`app/state.go` 里 `Undo()` 调 `SetSourceQuiet`，它整篇 `reparse()`。`MarkSourceDirty` 不进 `Hist`。
- MyGo v0.7.3（`~/go/pkg/mod/github.com/egoist/mygo@v0.7.3/ui/richtext`）已有关键能力 [源码]：
  - `Text` 是不可变值，注释写明「方法返回新文本，与原文本共享未改动部分，保存一个版本只花一个指针」，内部是 rope 加段落序列。
  - `ParagraphStyle.ID string` 是应用给段落起的名字，编辑时保留、撤销时恢复，`Document.IdentifyParagraphs(newID)` 给新段落发 ID；Markdown、HTML、剪贴板导出时不带它。
  - `Document` 有自己的撤销栈 `undo, redo []undoStep`，每步存 `before, after Text` 和变更区间，上限 200、1 秒合并（`document.go:383-395`），与 corra 的 `Undo` 参数相同。
- Windows/Linux 富文本模式下 Ctrl+Z 走哪套撤销 [未核对]：`views/` 与 `app/` 里只有 `app/state.go` 调用 `Hist.Undo`，没有对 `Document.Undo` 的调用，推测由 MyGo 编辑器内部处理，没有跟到事件入口。

### 7.2 逐项判断

| Lexical 机制 | 在 Go/MyGo 能否实现 | 说明 |
| --- | --- | --- |
| 不可变快照 | 能，且 MyGo 已提供 | `richtext.Text` 已是持久化值。corra 自己的块层只需要把块列表做成不可变切片，改一块就复制切片再换掉一项 |
| key 到节点的映射 | 能 | `map[BlockID]*Block` 加有序 `[]BlockID` |
| `getWritable` 写时复制 | 能，但没必要照搬 | 块是整体替换的粒度，不需要「节点内字段写时复制」。用值类型 `Block` 加 copy-on-write 切片即可 |
| dirty 集合 | 能 | `map[BlockID]struct{}`，在块替换时记录。Lexical 的 `false`（连带 dirty）对 corra 的两层结构（文档 → 块）不需要，因为没有深层嵌套 |
| 全局 pending 状态加 `update(fn)` 事务 | 能，但会改变调用方式 | Go 没有 JS 的「活动状态」全局变量，需要显式传 `*Tx`。收益是把一次用户操作内的多次改动合成一个历史条目 |
| 协调（reconciler） | 不需要 | MyGo 即时模式，每帧重画，无 DOM。dirty 集合的价值是缓存失效 |
| 位掩码 format | 不需要 | `richtext.Attributes` 已有样式字段和 `Tag`，corra 用 `Tag` 编码 `mark`/`sup`/`sub` |
| DecoratorNode | 能，对应已有 attachment | 见 7.4 |
| JSON 序列化 | 能，暂不需要 | 落盘格式就是 Markdown 源码 |
| Yjs 绑定 | 未核对 Go 生态 | corra 不做协同，不在范围内 |

### 7.3 能否把撤销改成按块 key 的增量快照

可以，思路是只把 `Undo` 的快照单位从「整份源码」换成「块列表」：

1. 块模型：`type Block struct { ID uint64; Src string; Kind blockKind }`，文档是 `[]Block`。`Split` 的输出直接成为初始块列表，`ID` 用递增计数器，和 Lexical 的 `keyCounter++` 一致。
2. 快照：`snapshot{ blocks []Block, caret Pos, sel }`。`Block` 是值类型、`Src` 是不可变 string，复制 `[]Block` 只复制切片头加每项几个字，不复制正文字符串，实际共享了未改块的内容。这和 Lexical 的 `_nodeMap` 复制相当：O(块数)，不是 O(字节数)。
3. 增量：一步历史可以只记被改块的旧版本和旧顺序，比整份切片更省，但块数少于几千时收益不大，不必先做。
4. dirty：`Snapshot()` 现在是整篇 `Save`。有了块 ID 和「哪些块变过」，可以只对 dirty 块重新 `Save`，其余块沿用缓存的 `Src`，再拼接。这是整个改动里收益最明确的一项（击键延迟随文档变长而增长的问题）。
5. 合并规则：保留 1 秒窗口，再参考 `lexical-history` 的做法，只在「同一块内连续插入或删除一个字符」时合并，其他操作（粘贴、格式化、块切换）强制开新条目。

具体改动点：

- `editor/undo.go`：`snapshot.src string` 改为 `blocks []Block`，`PushAt` 增加变更类型参数，实现类型感知的合并。
- `internal/mgdoc/split.go`、`codec.go`：`Split` 返回带 `Kind` 和稳定 `ID` 的块，`Load` 保留 ID 到段落的映射。可以利用 `richtext.ParagraphStyle.ID`，让 MyGo 在编辑、撤销时替我们维护段落到块 ID 的对应 [源码，见 7.1]。
- `codec.Save`：接受 dirty 集合，对未变块直接输出缓存的 `Src`。
- `app/state.go`：`Undo()` 不再 `SetSourceQuiet` 整篇 `reparse`，改为把被改块的段落替换掉。
- `views/rich_mygo.go`：`Snapshot` 用段落 ID 和 `doc.Version()` 找出变化的块。

风险：

- **与 MyGo 自带撤销重复。** `richtext.Document` 已有带结构共享的撤销栈，上限和合并时间与 corra 相同。如果富文本模式其实一直在用它，再建一套块级撤销会产生两套互相不知道的历史。动手前必须先弄清 7.1 中未核对的那一点。更合理的方向可能是富文本模式复用 `Document.Undo`，`editor/undo.go` 只服务源码模式。
- **ID 稳定性。** 一个块被 `Split` 重新切出来时（例如用户在源码模式下编辑后切回富文本）ID 会变，历史里的 ID 失效。需要规定：跨模式切换时清空历史（现状 `Refresh` 是追加一条快照，不清空）或按位置对齐。
- **块边界会因编辑改变。** 用户在文本块里敲两个换行，一块变两块；删除空行，两块并一块。Lexical 用 `splitText`/`mergeWithSibling` 加 key 规则处理这个问题，`ParagraphStyle.ID` 的规则（拆分时第一半保留 ID，合并时保留第一个）与之相近，可以直接用。
- **`stable` 检查。** `readText` 靠往返形状比较决定一块能否作为文字编辑，不能则降级为 `blockOther` 受保护块。块一旦变化，需要重新判定它是否仍然 stable，类型可能在可编辑和受保护之间切换，历史条目必须带上 `Kind`。
- **私有区字符占位。** 用户文档若本身含 0xF0000 起的字符，可能与占位冲突。`expand` 只做了数量边界判断，是否另有保护没有查 [未核对]。块级快照不改变这点。

### 7.4 DecoratorNode 与 attachment / 受保护块

对应关系：

| Lexical | corra / MyGo |
| --- | --- |
| `DecoratorNode`，块级 `isInline() === false` | `blockParagraph` 生成的单字符段落，内容是 `ObjectReplacement`，`Attachment.Data = rawRef{block: true}` |
| `DecoratorNode`，行内 | `inlineRaw` 生成的行内 attachment（`rawRef{block: false}`） |
| 节点上的数据（如 `EquationNode` 的公式文本） | `rawRef.raw`，整块原文，不做结构化 |
| `decorate()` 返回的视图 | `Attachment.Draw func(p *ui.Painter, r ui.Rect)` |
| `getTextContent()` 供纯文本与 Markdown 导出 | `Attachment.Name`，Markdown 导出靠 `Save` 把 `rawRef.raw` 写回 |
| `NodeSelection`、`isKeyboardSelectable()` | attachment 是一个字符，按字符参与选择，键盘选中细节 [未核对] |

差异：

- Lexical 的 DecoratorNode 有显式的类型和数据（`EquationNode.getEquation()`），Markdown 转换器按类型生成文本。corra 的 attachment 只存原文字符串，类型靠 `classify` 每次从原文重新推断。这使得无损往返很容易（原文一字不改），代价是无法在富文本模式里编辑块内容（`editHint` 就是「在源码模式中编辑」）。
- 如果以后要在富文本模式里编辑表格或公式，Lexical 的做法值得参考：节点持有结构化数据（表格是 `TableNode`→`TableRowNode`→`TableCellNode` 的树，单元格是可含段落的元素节点），导出由转换器负责。这会让 Markdown 不再无损，需要接受 5.3 里说的那类丢失，或者像现在一样保留 `raw` 并在结构未被改动时优先输出 `raw`。后一种做法 Lexical 本身没有，它是 corra 的优势，应该保留。
- 块 ID 可以直接放进 `rawRef`，让撤销时 attachment 的重绘缓存（例如数学公式位图 `mathItem.bmp`、图表 `startDiagram`）按 ID 复用，避免撤销后重新渲染。现在 `Load` 开头执行 `c.math = nil`，撤销一次就重新渲染全部公式。这是个明确的收益点。

## 8. 给 corra 的结论

1. Lexical 的核心可借用点是三项：稳定的节点 key，按 key 记录的 dirty 集合，历史条目共享未改节点。这三项在 Go 里都容易实现，不需要照搬其 `update` 事务和 reconciler。
2. 最值得做的是让 `Snapshot`、`Save` 按块增量，并让数学、图表的渲染缓存按块 ID 存活。撤销改造排在弄清 `Document.Undo` 与 `editor.Undo` 的分工之后。
3. 不要采纳 Lexical 的 Markdown 导入导出模型。它是有损投影，而 corra 已有的「原文一字不改地写回」是更强的保证。

## 9. 未核对项汇总

- lexical.dev 网页没有抓取，文档部分读的是仓库内的源文件。
- `@lexical/markdown` 往返丢失的具体项目（空行数、列表标记、缩进）是由代码结构推断，没有运行测试验证。
- `LexicalNormalization.ts` 中文本节点合并的完整条件没有细读。
- 富文本模式撤销实际走哪套（见 7.1）。
- macOS NSTextView 路径的撤销和 `rawRef` 处理没有读。MyGo attachment 的键盘选中行为没有核对。
- Go 生态中是否有可用的 Yjs 兼容实现没有调查。
