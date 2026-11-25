# Milkdown 的数据建模与 Markdown 往返

## 0. 范围、版本、可信度标记

- 读的源码：`Milkdown/milkdown` 的 commit `0655111`（2026-10-27，包版本 `@milkdown/*` 7.23.0）。文档读的是 `Milkdown/website` 的 commit `3cd8352`（2026-10-27）。`milkdown.dev` 站点本身在我的环境里无法 WebFetch，所以文档取自这个仓库的 `docs/`。
- remark-stringify 默认值读自 `remarkjs/remark` 主分支的 `packages/remark-stringify/readme.md`。
- 我还在 `/tmp` 里装了 `unified`、`remark-parse@11.0.0`、`remark-stringify@11.0.0`、`remark-gfm`（`mdast-util-to-markdown@2.2.4`）跑了一次往返。注意这是裸 remark，没有加载 Milkdown 的 `remarkHandlers` 和各个 remark 插件，只能说明 remark-stringify 的基线行为。
- 可信度标记：
  - **[源码]**：已在官方源码或文档核对。
  - **[实测]**：我在 `/tmp` 里运行验证。
  - **[第三方]**：仅来自第三方文章或搜索结果。
  - **[未核对]**：没有核对，只是推断。

## 1. 整体管线

```
Markdown 文本
  │  unified().use(remarkParse)            remark-parse 11 → mdast-util-from-markdown → micromark
  │  + remarkPluginsCtx 里的 remark 插件     (remark-gfm, remark-math, 自写 $remark 插件...)
  ▼
mdast 树（MarkdownNode，含 position）
  │  ParserState.next()                    @milkdown/transformer
  │  按 schema 里每个节点/标记的 parseMarkdown.match 找 runner
  ▼
ProseMirror 文档（prosemirror-model 的 Node）   ← 编辑器的唯一真相
  │  用户操作 = ProseMirror Transaction（prosemirror-state / -transform）
  ▼
SerializerState.next()                     @milkdown/transformer
  │  按 toMarkdown.match 找 runner，生成 mdast
  ▼
mdast 树（无 position）
  │  remark.stringify()                    remark-stringify 11 → mdast-util-to-markdown
  ▼
Markdown 文本
```

- 编辑器内部模型是 ProseMirror 文档，不是 Lexical。**[源码]** `packages/transformer/src/parser/state.ts` 里 `ParserState extends Stack<Node, ParserStackElement>`，`Node` 来自 `@milkdown/prose/model`。
- remark 实例在 `packages/core/src/internal-plugin/init.ts` 创建：先 `unified().use(remarkParse).use(remarkStringify)`，等配置完成后用 `remarkStringifyOptionsCtx` 的值重建一次 `unified().use(remarkParse).use(remarkStringify, options)`。**[源码]**
- 官方架构文档把流程写成：Markdown 文本 → remark AST → ProseMirror schema → ProseMirror 文档，序列化反向。**[源码]** `Milkdown/website/docs/guide/architecture-overview.md`。
- 包职责：`@milkdown/transformer` 只含 `ParserState`/`SerializerState` 和类型；`@milkdown/core` 管 remark 实例和 ctx；`@milkdown/preset-commonmark`、`@milkdown/preset-gfm` 提供节点 schema；`@milkdown/plugin-history` 包装 `prosemirror-history`。**[源码]**
- 解析时没有按块增量，整份文档每次 `remark.runSync(remark.parse(markdown))` 后整体建 ProseMirror 文档。序列化也是整份文档走一遍。**[源码]** `ParserState.run`、`SerializerState.run`。

## 2. mdast 到 ProseMirror 的映射

### 2.1 接口

每个节点或标记都是一个 `NodeSchema`/`MarkSchema`，它是 ProseMirror 的 `NodeSpec`/`MarkSpec` 的超集，多两个字段：**[源码]** `packages/transformer/src/utility/types.ts`

```ts
interface NodeSchema extends NodeSpec {
  readonly toMarkdown: NodeSerializerSpec      // { match(node: Node): boolean; runner(state: SerializerState, node: Node): void }
  readonly parseMarkdown: NodeParserSpec       // { match(node: MarkdownNode): boolean; runner(state: ParserState, node: MarkdownNode, proseType: NodeType): void }
  readonly priority?: number
}
```

