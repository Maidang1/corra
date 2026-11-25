# Tiptap 的数据建模

## 0. 范围与来源

Tiptap 是 ProseMirror 之上的一层封装。文档树、位置、Step、Mapping、history 都直接用 ProseMirror 的，这些内容见同目录的 `prosemirror.md`，本文不重复。本文只写 Tiptap 增加的东西：声明方式、命令系统、JSON 内容格式、Markdown 包、协同包。

可信度标签：**[源码]** 表示已在下列提交的源码里核对；**[官方]** 表示已在 tiptap.dev 官方文档核对；**[未核对]** 表示没有验证。

读到的版本：`ueberdosis/tiptap` 的 `main`，commit `643792f`（2026-11-22 提交），`@tiptap/core` 与 `@tiptap/markdown` 均为 3.30.3，`@tiptap/y-tiptap` 为 3.0.9。`@tiptap/pm` 3.30.3 依赖 prosemirror-model `^1.25.12`、prosemirror-transform `^1.12.2`、prosemirror-state `^1.4.4`、prosemirror-history `^1.5.1` **[源码]**。

来源 URL：
- https://github.com/ueberdosis/tiptap（packages/core/src：`Extendable.ts`、`Node.ts`、`Mark.ts`、`Extension.ts`、`CommandManager.ts`、`ExtensionManager.ts`、`types.ts`、`helpers/`）
- 同仓库 `packages/extension-heading`、`extension-bold`、`extension-text-align`、`extensions/src/undo-redo`、`extension-collaboration`、`y-tiptap`、`markdown`
- https://tiptap.dev/docs/editor/markdown（官方 Markdown 页）

## 1. 文档数据结构

Tiptap 不定义新的文档模型。`Editor` 持有 ProseMirror 的 `EditorState`，用 `new EditorView(...)` 渲染 **[源码]**（`Editor.ts` 的 `createView`）。节点、标记、属性的运行时形态与 ProseMirror 完全一致。

Tiptap 新增的是**声明方式**：不手写 `SchemaSpec`，而是把每种节点、标记、功能写成一个扩展对象，Tiptap 在启动时合并成 `Schema`。

### 1.1 三种扩展

三者都继承 `Extendable`，用静态方法 `create(config)` 构造，用 `configure(options)` 改选项，用 `extend(config)` 派生 **[源码]**：

- `Extension`：不产生 schema 类型，只加命令、快捷键、插件、全局属性等（`type = 'extension'`）。
- `Node`：产生 `NodeType`。配置里有 `group`、`content`、`marks`、`inline`、`atom`、`selectable`、`draggable`、`code`、`whitespace`、`defining`、`isolating`、`topNode`，字段名与 ProseMirror 的 `NodeSpec` 一一对应。
- `Mark`：产生 `MarkType`，配置里有 `inclusive`、`excludes`、`exitable` 等，前两个与 `MarkSpec` 同名 **[源码]**（`Mark.ts`，其余字段未逐项核对）。

`ExtendableConfig` 里的通用字段 **[源码]**：`name`、`priority`、`addOptions`、`addStorage`、`addGlobalAttributes`、`addCommands`、`addKeyboardShortcuts`、`addInputRules`、`addPasteRules`、`addProseMirrorPlugins`、`addDecorations`、`addExtensions`，以及生命周期回调 `onBeforeCreate`、`onCreate`、`onUpdate`、`onSelectionUpdate`、`onTransaction`、`onFocus`、`onBlur`、`onDestroy`、`dispatchTransaction`。`Node` 另有 `addAttributes`、`parseHTML`、`renderHTML`、`renderText`、`addNodeView`。

### 1.2 一个具体例子

`extension-heading` 的声明（节选）**[源码]**：

```ts
export const Heading = Node.create<HeadingOptions>({
  name: 'heading',
  addOptions() { return { levels: [1,2,3,4,5,6], HTMLAttributes: {} } },
  content: 'inline*',
  group: 'block',
  defining: true,
  addAttributes() { return { level: { default: 1, rendered: false } } },
  parseHTML() { /* h1..h6 -> attrs.level */ },
  renderHTML({ node, HTMLAttributes }) { /* ["h" + level, attrs, 0] */ },
  parseMarkdown(token, helpers) { return helpers.createNode('heading', { level }, helpers.parseInline(token.tokens || [])) },
  renderMarkdown: (node, h) => `${'#'.repeat(level)} ${h.renderChildren(node.content)}`,
  addCommands() { /* setHeading, toggleHeading */ },
})
```

