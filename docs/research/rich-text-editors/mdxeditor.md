# MDXEditor 的数据建模与 Markdown 往返

## 0. 范围、版本、可信度标记

- 读的源码：`mdx-editor/editor` 的 commit `0d0cfba`（2026-11-02，`package.json` 版本字段是 `0.0.0-development`，依赖 `lexical ^0.48.0`、`mdast-util-from-markdown ^2.0.0`、`mdast-util-to-markdown ^2.1.3`）。文档读的是同仓库 `docs/` 目录（`error-handling.md`、`extending-the-editor.md`、`html.md`、`admonitions.md`）。官网 mdxeditor.dev 没有单独抓取，以仓库里的 markdown 为准。
- Go 侧：goldmark v1.8.6（本地模块缓存，带探针实测）、gomarkdown/markdown commit `1671018`。
- 可信度标记：
  - **[源码]**：已在官方源码或文档核对。
  - **[实测]**：我在 `/tmp` 运行验证。
  - **[第三方]**：仅来自第三方文章或搜索结果。
  - **[未核对]**：没有核对，只是推断。

## 1. 整体管线

```
Markdown 文本
  │  mdast-util-from-markdown 的 fromMarkdown(markdown, {extensions, mdastExtensions})
  │  syntaxExtensions = micromark 扩展 (micromark-extension-gfm-strikethrough/-table/-task-list-item、
  │                      -mdxjs、-mdx-jsx、-mdx-md、-frontmatter、-directive、-highlight-mark、自写 comment)
  │  mdastExtensions  = mdast-util-*  的 fromMarkdown 扩展
  ▼
mdast 树（带 position，但后面不使用）
  │  importMdastTreeToLexical(): 按 priority 找第一个 testNode 为真的 MdastImportVisitor
  ▼
Lexical 编辑器状态（lexical 0.48 + @lexical/rich-text、@lexical/list 等）  ← 编辑器的唯一真相
  │  用户操作 = Lexical update；registerUpdateListener 检测 dirty
  ▼
exportLexicalTreeToMdast(): 按 priority 找第一个 testLexicalNode 为真的 LexicalExportVisitor
  ▼
mdast 树（新建，无 position）
  │  mdast-util-to-markdown 的 toMarkdown(tree, {extensions: toMarkdownExtensions, ...toMarkdownOptions})
  ▼
Markdown 文本（再 .trim() 后发布到 markdown$ 单元）
```

- 内部模型是 Lexical，不是 ProseMirror。**[源码]** `src/importMarkdownToLexical.ts` 里 `import { ElementNode, LexicalNode } from 'lexical'`；`core/index.ts` 用 `createExtensionEditor`、`$getRoot()` 构建编辑器。
- 解析不经过 `unified`/`remark`，直接用 `mdast-util-from-markdown` 加 micromark 扩展。**[源码]** `importMarkdownToLexical.ts` 第 4 行与第 241 行。这一点与 Milkdown（`unified().use(remarkParse)`）不同，但底层同属 micromark/mdast 体系。
- 状态管理用自家库 `@mdxeditor/gurx`（`Cell`、`Signal`、`Action`、`realmPlugin`）。**[源码]** `docs/extending-the-editor.md`。
- 每次 Lexical 更新里只要 `dirtyElements`/`dirtyLeaves` 非空，就把整份文档重新导出为 Markdown，写入 `markdown$`，再触发 `onChange`。**[源码]** `core/index.ts` 的 `registerUpdateListener`。没有增量序列化。
- 初始加载时 `initialMarkdownNormalize$` 先置 true，导出后再置 false，`onChange(markdown, initialMarkdownNormalize)` 的第二个参数就是它，用来区分"只是加载时被规范化"和"用户改了内容"。**[源码]** `core/index.ts` 第 233 到 241 行、第 601 行。

## 2. mdast 到 Lexical 的映射

### 2.1 接口（**[源码]**）