- 解析方向：`ParserState.#matchTarget` 遍历 `schema.nodes` 和 `schema.marks`，取第一个 `parseMarkdown.match(mdastNode)` 为真的类型，再调它的 `runner`。runner 里用 `state.openNode / closeNode / addNode / addText / openMark / closeMark / next(children)`。**[源码]**
- 序列化方向：`SerializerState` 用 `match` 找 runner，runner 里用 `state.openNode(type, value?, props?) / addNode(type, children?, value?, props?) / withMark(mark, type, value?, props?) / next(content)`，产物是 mdast 节点。**[源码]**
- 解析与序列化是两个独立的函数，没有保证互逆的机制，互逆性由每个 schema 作者自己维护。

### 2.2 真实例子（均 **[源码]**）

| mdast | ProseMirror | parseMarkdown | toMarkdown |
|---|---|---|---|
| `root` | `doc` | `state.injectRoot(node, type)` | `openNode('root')`，最后一个空段落（输入区）不输出 |
| `heading` | `heading`，attrs `{level, id}` | `openNode(type, {level: node.depth}).next(children).closeNode()` | `openNode('heading', undefined, {depth: node.attrs.level})` |
| `list`（无序） | `bullet_list`，attrs `{spread}` | `openNode(type, {spread: node.spread ?? false})` | `openNode('list', undefined, {ordered: false, spread})` |
| `listItem` | `list_item`，attrs `{label, listType, spread}` | 有 `node.label` 视为有序，label 由 `remarkAddOrderInListPlugin` 预先写进 mdast | `openNode('listItem', undefined, {spread})` |
| `code` | `code_block`，attrs `{language}` | `node.lang` 存入 `language`，`node.value` 作为文本子节点 | `addNode('code', undefined, text, {lang})` |
| `html` | `html`（行内 atom，attrs `{value}`） | `addNode(type, {value: node.value})` | `addNode('html', undefined, node.attrs.value)` |
| `strong`、`emphasis` | 标记 | 标记 attrs 里存 `marker`（`*` 或 `_`） | 自定义 `remarkHandlers` 读 `node.marker` |

- 表格在 `@milkdown/preset-gfm` 的 `node/table/schema.ts`：mdast 的 `table`/`tableRow`/`tableCell` 对应 `table`/`table_header_row`/`table_row`/`table_cell`/`table_header`，对齐信息存在 `align` attr。**[源码]**
- 脚注：`footnoteDefinition` 对应 `footnote_definition`（attr `label`，内容 `block+`），`footnoteReference` 对应行内节点。**[源码]** `preset-gfm/src/node/footnote/`。
- remark 阶段的预处理插件（都是 `$remark(...)`）：`remarkAddOrderInListPlugin` 给有序列表项写 `label`，`remarkLineBreak` 把文本里的换行拆成 `break` 节点，`remarkHtmlTransformer` 把容器里的块级 `html` 包进 `paragraph`，`remarkMarker` 记录强调符号，`remarkInlineLinkPlugin` 包装 `remark-inline-links`。**[源码]** `preset-commonmark/src/plugin/`。
- `remark-inline-links@7.0.0` 单独跑 `see [a][ref] and [ref]` 加 `[ref]: http://x.com "T"`，输出 `see [a](http://x.com "T") and [ref](http://x.com "T")`，定义行消失。Milkdown 的 `composed/plugins.ts` 默认加载它，所以引用式链接写回后会变成内联链接。**[实测]** 插件行为，**[源码]** 默认加载。

## 3. 保真问题

### 3.1 官方做的保真措施（**[源码]**）