同一个对象里同时放了 schema（`content`、`group`）、属性（`addAttributes`）、DOM 互转（`parseHTML`/`renderHTML`）、Markdown 互转和命令。ProseMirror 里这些分散在 schema、解析器、序列化器和命令模块里。

`Attribute` 类型有 `default`、`validate`、`rendered`、`renderHTML`、`parseHTML`、`keepOnSplit`、`isRequired` **[源码]**（`types.ts`）。

### 1.3 全局属性

`addGlobalAttributes()` 让一个扩展给**别的**节点类型加属性。`extension-text-align` 对一组 `types` 加 `textAlign` 属性，并给出 `parseHTML`/`renderHTML` **[源码]**。ProseMirror 里要改多个 `NodeSpec` 才能做到。

### 1.4 内容格式

`JSONContent`：`{type?, attrs?, content?: JSONContent[], marks?: {type, attrs?}[], text?}` **[源码]**（`types.ts`），与 `Node.toJSON()` 的输出同构。`editor.getJSON()`、`getHTML()`、`getText()` 是读取方式 **[源码]**（`Editor.ts`）。

`getSchemaByResolvedExtensions` 把扩展合并成 `new Schema(spec)`；扩展先按 `priority` 排序（`sortExtensions`）并展开子扩展（`flattenExtensions`）**[源码]**。

## 2. 位置与选区

没有新增。位置是 ProseMirror 整数，选区是 ProseMirror `Selection`。Tiptap 加了两个辅助：

- `NodePos`：包住 `ResolvedPos`，提供类似 DOM 的树导航（`node`、`element` 等）**[源码]**。它的 `element` 返回 `HTMLElement`，所以这个类依赖 DOM。
- `MappablePosition`：只有一个 `position: number` 字段和 `toJSON/fromJSON`，作为可扩展基类，配合 `getUpdatedPosition` 把位置映射过事务 **[源码]**（`helpers/MappablePosition.ts`）。

## 3. 变更表示与命令系统

Tiptap 不新增 Step 类型。它新增的是**命令层**，这是它相对 ProseMirror 的最大差异。

ProseMirror 的命令是 `(state, dispatch?, view?) => boolean`。Tiptap 的命令是两层函数 **[源码]**（`types.ts`）：

```ts
type CommandProps = { editor, tr, commands, can, chain, state, view, dispatch }
type Command = (props: CommandProps) => boolean
type CommandSpec = (...args: any[]) => Command
```

扩展在 `addCommands()` 里返回 `{ name: (...args) => (props) => boolean }`。`ExtensionManager.commands` 把所有扩展的命令合成一个 `RawCommands` 对象 **[源码]**。

`CommandManager` 把它包成三种调用方式 **[源码]**：

- `editor.commands.xxx()`：从 `state.tr` 起一个事务，执行命令，立刻 `view.dispatch(tr)`。
- `editor.chain().a().b().run()`：多个命令共用同一个 `tr`，`run()` 时才派发一次。所以一条链是一个事务，一次撤销。`run()` 返回各命令布尔值是否全为 true。
- `editor.can().xxx()`：传入 `dispatch: undefined`，命令只检查能否执行，不修改文档。

命令里还能调用别的命令：`props.commands`、`props.chain()`。例如 `toggleMark` 先 `isMarkActive`，再调用 `commands.unsetMark` 或 `commands.setMark` **[源码]**（`commands/toggleMark.ts`）。核心带 62 个命令文件（`ls packages/core/src/commands` 去掉测试后的数量）。通用的 `command(fn)` 允许直接拿到 `tr` 写自定义逻辑。

对 Go 实现有价值的是这个模式，不是具体代码：命令 = 「读状态、可选写事务、返回是否适用」，`can` = 干跑，`chain` = 合并事务。

其他辅助：`combineTransactionSteps(oldDoc, transactions)` 把多个事务的步骤拼成一个 `Transform`；`getChangedRanges` 从 `Transform` 导出 `{oldRange, newRange}` 列表 **[源码]**。这两个用于 `onUpdate` 之类的「哪里变了」查询。

## 4. 撤销与协同

### 4.1 撤销

`@tiptap/extensions` 里的 `UndoRedo` 扩展只是包装 `prosemirror-history`：`import { history, redo, undo } from '@tiptap/pm/history'`，`addProseMirrorPlugins` 返回 `[history(this.options)]` **[源码]**。`StarterKit` 默认带它，可用 `undoRedo: false` 关掉 **[源码]**。撤销机制见 `prosemirror.md` 第 4.1 节。