```ts
// src/importMarkdownToLexical.ts
interface MdastImportVisitor<UN extends Mdast.Nodes> {
  testNode: ((mdastNode: Mdast.Nodes, descriptors: Descriptors) => boolean) | string  // 字符串时与 node.type 比较
  visitNode(params: { mdastNode: UN; mdastParent: Mdast.Parent | null; lexicalParent: LexicalNode;
                      descriptors: Descriptors; metaData: MetaData; actions: {...} }): void
  priority?: number   // 默认 0，越大越先试
}
// actions: visitChildren, addAndStepInto, addFormatting, removeFormatting, getParentFormatting,
//          addStyle, getParentStyle, nextVisitor

// src/exportMarkdownFromLexical.ts
interface LexicalExportVisitor<LN extends LexicalNode, UN extends Mdast.Nodes> {
  testLexicalNode?(lexicalNode: LexicalNode): lexicalNode is LN
  visitLexicalNode?(params: { lexicalNode: LN; mdastParent: Mdast.Parent; actions: {...} }): void
  shouldJoin?(prevNode: Mdast.RootContent, currentNode: UN): boolean
  join?<T extends Mdast.RootContent>(prevNode: T, currentNode: T): T
  priority?: number
}
// actions: visitChildren, addAndStepInto(type, props?, hasChildren?), appendToParent,
//          registerReferredComponent, visit, nextVisitor
```

- 与 Milkdown 的差别：Milkdown 的 schema 把解析和序列化放在同一个对象里；MDXEditor 把它们拆成两个独立的访问者数组，通过 `addImportVisitor$`、`addExportVisitor$`、`addLexicalNode$` 三个 appender 分别注册。**[源码]** `core/index.ts` 第 342 到 366 行。
- 内联格式的处理不同。mdast 把强调写成包裹节点，Lexical 把粗体、斜体写成文本节点的 format 位。导入时用 `addFormatting(FORMAT)` 往 `WeakMap<Mdast.Parent, number>` 里累加格式；导出时 `LexicalTextVisitor` 把相邻文本节点按格式重新拼成 `strong`/`emphasis`/`delete`/`highlight`，下划线等用 `<u>` 之类的 `mdxJsxTextElement`，由 `shouldJoin`/`join` 合并相邻同类节点。**[源码]** `MdastFormattingVisitor.ts`、`LexicalTextVisitor.ts`。

### 2.2 真实例子（**[源码]**）

| mdast | Lexical | 导入访问者 | 导出访问者 |
|---|---|---|---|
| `heading` | `HeadingNode`（`@lexical/rich-text`） | `testNode:'heading'`；`addAndStepInto($createHeadingNode('h'+depth))` | `$isHeadingNode`；`addAndStepInto('heading', {depth})` |
| `list`、`listItem` | `ListNode`、`ListItemNode`（`@lexical/list`） | 有 `checked` 为 `check`，有 `ordered` 为 `number`，否则 `bullet`；`$createListNode(type, start)` | 固定写 `spread: false`；列表项里的行内内容包进新建的 `paragraph` |
| `code` | `CodeBlockNode`（自定义 DecoratorNode，字段 `code/language/meta`） | `testNode` 里调用 `findCodeBlockDescriptor`，没有匹配的编辑器描述符就返回 false | `addAndStepInto('code', {value, lang, meta}, ...)` |
| `table` | `TableNode`（DecoratorNode，字段 `__mdastNode: Mdast.Table`） | `lexicalParent.append($createTableNode(mdastNode))`，整张表的 mdast 原样存进节点 | `appendToParent(mdastParent, lexicalNode.getMdastNode())`，原样放回 |
| `yaml` | `FrontmatterNode` | `testNode:'yaml'`；`$createFrontmatterNode(mdastNode.value)` | `addAndStepInto('yaml', {value})` |
| `leafDirective`、`containerDirective`、`textDirective` | `DirectiveNode`（字段 `__mdastNode: Directives`） | `testNode` 要求有匹配的 `DirectiveDescriptor` | 原样 `appendToParent(mdastParent, getMdastNode())` |
| `mdxJsxFlowElement`、`mdxJsxTextElement` | `LexicalJsxNode`（字段 `__mdastNode`） | 要求有 `JsxComponentDescriptor`（`name` 匹配或 `'*'`），`priority: -200` | 原样放回，同时登记引用的组件以便重建 import |
| 其余 HTML 标签（`mdxJsx*` 且 `isMdastHTMLNode`） | `GenericHTMLNode`（ElementNode，子内容可编辑） | `MdastHTMLVisitor`，`priority: -100` | `LexicalGenericHTMLVisitor` |