- **强调符号**：`remarkMarker` 用 `file.value.charAt(node.position.start.offset)` 读出源码里的 `*` 或 `_`，写到 `node.marker`；`strong`/`emphasis` 的 mark schema 把它存成 attr，`remarkHandlers.strong/emphasis` 再原样写回。这是 Milkdown 整个转换链里唯一用到 mdast `position` 的地方（另有 `clone-node.ts` 复制节点时克隆 position）。
- **空行**：段落为空时，序列化成 `<br />`（`EMPTY_LINE_PLACEHOLDER`），解析时 `remarkPreserveEmptyLinePlugin` 把它还原成空段落。其余位置的 `<br>` 当作用户写的 HTML。
- **行内换行**：`remarkLineBreak` 把文本换行转成 `{type:'break', data:{isInline:true}}`，用来区分软换行和硬换行。
- **行首尾空格**：`SerializerState.#moveSpaces` 把标记内容两端的空格挪到标记外面，避免产生 `** a**` 这类无效强调。
- **文本转义**：`remarkHandlers.text` 对只含空白且无 `*_\` 的文本直接返回，其他走 `state.safe(value, {encode: []})`。
- **HTML**：`html` 是行内 atom 节点，值原样存，块级 HTML 被 `remarkHtmlTransformer` 包进段落后也走同一个节点。
- **数学公式**（Crepe 的 latex feature，`packages/crepe/src/feature/latex/`）：`remark-math` 解析出 `math` 和 `inlineMath`；块级 `math` 先被 `remarkMathBlockPlugin` 改成 `code`（`lang: 'LaTeX'`），序列化时 `blockLatexSchema` 再把语言为 latex 的代码块写回 `math`。行内是独立节点 `math_inline`。`remarkInlineMathGuardPlugin` 用 `position.start/end.offset` 在源码里回查，把不符合 pandoc 规则的 `$5 and $10` 还原为文本。
- **自定义指令**：官方示例用 `remark-directive` 的 `leafDirective`（见第 4 节）。`website/docs/plugin/example-iframe-plugin.md`。

### 3.2 往返后会变的内容

下表第 2 列是 **[实测]**（裸 remark-stringify 11 + remark-gfm），第 3 列是 Milkdown 是否有对策 **[源码]**。

| 输入 | 裸 remark-stringify 输出 | Milkdown 对策 |
|---|---|---|
| setext 标题 `Title\n=====` | `# Title` | 无，`setext` 选项默认 false |
| `-` 与 `+` 无序列表 | 都改成 `*`，相邻两个列表第二个改用 `-` | 无，列表符号不存 attr，由全局 `bullet` 选项决定（官方 FAQ 示例 `bullet: '*'`） |
| `_em_`、`__strong__` | `*em*`、`**strong**` | 有，`remarkMarker` 保留 |
| `~~~js` 围栏 | 改成反引号围栏 | 无 |
| 缩进代码块 | 改成无语言的围栏代码块 | 无 |
| `1) one` | `1. one` | 无 |
| 表格对齐空格 | 重排成等宽 | 无（`:-:` 对齐信息保留） |
| 行尾两空格硬换行 | `\` + 换行 | 无 |
| `\$5` | `$5`（转义被丢，加了 `remark-math` 后行为不同，未测） | 部分 |
| 松散列表 `spread` | 由节点的 `spread` 决定 | `spread` 存成 attr，因此保留 |
| 列表项缩进、空行数量 | 按 `listItemIndent`（默认 `one`）重排，多余空行合并 | 无 |

- 默认选项见 remark-stringify 文档：`bullet: '*'`、`emphasis: '*'`、`strong: '*'`、`fence: '`'`、`fences: true`、`rule: '*'`、`setext: false`、`listItemIndent: 'one'`、`incrementListMarker: true`。**[源码]** `remark/packages/remark-stringify/readme.md`。
- Milkdown 暴露的配置入口是 `remarkStringifyOptionsCtx`，初始值 `{handlers: remarkHandlers, encode: []}`。FAQ 示例设置 `bullet`、`fences`、`incrementListMarker`。**[源码]** `packages/core/src/internal-plugin/atoms.ts`、`website/docs/guide/faq.md`。
- front matter：`Milkdown/milkdown` 仓库和 `website/docs` 里没有 `frontmatter`/`yaml` 相关代码或文档（grep 无命中），即内核不支持。裸 remark 对 `---
title: a
tags: [x]
---` 的往返结果是 `***