### 4.2 协同

`@tiptap/extension-collaboration` 不用 `prosemirror-collab`。它依赖 `@tiptap/y-tiptap` 和 `yjs ^13`（peer dependency），用 `ySyncPlugin`、`yUndoPlugin`、`undo`、`redo` **[源码]**。

- `y-tiptap` 把 ProseMirror 文档同步到 Yjs 的 `Y.XmlFragment`（`ySyncPlugin(yXmlFragment, ...)`），类型有 `Y.XmlText` 等 **[源码]**。
- 光标位置不用整数，用 Yjs 的相对位置：`absolutePositionToRelativePosition`、`relativePositionToAbsolutePosition` **[源码]**。
- 撤销用 Yjs 的 `UndoManager`（`RestorableUndoManager = UndoManager & ...`），只撤销本地用户的改动 **[源码]**。装了协同扩展再装 `UndoRedo` 会触发警告 `"@tiptap/extension-collaboration" comes with its own history ...` **[源码]**（`collaboration.ts`，警告全文未读完）。
- Yjs 本身是 CRDT：这是常识，本文没有读 Yjs 源码 **[未核对]**。

所以 Tiptap 在协同上给了两条路线：官方扩展走 Yjs CRDT，底层 ProseMirror 也支持 `prosemirror-collab` 的中心权威变基。

## 5. 序列化与 Markdown 往返

Tiptap 3 有官方的 `@tiptap/markdown` 包 **[源码]**，官方文档称它是早期版本，可能变化 **[官方]**。

- 解析器是 `marked`（`^17.0.1`）**[源码]**。`MarkdownManager.parse(markdown)` 先 `lexer.lex` 得到 token，再逐个交给扩展，返回 `{type: 'doc', content}` 这样的 `JSONContent`，不直接建 ProseMirror 节点 **[源码]**。
- 序列化是 `MarkdownManager.serialize(JSONContent)`，对 JSON 递归调用各扩展的 `renderMarkdown(node, helpers, ctx)` **[源码]**。
- 每个扩展用三个字段接入：`markdownTokenName`（对应 marked 的 token 名）、`parseMarkdown(token, helpers)`、`renderMarkdown(node, helpers, ctx)`；非标准语法再加 `markdownTokenizer`，节点缩进用 `markdownOptions.indentsContent`，标记重叠时用 `markdownOptions.htmlReopen` 退回 HTML 写法 **[源码]**（`Extendable.ts`）。
- `parseHelpers` 有 `parseInline`、`applyMark`、`createNode` 等；渲染 helpers 有 `renderChildren`、`renderChild` **[源码]**。
- 使用方式：`contentType: 'markdown'` 传 Markdown 字符串，`editor.getMarkdown()` 取出 **[源码]**（`markdown/src/Extension.ts`）；官方页面写了 `contentType`，没有提到 `getMarkdown` **[官方]**。

会丢或会改的信息：

- 没有 `renderMarkdown` 的节点序列化为空字符串 **[源码]**（`renderNodeToMarkdown` 中 `if (!handler) return ''`）。即静默丢弃，不报错。
- 没有对应处理器且没有子 token 的 marked token 在解析时返回 `null`，同样静默丢弃 **[源码]**（`parseFallbackToken` 的 `default` 分支）。
- 原始 HTML token 走 `parseHTMLToken`，依赖各扩展的 `parseHTML`，而这条路径用 DOM 解析（`generateJSON` 用了 `DOMParser`）**[源码]**。浏览器外需要提供 DOM 实现。
- 官方列出的限制：评论不支持；表格单元格只能有一个子节点 **[官方]**。
- 排版细节（空行数量、列表标记、强调符号）不在 JSON 里，序列化时重新生成。`MarkdownManager` 里有处理空行的代码（`createImplicitEmptyParagraphsFromSpace`、`countParagraphSeparators`），说明它尝试保留空段落，但保真程度没有做往返实验 **[未核对]**。

## 6. 在 MyGo / Go 里能否实现

Tiptap 的价值不在模型，而在扩展 API。逐项判断：