- 观察：表格、指令、JSX 三类"复杂块"都采用同一种做法：Lexical 里放一个 `DecoratorNode`，节点字段直接保存 mdast 节点（`structuredClone` 复制），导出时原样放回。局部编辑发生在 React 组件里，改的是这份 mdast 的拷贝。这等价于 corra 的 `rawRef`，区别是保存的是 AST 子树而不是原文字符串。**[源码]**
- 内核提供的插件按功能拆分：`headingsPlugin`、`listsPlugin`、`quotePlugin`、`tablePlugin`、`linkPlugin`、`imagePlugin`、`codeBlockPlugin`、`frontmatterPlugin`、`directivesPlugin`、`jsxPlugin`、`thematicBreakPlugin` 等，没装对应插件就没有对应访问者。**[源码]** `src/plugins/` 目录。

## 3. 保真问题

### 3.1 官方做的保真措施（**[源码]**）

- **复杂块用 mdast 原样回放**：表格、指令、JSX、frontmatter 的内部结构不经过 Lexical 的富文本模型。
- **JSX 的 import**：`mdxjsEsm` 节点在导入时被忽略（`MdastMdxJsEsmVisitor.visitNode` 是空函数），导出时由用到的组件名重新生成 import 语句（`registerReferredComponent`、`createImportDeclaration`）。所以文档里不对应任何已注册组件的 ESM 语句（例如 `export const x = 1`）会丢失，import 顺序和写法被规范化。这一条是从访问者代码推出的，**[未核对]** 具体行为（没有跑一个 MDX 样本验证）。
- **JSX 种类不一致**：`jsxPlugin` 的 `kindMismatchPolicy`，`'source'`（默认，保留解析结果）、`'normalize'`、`'error'`。**[源码]** `docs/error-handling.md`。
- **内联 HTML**：启用后默认用 `mdast-util-mdx-jsx` 解析 HTML，`<u>`、`<sub>`、`<sup>` 当作格式位；只含 `style` 属性的 `<span>` 当作样式载体；其他标签走 `GenericHTMLNode`，属性保存在节点里。
- **未知文本指令**：`directivesPlugin` 有 `escapeUnknownTextDirectives` 选项，开启后未注册的 `textDirective` 被还原成 `:name` 文本，避免 `12:30` 之类的内容被吞。**[源码]** `MdastDirectiveVisitor.ts`。
- **表格格式**：`gfmTableToMarkdown({tableCellPadding, tablePipeAlign})`，默认两个都为 true，输出对齐的管道表格。

### 3.2 往返后会变的内容

- 默认导出选项只有 `listItemIndent: 'one'`（`DEFAULT_MARKDOWN_OPTIONS`）。**[源码]** `MDXEditor.tsx` 第 108 行。其余沿用 `mdast-util-to-markdown` 默认值：强调 `*`、加粗 `**`、无序列表符号 `*`、围栏用反引号、标题 ATX、分隔线 `***`。这些默认值我在 Milkdown 文档里用同一底层库 `mdast-util-to-markdown@2.2.4` **[实测]** 过（setext 改 ATX、`-`/`+` 改 `*`、`~~~` 改反引号、缩进代码改围栏）。MDXEditor 本身没有保留这些符号的机制，`toMarkdownOptions` 是唯一入口，且是全局的。
- **列表**：导出固定 `spread: false`，松散列表会变紧凑，列表项间的空行被去掉。**[源码]** `LexicalListVisitor.ts`、`LexicalListItemVisitor.ts`。深层嵌套列表有已知的"展平"hack，源码注释写明 "Deeply nested lists are still not supported"。
- **首尾空白**：`markdown$` 在发布前 `.trim()`；初始值默认 `trim: true`。末尾总会追加一个空段落供输入，导入时 `mdastRoot` 末节点不是段落就补一个。**[源码]**
- **HTML 注释**：`commentFromMarkdown({ast:false})` 在 `exit.comment` 里只调用 `this.resume()` 取出文本，不创建 mdast 节点，因此 `<!-- ... -->` 不会进入 Lexical，导出时丢失。**[源码]** `mdastUtilHtmlComment.ts`；"导出时丢失"是由此推出的结论，没有跑样本，**[未核对]**。
- **链接引用与定义**：整个 `src` 下没有处理 `definition`、`linkReference`、`imageReference` 的访问者（grep 无命中）。含 `[text][ref]` 的文档按第 3.3 节的机制整体加载失败。**[源码]** grep 结果；失败行为由导入器推出。
- **脚注、行内数学、块级数学**：仓库里没有 `footnote` 或 `katex`/`math` 支持（grep 无命中），`package.json` 里也没有 `mdast-util-math`、`mdast-util-gfm-footnote`。脚注定义和数学语法不被解析成专门节点，是否作为普通文本保留取决于 micromark 默认行为，**[未核对]**。
- **默认语法开关**：开启 HTML 处理时同时加载 `mdxMd()`，它关闭缩进代码、自动链接和原生 HTML 语法（由 micromark-extension-mdx-md 的 README 说明）。**[源码]** 加载位置 `core/index.ts` 第 1050 行；**[源码]** 关闭内容来自该包 README。因此缩进代码块和 `<https://...>` 自动链接在默认配置下不是 CommonMark 行为。
- 官方的"往返相同"测试被整体跳过：`test/impex.test.tsx` 是 `describe.skip('restore the tests in this file at some point', ...)`。**[源码]** 这说明项目本身没有持续保证 byte 级往返。