title: a
tags: \[x]
----------`，即被当作分隔线加 setext 标题并被转义改写。**[源码]** 无支持，**[实测]** 改写结果。
- 脚注：`preset-gfm` 有节点支持。裸 remark + gfm 实测脚注定义原样回写。**[实测]**
- 链接引用定义 `[ref]: url`：裸 remark 保留；Milkdown 默认加载 `remarkInlineLinkPlugin`，会被内联化（见 2.2 节末尾）。

### 3.3 不认识的节点

- 解析时找不到 `parseMarkdown.match` 为真的类型，`#matchTarget` 直接 `throw parserMatchError(node)`。`getDoc` 里没有 try/catch，错误会冒到 `editor.create()`。**[源码]** `parser/state.ts`、`core/src/internal-plugin/editor-state.ts`。
- 序列化时找不到匹配则 `throw serializerMatchError`。**[源码]**
- 没有 passthrough 或 raw 兜底节点。所有"保真"都靠为每种语法写一对 schema。Milkdown 里唯一接近 raw 的是 `html` 节点（只覆盖 mdast 的 `html`）。**[源码]**
- 触发条件：装了会产生新 mdast 类型的 remark 插件（如 `remark-directive`、`remark-math`），却没有为这个类型写 schema，就会抛错。没装语法插件时，对应语法只是被当成普通文本，不抛错也不保真。**[源码]** 抛错路径；后半句是 micromark 的常规行为，**[未核对]**。

## 4. 扩展机制：新增一种语法要改哪几处

官方示例 iframe 插件列出五件事。**[源码]** `website/docs/plugin/example-iframe-plugin.md`

1. **remark 插件**：用 `$remark('remarkDirective', () => directive)` 把解析插件接进 `remarkPluginsCtx`（`packages/utils/src/composable/composed/$remark.ts`）。签名 `$remark<Id, Options>(id, remark: (ctx) => RemarkPluginRaw<Options>, initialOptions?)`。
2. **schema**：`$node(id, (ctx) => NodeSchema)` 或 `$nodeSchema`（带 ctx 的 `$nodeSchema` 在 preset 里更常用），定义 `group/attrs/parseDOM/toDOM`。
3. **parseMarkdown**：`match: node => node.type === 'leafDirective' && node.name === 'iframe'`，runner 里 `state.addNode(type, {src})`。
4. **toMarkdown**：`runner: (state, node) => state.addNode('leafDirective', undefined, undefined, {name:'iframe', attributes:{src}})`。
5. **input rule**：`$inputRule(...)` 让用户键入时触发。

另外按需补：`$command`（命令）、`$useKeymap`（快捷键）、`$view`/`$prose`（NodeView、ProseMirror 插件）、`blockContainerTypes`（新增流式容器类型时注册，否则空行占位符无法往返）。**[源码]** `block-container-types.ts` 的注释写明了这个要求。

- 数学公式：remark 插件 `remark-math`，加一个 `math_inline` 节点（`$nodeSchema`），块级复用 `code_block` 并扩展 `toMarkdown`（`codeBlockSchema.extendSchema`）。
- 图表（mermaid 等）：Milkdown 没有专用节点。Crepe 的 CodeMirror 代码块通过 `codeBlockConfig.renderPreview(language, content, applyPreview)` 按语言渲染预览，节点仍是 `code_block`，所以往返天然保真。**[源码]** `crepe/src/feature/latex/index.ts` 里对 `renderPreview` 的覆盖。
- callout：没有内置。可以用 `remark-directive` 的 `containerDirective` 加自定义节点。**[未核对]** 官方没有 callout 示例，这只是由 iframe 示例类推。

## 5. 位置映射

- remark 解析出的 mdast 节点带 `position`（行、列、offset）。但 `ParserState` 的 runner 签名里只传 `node`，官方 runner 不把 `position` 写进 ProseMirror 节点 attrs。**[源码]** `transformer/src` 中对 `position` 的 grep 无命中。
- ProseMirror 里的位置是文档内整数偏移，与源码偏移没有对应表。编辑后无法由模型推回源码的某一段。
- 唯一用法是 `remarkMarker` 和 `remarkInlineMathGuardPlugin` 在 remark 阶段（解析时）回查源码字符，用完即丢。**[源码]**
- 序列化产物是全新字符串，与旧源码没有 diff 对应。想要"只改动被编辑的块"需要自己做。**[源码]** 无此机制。
- 撤销用 `prosemirror-history`（`@milkdown/plugin-history` 包装），记录的是 Transaction 的 step，不是源码快照。**[源码]**

## 6. 对 corra 的启发：能否用 Go AST 访问者替换"空行切块 + 私有区还原"