| 概念 | 依赖浏览器吗 | Go 里可行吗 | 说明 |
| --- | --- | --- | --- |
| `Node.create`/`Mark.create` 声明对象 | 否 | 可行 | 在 Go 里是一个注册表：`map[string]BlockSpec`，每个规格带 classify、parse、render、命令。 |
| `priority` 排序、`extend`/`configure` | 否 | 可行但收益小 | corra 的块类型是封闭集合，不需要第三方扩展。 |
| `CommandManager` 的 chain/can | 否 | 可行 | 用「命令接收一个编辑事务对象，返回 bool」实现，`can` 传 `nil` 事务。 |
| `JSONContent` | 否 | 可行 | Go 结构体加 `encoding/json`。 |
| `MarkdownManager` 的 token 注册表 | 否 | 可行 | 概念是「按 token 名找处理器」。 |
| `parseHTML`/`renderHTML`、NodeView、`NodePos.element` | 是 | 不适用 | MyGo 没有 DOM。 |
| `generateJSON`、`getHTML` | 是 | 不适用 | 用了 `DOMParser`/`document.implementation`。 |
| Yjs 协同 | 否（Yjs 本身） | 超出范围 | Go 有第三方 CRDT 实现，但 corra 是单机应用。**[未核对]** |

### 6.1 对 corra 的具体建议

1. **把 `mgdoc` 里分散的「块类型」知识集中成注册表。** 现在块类型的识别在 `classify.go`、绘制在 `blocks.go` 的 `blockParagraph` 的 `switch kind`、还原在 `Save` 里按 `rawRef` 处理。Tiptap 的做法是每个类型一个对象，同时有解析和渲染两个方向。corra 可以定义 `type BlockSpec struct { Kind; Match func(seg string) bool; Draw func(...); Height func(...) }`。新增一种块（比如以后加任务列表块）只加一项，不改多个 `switch`。这是组织代码的改动，不改变行为。
2. **把 Markdown 往返的「失败」改为显式可见。** Tiptap 对无处理器的节点静默输出 `''`，corra 现有机制更安全：无法稳定往返的段落变成 `blockOther` 并原样保存（`stable()`）。保持这一点，不要引入「找不到处理器就丢」的路径。可以借鉴的只有 `markdownOptions.htmlReopen` 的思路：标记重叠写不出时退回另一种写法，corra 里 `mark`/`sup`/`sub` 的交错标签目前由 `codec.go` 的 `stack` 逻辑处理。
3. **命令对象。** `views/rich_mygo.go` 的 `Format(command string)` 是字符串分发。可以改成命令表 `map[string]func(*MygoRich, args) bool`，并支持「干跑」判断工具栏按钮是否可用（对应 `can()`）。工具栏现在的 `toolButton.on` 状态由别处计算。收益是命令和按钮状态来自同一份定义。
4. **链式事务对应 MyGo 的 `Document.Edit`。** `Edit(fn)` 把多个操作合成一个撤销步骤，作用等同 `chain().run()` **[源码，MyGo v0.7.3]**。corra 做复合命令（比如插入脚注同时改行内附件）时应放进同一个 `Edit` 回调。
5. **全局属性的对应物。** MyGo 的 `ParagraphStyle` 是固定结构（`Indent`、`Align`、`Quote`、`CodeBlock`、`ID`），没有像 `addGlobalAttributes` 那样的扩展点。corra 用 `Attributes.Tag` 字符串（`"mark+sup"`、`"lang:go"`）承载额外信息，这是现有的扩展机制，但是字符串约定容易冲突。若要加新的行内属性，需要先约定统一的编码，或向 MyGo 申请通用扩展字段。

### 6.2 会碰到的问题

- 把块类型注册表化之后，`stable()` 的判定仍然必须在 `readText` 里统一执行，不能让某个 spec 自行绕过，否则会破坏「不丢原文」的保证。
- 如果引入 JSON 作为中间格式，要决定附件的 `raw` 是否进 JSON。放进去就是 Tiptap 的路线，文档可以脱离 Markdown 持久化；不放就继续以 Markdown 为唯一真源。corra 目前选后者，这一点不用变。
- 命令表只是整理接口，不解决撤销问题：MyGo 的 `Document` 撤销在 `SetText` 时清空，`app.Hist` 的快照栈只管源码模式，两者的衔接没有逐行核对 **[未核对]**。

## 7. 未核对项

- `Mark.ts` 的完整配置字段清单没有逐项核对。
- Yjs 的内部算法没有读。
- Tiptap 官方文档中命令系统、协同扩展的页面没有读，命令系统的结论来自源码。
- `@tiptap/markdown` 的往返保真度没有做实验。