### 3.3 不认识的节点怎么处理

- 导入：`visit()` 找不到访问者就 `throw new UnrecognizedMarkdownConstructError`；`fromMarkdown` 本身失败则抛 `MarkdownParseError`。**[源码]**
- 这两类错误在 `tryImportingMarkdown` 里被捕获，写入 `markdownProcessingError$`（含 `error` 和原始 `source`），并调用 `onError`。编辑器保持空白，`DiffSourceWrapper` 显示提示"可在源码模式修复后再切回"。**[源码]** `core/index.ts` 第 700 到 725 行、`diff-source/DiffSourceWrapper.tsx`。
- 导出：找不到 Lexical 访问者就 `throw new Error('no lexical visitor found for ...')`。**[源码]**
- 官方文档的建议：注册一个低优先级的"兜底"访问者处理所有未知节点（`docs/error-handling.md`）。内核没有内置这样的兜底。**[源码]**
- 内核提供的救生口是源码模式（`diff-source` 插件的 `SourceEditor`），用户在源码里改完再切回来。源码视图本身就是原文，所以"无法解析的文档"至少不会丢字。

## 4. 扩展机制：新增语法要改哪几处

文档说明与 frontmatter、高亮的实现一致。**[源码]** `plugins/frontmatter/index.ts`、`core/index.ts` 里的 `highlightMark()`。

1. **micromark 语法扩展**：`addSyntaxExtension$`，类型 `MarkdownParseOptions['syntaxExtensions'][number]`。例：`frontmatter()`、`directive()`、`gfmTable()`、`highlightMark()`。
2. **mdast 构造扩展**：`addMdastExtension$`，类型 `FromMarkdownExtension`。例：`frontmatterFromMarkdown('yaml')`、`directiveFromMarkdown()`。
3. **mdast 序列化扩展**：`addToMarkdownExtension$`。例：`frontmatterToMarkdown('yaml')`、`directiveToMarkdown()`。
4. **Lexical 节点类**：`addLexicalNode$`，继承 `DecoratorNode`（用 React 渲染）或 `ElementNode`。
5. **导入访问者**：`addImportVisitor$`，`MdastImportVisitor`。
6. **导出访问者**：`addExportVisitor$`，`LexicalExportVisitor`。
7. UI：工具栏按钮、`addComposerChild$`、`addNestedEditorChild$`、插入命令的 `Action`。

以 `realmPlugin({init(realm, params){ realm.pubIn({[addImportVisitor$]: ..., [addLexicalNode$]: ..., [addExportVisitor$]: ..., ...}) }})` 包装。**[源码]** `docs/extending-the-editor.md`。

- 不想写 micromark 扩展时，可以复用现成的 mdast 语法：callout 用 `directivesPlugin` 加 `DirectiveDescriptor`（`testNode`、`name`、`attributes`、`hasChildren`、`type`、`Editor`），官方的 `AdmonitionDirectiveDescriptor` 就是这样做的，语法是 `:::note ... :::`。**[源码]** `docs/admonitions.md`、`directives/index.ts`。
- 自定义组件（图表等）用 `jsxPlugin` 加 `JsxComponentDescriptor`（`name`、`kind: 'flow'|'text'`、`source`、`props`、`hasChildren`、`Editor`），写成 `<Chart .../>`。
- 数学公式：内核没有，需要自己接 `mdast-util-math` 与 `micromark-extension-math`（这两个包在 npm 上存在，但我没有核对它们在 MDXEditor 里的整合效果），并写节点与两个访问者。**[未核对]**
- 对于"原文不动"的需求，扩展点是 `DecoratorNode` 里存 mdast 或原文，导出访问者原样放回。

## 5. 位置映射