### 6.1 corra 现状（读自 `internal/mgdoc`、`views/rich_mygo.go`、`editor/undo.go`）

- `Split` 按空行把源码切成段，围栏代码和 `$$` 块保留内部空行。`classify` 用正则判断每段是否受保护块（表格、数学、图表、提示块、脚注、HTML、分隔线、整行图片、引用定义）。其余段交给 MyGo 自带的 `richtext.ParseMarkdown`，行内的 `==`、`^`、`~`、`$..$`、`[^1]`、尖括号标签先用私有区字符（`U+E001` 到 `U+E006` 表示高亮、上标、下标的开关，`0xF0000` 起表示原文占位）替换，读完再换回。
- `Save` 反向：附件的 `rawRef.raw` 先换成私有区占位，让 `richtext.Text.Markdown()` 序列化，最后把占位替换回原文。`stable()` 用"写出再读回形状不变"检验每段是否安全，不安全就整段降级为 `blockOther`。
- 撤销：`editor/undo.go` 是源码字符串快照栈（200 条、1 秒合并）。`Snapshot()` 只在 `doc.Version()` 变化时才重新 `Save`，所以没改动的文档不会被重写。
- `go.mod` 已有 `github.com/yuin/goldmark v1.8.6` 和 `goldmark-emoji`；`internal/mdext` 已有 goldmark 扩展（高亮、上下标、行内 HTML 标签、frontmatter 拆分、`[TOC]`、数学 span）。渲染侧已经在用 goldmark，编辑侧还没有。

### 6.2 goldmark 能提供什么（**[实测]**，v1.8.6，加载 GFM 和 Footnote 扩展，探针在 `/tmp/rte-gm`）

- **源码位置**：叶子块（`Paragraph`、`Heading`、`FencedCodeBlock`、`CodeBlock`、`HTMLBlock`、`TextBlock`、`LinkReferenceDefinition`、`TableCell`）有 `Lines()`，是 `text.Segment{Start, Stop}` 字节偏移。行内只有 `ast.Text` 带 `Segment`。
- **缺口**：
  - 容器块（`List`、`ListItem`、`Blockquote`、`Table`、`TableRow`）没有 `Lines()`，整块范围要自己由子节点推。
  - `Heading` 的范围只含标题文字，`# ` 前缀和 setext 下划线不在内；围栏代码的范围只含内容行，开关围栏行不在内。
  - `Emphasis`、`Link`、`CodeSpan` 本身没有范围，只有子 `Text` 有。
  - 脚注定义被移到文档末尾的 `FootnoteList`，位置顺序丢失，未被引用的定义不在 AST 里。
  - `$$` 块在没有数学扩展时被解析成 `Paragraph`。
  - v1.8.6 有 `LinkReferenceDefinition` 节点，带范围，不会像旧版那样消失。
- **往返**：goldmark 核心只有 HTML renderer，没有 Markdown renderer。第三方 `teekennedy/goldmark-markdown`、`RolfLewis/goldmark-down`、`mdigger/goldmark-formatter` 存在，但是否无损往返 **[未核对]**（仅看到搜索结果）。
- `gomarkdown/markdown`（commit `1671018`）自带 `md/md_renderer.go`，能输出 Markdown，但 `ast` 里没有源码位置字段（对 `Offset`、`Position`、`Source` 的 grep 无命中）。**[源码]**

### 6.3 与 Milkdown 路线的对比

| 方面 | Milkdown | corra 现状 |
|---|---|---|
| 解析器 | remark（带 position） | 按空行切块 + MyGo 自带 `ParseMarkdown` |
| 不认识的语法 | 抛错，没有 raw 兜底 | 降级为受保护块，原文字节不变 |
| 写回 | 整份重新生成，格式被规范化 | 受保护块原样，普通段落经 MyGo 序列化 |
| 未改动内容 | 也被规范化（`*`、围栏、标题风格） | 文档版本未变则不写；改了则普通段落仍被规范化 |
| 位置 | 解析后丢弃 | 切块时天然知道每段原文 |

### 6.4 corra 现有做法更稳的情形

- 遇到 AST 没有节点的语法（自定义 `==`、`^`、数学、`[TOC]`、各种方言），原文直接作为 raw 保留，不会抛错。Milkdown 在这种情形下抛 `parserMatchError`。
- 受保护块的往返是字节级的，而 remark-stringify 是规范化的（上表）。
- `stable()` 是按段落的自检，给了"写不回就不编辑"的安全阀，Milkdown 没有。
- 不依赖第三方 Markdown renderer，也不需要写一整套 `parse/toMarkdown` 对。

### 6.5 corra 现有做法更差的情形

- 空行切块不理解结构：列表项之间、引用内部有空行时，一个逻辑块被切成多段；`hasUnsafeContent` 只能整段降级，整个列表或引用变成"源码块"，不可原地编辑。
- `classify` 靠正则，嵌套情形（引用里的表格、列表里的围栏、`<details>` 内的 Markdown）判断会漏或错。
- 私有区字符方案要求原文不含这些码点；`tokenize` 要自己重写一遍行内规则（反引号、链接目标、裸 URL、`<`），与 CommonMark 规则分叉。
- 没有节点级位置，无法做"光标所在块对应源码的哪一段""只重写被改动的块"。每次改动都整份 `Save`，普通段落被 MyGo 规范化。
- 撤销是整份快照，没有结构化事务。

### 6.6 具体改动点与风险

1. **只用 goldmark 做切块，不换模型**。用 `parser.Parse` 取顶层 `Document` 的子节点，每个顶层块算出 `[start, end)`：start 为首个叶子 `Lines().At(0).Start` 回退到行首，对围栏、ATX 标题、setext 标题再向前或向后补开关行；end 取下一个顶层块的 start，去掉尾部空行。块之间的间隙原样作为 raw。这样能替换 `Split` 与 `classify` 的大部分正则，并且列表、引用不会再被空行切碎。风险：容器块边界要自己推，必须用语料做回归，断言"所有块范围拼接加间隙等于原文"。
2. **每个顶层块带源码区间**：`rawRef` 增加 `start,end`，`Codec` 保存原文与区间表。保存时对未改动的块直接切原文，只对被改动的块重新生成。风险：MyGo 的 `Document` 是否能告诉你哪一段被改过（`Version()` 是整体计数）**[未核对]**，需要读 `richtext` 的编辑 API。
3. **把 `classify` 的受保护类型改成按 goldmark 节点类型判定**（`ast.KindTable`、`KindHTMLBlock`、`KindFootnote...`、自定义 `mdext` 节点），围栏语言从 `FencedCodeBlock.Language()` 取。风险：脚注定义在 AST 里被移到末尾，不能靠 AST 顺序定位，仍需用原文位置。
4. **行内私有区替换**：可以改为遍历块内行内 AST，用 `Text.Segment` 回查原文，把 `mdext` 的 `Mark/Superscript/Subscript`、数学 span 映射为带 `Tag` 的 run，逐步淘汰 `tokenizeLine`。风险：行内节点没有整体范围，要用首尾子 `Text` 推，加上分隔符长度。
5. **不建议整体换成 Milkdown 式"AST 出、AST 回"全量模型**：goldmark 没有官方 Markdown renderer，换过去要么自写 renderer，要么引入第三方并接受规范化。保留 raw 间隙策略是 corra 相对 Milkdown 的优势。
6. **撤销**：若有了块区间，可以把快照栈改为"块级差异"以省内存，但 200 条整份快照对普通文档足够，优先级低。

## 7. 来源

- https://github.com/Milkdown/milkdown （commit 0655111，`packages/transformer`、`core`、`plugins/preset-commonmark`、`preset-gfm`、`crepe/src/feature/latex`）
- https://github.com/Milkdown/website （commit 3cd8352，`docs/guide/architecture-overview.md`、`faq.md`、`docs/plugin/example-iframe-plugin.md`）
- https://github.com/remarkjs/remark/blob/main/packages/remark-stringify/readme.md
- https://github.com/remarkjs/remark-frontmatter/blob/main/readme.md （第三方说明，未在 Milkdown 内核对）
- https://github.com/yuin/goldmark （v1.8.6，本地模块缓存 + 实测）
- https://github.com/gomarkdown/markdown （commit 1671018）
- https://github.com/teekennedy/goldmark-markdown 、https://github.com/RolfLewis/goldmark-down 、https://github.com/mdigger/goldmark-formatter （仅见搜索结果，未核对）