- mdast 节点由 `fromMarkdown` 带上 `position`，但整个 `src` 下没有读取 `.position` 的代码（grep 只命中无关的 CSS 定位）。**[源码]**
- 保存在 `DecoratorNode` 里的 mdast 节点会带着解析时的 `position`，编辑后过期，导出也没有利用。
- Lexical 的位置是节点 key 加偏移，和源码偏移没有映射。要求"光标所在块对应源码哪几行"无法由内核回答。
- 内核提供的补偿是第三视图：`diff-source` 插件的 `DiffViewer`（对比初始 Markdown 与当前导出）和 `SourceEditor`。让用户看到规范化改了什么，而不是在模型里追踪。**[源码]**
- 撤销：`@lexical/history`，`historyState$` 是整个编辑器共享的 `createEmptyHistoryState()`，嵌套编辑器共用（`historyMode: 'root-shared'`）。记录的是 Lexical 状态，不是源码快照。**[源码]**

## 6. 对 corra 的启发

### 6.1 corra 现状（读自 `internal/mgdoc/{codec,blocks,split,classify}.go`、`views/rich_mygo.go`、`editor/undo.go`）

- `Split` 按空行切块（围栏和 `$$` 内部空行保留）；`classify` 用正则判定受保护块（表格、数学、图表、提示块、脚注、HTML、分隔线、整行图片、引用定义）；其余段用 MyGo 自带的 `richtext.ParseMarkdown`，行内扩展语法先替换成私有区字符。
- 保护块变成一个 `ObjectReplacement` 字符加 `richtext.Attachment`，`Data` 是 `rawRef{raw, block}`；`Save` 把附件换成 `0xF0000` 起的占位符，序列化后再换回原文。`stable()` 用"写出再读回形状相同"做自检，失败就整段降级为 `blockOther`。
- 撤销是源码快照栈（`editor/undo.go`，200 条、1 秒合并）。`MygoRich.Snapshot()` 只在 `doc.Version()` 变化时才 `Save`。
- `go.mod` 里有 `github.com/yuin/goldmark v1.8.6`，渲染侧（`render/`）和 `internal/mdext`（高亮、上下标、内联 HTML 标签、frontmatter 拆分、数学 span）已在用。

### 6.2 goldmark 与 gomarkdown 的能力（**[实测]**，探针在 `/tmp/rte-gm`，加载 GFM 与 Footnote 扩展）

- 叶子块有 `Lines()`（`text.Segment{Start, Stop}` 字节偏移）：`Paragraph`、`Heading`、`FencedCodeBlock`、`CodeBlock`、`HTMLBlock`、`TextBlock`、`TableCell`，v1.8.6 还有 `LinkReferenceDefinition`。行内只有 `ast.Text` 带 `Segment`。
- 容器块（`List`、`ListItem`、`Blockquote`、`Table`、`TableRow`）没有 `Lines()`。
- 范围不含标记：`Heading` 的范围只含标题文字（`# ` 和 setext 下划线在外面），围栏代码只含内容行，开关围栏行在外面。实测 `Setext\n======` 的 `Lines` 只指向 `Setext`。
- 脚注定义被移进文档末尾的 `FootnoteList`，原位置丢失；没有被引用的定义不在 AST 里。
- 没有数学扩展时，`$$ ... $$` 被解析成 `Paragraph`；`internal/mdext` 里已有的行内数学扩展不覆盖块级数学。
- goldmark 核心只带 HTML renderer，没有 Markdown renderer。第三方 `teekennedy/goldmark-markdown` 等只在搜索结果里见到，是否无损 **[未核对]**。
- gomarkdown/markdown 自带 `md/md_renderer.go`，但 `ast` 没有源码位置字段（对 `Offset`、`Position`、`Source` 的 grep 无命中）。**[源码]**

### 6.3 与 MDXEditor 路线对比

| 方面 | MDXEditor | corra 现状 |
|---|---|---|
| 解析 | micromark，AST 完整 | 空行切块加 MyGo `ParseMarkdown` |
| 复杂块 | 节点内存 mdast，导出放回 | 附件内存原文，保存时放回 |
| 未知语法 | 解析整体失败，进入源码模式 | 降级成源码块，其余照常编辑 |
| 写回 | 整份重导出，规范化 | 保护块原样，其余段经 MyGo 序列化 |
| 加载失败粒度 | 整篇文档 | 单个块 |
| 位置 | 丢弃 | 切块时就有原文 |

### 6.4 corra 现有做法更稳的情形

- 任意一小段不被识别，不会让整篇文档打不开。MDXEditor 是整篇失败，编辑器保持空白（`docs/error-handling.md`）。
- 保护块字节级保真。MDXEditor 的表格虽然存 mdast，导出仍要经过 `gfmTableToMarkdown` 重排版。
- 方言、自定义语法无需写 micromark 扩展即可保留。
- 注释、引用定义、脚注这类 MDXEditor 访问者不存在的节点，corra 当前都当成保护块或普通文本保住。

### 6.5 corra 现有做法更差的情形

- 空行切块不懂嵌套。列表项之间有空行或引用里有空行时，一个逻辑块被切碎；`hasUnsafeContent` 只能整段降级，整个列表变成不可编辑的源码块。MDXEditor 的列表、引用由 AST 决定边界。
- 正则 `classify` 对引用里的表格、列表里的围栏、`<details>` 内 Markdown 判断易错。
- 私有区字符要求原文不含这些码点，`tokenize` 要自己重复一遍行内规则，与 CommonMark 分叉。
- 没有结构化事务。MDXEditor 用 Lexical 历史记录；corra 是整份快照，且 `Save` 每次改动都重写所有普通段落。

### 6.6 具体改动点与风险

1. **用 goldmark 做块边界，保留 raw 间隙策略**。顶层 `Document` 子节点算出 `[start,end)`：start 取首个叶子 `Lines().At(0).Start` 回退到行首，围栏、ATX、setext 另补开关或下划线行；end 取下一个顶层块的 start 并去掉尾部空行。块与块之间的间隙原样作为 raw。风险：容器块边界要自己推，需要建语料做断言"各块加间隙拼回等于原文"。
2. **把"AST 子树"和"原文"都作为附件的载荷**：借鉴 MDXEditor 的 `DecoratorNode.__mdastNode`，`rawRef` 增加 `start, end` 偏移和 goldmark 节点种类，渲染（表格、提示块）直接读 AST，不再用 `parseTable` 手写解析。保存仍写原文区间。风险：goldmark 的 `Table`、`Alert` 需要和 `render/` 用同一套扩展配置，否则两边解析结果不同。
3. **逐步替换 `classify` 的正则**：用 `ast.KindTable`、`KindHTMLBlock`、自定义 mdext 节点判定，围栏语言取 `FencedCodeBlock.Language(source)`。脚注因为被移到 `FootnoteList`，需用原文位置而非 AST 顺序。
4. **增加"加载失败退回源码块"的粒度**：MDXEditor 的教训是整体失败太粗。corra 现有的 `stable()` 加 `blockOther` 机制应该保留，并推广到新流程的每个块。
5. **脏块保存**：现在 `Save` 整份重写。有了块区间后，只对被修改的块重新生成，其余块切原文。风险：MyGo `Document` 是否提供"哪个段落改过"的信息 **[未核对]**，需要读 `richtext` 的编辑 API；若没有，可对每块比较加载时的文本形状。
6. **不建议整体换成"AST 出、AST 回"**：goldmark 无官方 Markdown renderer，换过去要么自写，要么接受第三方规范化。MDXEditor 的往返测试被整体跳过，也提示这条路维护成本高。
7. 参考 MDXEditor 的 `DiffViewer`：在 UI 上展示"打开后保存会改动的部分"，对 Markdown 用户比悄悄规范化更可接受。

## 7. 来源

- https://github.com/mdx-editor/editor （commit 0d0cfba；`src/importMarkdownToLexical.ts`、`src/exportMarkdownFromLexical.ts`、`src/plugins/*`、`docs/error-handling.md`、`docs/extending-the-editor.md`、`docs/html.md`、`docs/admonitions.md`、`src/test/impex.test.tsx`）
- https://github.com/micromark/micromark-extension-mdx-md （readme，说明 `mdxMd()` 关闭的语法）
- https://github.com/syntax-tree/mdast-util-from-markdown 、https://github.com/syntax-tree/mdast-util-to-markdown （MDXEditor 依赖的底层库；`mdast-util-to-markdown@2.2.4` 默认行为 **[实测]** 于 `/tmp/rte-node`）
- https://github.com/yuin/goldmark （v1.8.6，本地模块缓存 + 实测）
- https://github.com/gomarkdown/markdown （commit 1671018）
- https://github.com/teekennedy/goldmark-markdown 、https://github.com/RolfLewis/goldmark-down 、https://github.com/mdigger/goldmark-formatter （仅见搜索结果，未核对）
