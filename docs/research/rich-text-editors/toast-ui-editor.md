# TOAST UI Editor（nhn/tui.editor）数据建模调研

## 0. 版本与来源

- 读的版本：`apps/editor` 3.2.3（`apps/editor/package.json`），仓库 commit `892eb6f7941a77f12bab9bee80d01d048ce42907`（2026-11-15，`git clone --depth 1`）。`libs/toastmark/package.json` 里的版本号是 `0.0.1-alpha.1`，这个版本号与发布版不一致，原因未核对。
- 许可证：MIT（仓库 `LICENSE`，两个 package.json 的 `license` 字段）。
- 来源：
  - https://github.com/nhn/tui.editor （源码，路径见各节）
  - `docs/v3.0-migration-guide.md`、`docs/en/widget.md`、`docs/en/custom-block.md`、`docs/en/plugin.md`（仓库内官方文档）
- 本文没有运行 tui.editor，所有行为来自读源码。
- 可信度标记：【源码】已在源码核对；【文档】已在官方文档核对；【推断】由源码结构推断，未运行验证；【未核对】。

## 1. 先纠正一个常见说法

「Markdown 模式基于 CodeMirror」是 v2 的情况。v3 没有 CodeMirror。

- `docs/v3.0-migration-guide.md` 第 5 行写明 v3 移除了 CodeMirror、squire、to-mark 依赖，改用 ProseMirror 的抽象模型。【文档】
- `apps/editor/package.json` 的依赖只有 `prosemirror-*` 系列。仓库里除文档、示例和 toastmark 的 demo 外没有 CodeMirror 引用。【源码】
- Markdown 模式（`src/markdown/mdEditor.ts`）和 WYSIWYG 模式（`src/wysiwyg/wwEditor.ts`）各自是一个 ProseMirror `EditorView`，共用 `src/base.ts` 里的基类。【源码】

## 2. 数据结构

### 2.1 Markdown 模式：「一行一个段落」的 ProseMirror 文档

- schema：`doc` 内容为 `block+`（`markdown/nodes/doc.ts`），`paragraph` 内容为 `inline*`（`markdown/nodes/paragraph.ts`）。每个源码行对应一个 `paragraph` 节点。【源码】
- `setMarkdown` 把文本按换行拆成行，每行建一个段落，再整体 `replaceWith` 替换文档（`mdEditor.ts` 第 340 行附近）。【源码】
- 另有 `widget` 节点：inline、atom，`content: 'text*'`，属性 `info`。节点里保存的是源码文本，渲染由 `nodeViews.widget` 负责（`widget/widgetNode.ts`）。【源码】
- 语法高亮不是 ProseMirror Decoration，而是在 `appendTransaction` 里给文本加 mark（`markdown/plugins/syntaxHighlight.ts`）。mark 种类有 heading、strong、emph、table、codeBlock 等（`markdown/marks/`）。【源码】

### 2.2 解析器：ToastMark

- 位于 `libs/toastmark`。其 README 称它是从 commonmark.js 扩展而来的解析器，自己实现了 GFM，另有 front matter 和 custom block。【文档+源码】
- 节点带 `sourcepos`，类型是 `[[行, 列], [行, 列]]`，行列都是 1 起算（`types/toastmark.d.ts`：`Pos = [number, number]`，`Sourcepos = [Pos, Pos]`）。【源码】
- `ToastMark` 持有 `lineTexts: string[]` 和 AST 根节点。`editMarkdown(startPos, endPos, newText)` 先改 `lineTexts`，再只重解析受影响的块，返回 `EditResult`（新节点和被移除的行范围），并把后面节点的行号整体平移（`updateNextLineNumbers`）。【源码】

### 2.3 WYSIWYG 模式：语义化的 ProseMirror 文档

- 节点有 heading、bulletList、orderedList、listItem、table 系列、codeBlock、customBlock、html、htmlComment、frontMatter 等；mark 只有 code、emph、link、strike、strong（`wysiwyg/nodes/`、`wysiwyg/marks/`）。【源码】
- 因此 WYSIWYG 的 mark 集合比 Markdown 语法窄。上标、高亮这类 corra 已经支持的语法，在 tui 里要靠插件扩展。【推断】

## 3. 位置与选区

- Markdown 模式的选区是 ProseMirror 的整数位置（含节点开闭标记）。对外 API 用 `[行, 列]`。
- 两种坐标互转在 `markdown/helper/pos.ts`：`getEditorToMdPos(doc, from, to)` 和 `getMdToEditorPos(doc, startPos, endPos)`。后者先用 `getStartPosListPerLine` 算每行起点，再加列号，并用 `getWidgetNodePos` 修正 widget 的长度差。【源码】
- 两个模式之间不存在通用位置映射。切换模式时只同步「当前聚焦的一个节点」。转换器遍历时，遇到 `focusedNode` 就累加已生成节点的大小或已生成文本的行列，记为 `mappedPosWhenConverting`，切换后用它设置选区（`convertors/convertor.ts`、`toWwConvertorState.ts` 第 150 行附近、`toMdConvertorState.ts` 第 350 行附近）。【源码】
- 这个同步依赖 `setFocusedNode` 事件，精度是块级。【推断】

## 4. 变更表示与同步

Markdown 模式的一次编辑流程（`mdEditor.ts` 的 `dispatchTransaction` 和 `updateMarkdown`）：【源码】

1. ProseMirror 产生 Transaction，每个 step 含 `from`、`to`、`slice`。
2. `getEditorToMdPos` 把 `from`、`to` 转成 `[行, 列]`，`getChanged(slice)` 把 slice 还原成纯文本（遇块边界补 `\n`）。
3. 调 `toastMark.editMarkdown(start, end, text)` 做增量解析，得到 `editResult`。
4. `syntaxHighlight` 插件的 `appendTransaction` 用 `editResult` 里的节点重新加 mark，用 `removedNodeRange` 清理旧背景。
5. 预览收到 `updatePreview` 事件，只重绘受影响的块。

要点：纯文本是唯一真相，ToastMark 只是它的增量解析缓存。`ReplaceAroundStep` 被跳过，不触发重解析。【源码】

## 5. 两种模式的转换

`editorCore.ts` 的 `changeMode`、`setMarkdown`、`getMarkdown`：【源码】

- Markdown 到 WYSIWYG：取 `toastMark.getRootNode()`，用 `Convertor.toWysiwygModel(mdNode)` 遍历 AST，生成整份 ProseMirror 文档，再 `wwEditor.setModel` 整体替换。没有增量。
- WYSIWYG 到 Markdown：`Convertor.toMarkdownText(wwNode)` 从 ProseMirror 文档重新序列化为文本，再 `mdEditor.setMarkdown` 整体替换，然后触发 ToastMark 重解析。
- 转换器分两套：`convertors/toWysiwyg/`（按 AST 节点类型注册）和 `convertors/toMarkdown/`（按 ProseMirror 节点类型注册）。插件可以通过 `toHTMLRenderers`、`toMarkdownRenderers` 覆盖（`docs/en/plugin.md`）。【文档】
- 两个模式各有独立的 `history()` 插件（`base.ts` 第 92 行），也就是两份撤销栈。跨模式撤销是否可行：【未核对】。

往返保真：WYSIWYG 到 Markdown 是规范化输出，不保留原写法。

- `toMdConvertors.ts` 里 `bulletList` 的分隔符固定为 `*`，`emph` 固定为 `*`，`strong` 固定为 `**`。所以原文里的 `-` 列表、`_斜体_` 经过 WYSIWYG 往返后会变成 `*` 写法。【源码】
- 文档自己也承认 WYSIWYG 不能渲染一般 HTML 节点，只能靠 `customHTMLRenderer` 为特定标签补支持（`docs/en/custom-block.md`）。【文档】
- 只在 Markdown 模式内编辑时，文本不经过转换，字节不变。【推断】

## 6. 撤销

- 每个模式用 `prosemirror-history`（`base.ts` 引入并在 `defaultPlugins` 注册），快捷键 `Mod-z` 和 `Shift-Mod-z`。【源码】
- 历史记录的是 ProseMirror step 的反向 step，不是文档快照。这是 `prosemirror-history` 的标准做法，tui 没有自定义。【推断，未读 prosemirror-history 源码】
- 模式切换时用 `replaceWith(0, doc.content.size, ...)` 整体替换，这一步本身在目标视图的历史里是一个大 step。是否会被合并或清空：【未核对】。

## 7. 自定义语法与 widget

三种扩展点，由浅到深：

1. 自定义块（custom block）。语法是 `$$名字 ... $$`。ToastMark 在 `commonmark/custom/customBlockStart.ts` 里解析成 `customBlock` 节点，用 `customHTMLRenderer` 里同名函数渲染，WYSIWYG 里点击后用文本方式编辑。内部仍以原文文本为数据。【源码+文档】
2. 内联 widget。`widgetRules: [{ rule: RegExp, toDOM(text) }]`。命中规则的文本被替换成 `widget` 节点，节点里仍存 `$$widgetN 原文$$` 形式的文本（`widget/rules.ts` 的 `createWidgetContent`），导出时用 `unwrapWidgetSyntax` 还原。所以源码不丢。【源码+文档】
3. 解析器插件。`markdownParsers`（`CustomParserMap`，键是 MdNodeType）进入 `ToastMark` 的 `customParser`，在 `blocks.ts` 第 229 到 241 行遍历节点时回调；`table-merged-cell` 插件用它扩展表格语法（`plugins/table-merged-cell/src/markdown/parser.ts`）。【源码】
4. 命令。`markdownCommands`、`wysiwygCommands` 分别注册到两个模式（`docs/en/plugin.md`）。【文档】

## 8. 在 MyGo / Go 里能否实现

逐项判断。corra 现状见 `internal/mgdoc/codec.go`、`views/rich_mygo.go`、`editor/undo.go`。

| 项 | 判断 | 改动点 | 风险 |
| --- | --- | --- | --- |
| 双模式各自一个模型 | 已经是这样 | corra 的源码模式是文本，文稿模式是 `richtext.Text`，两边都整份转换（`Codec.Load`、`Codec.Save`） | tui 的 Markdown 模式本身就是文本加解析缓存，比 corra 的「源码模式」多一层增量解析，文稿模式没有 |
| 增量解析（ToastMark 式） | 可做 | `Split` 已按空行切块。可给每块记录起止行，编辑后只对包含变更行的块重新 `classify` 和 `readText`；后续块只平移行号 | 列表和 fenced code 会跨空行，要像 `Split` 一样特判；引用定义（`[id]: url`）会影响别的块，ToastMark 为此有 `refMap` 与 `refDefCandidateMap` |
| `sourcepos` 式位置 | 可做 | Go 解析器需要给节点附源码范围。goldmark 是否提供到内联级：【未核对】。corra 目前用 `richtext.ParseMarkdown`，它的节点是否带源码位置：【未核对】 | 若解析器不给位置，要自己维护「文稿位置 到 源码位置」的表 |
| 模式切换时同步光标 | 可做，收益有限 | 在 `Codec.Save` 输出时记录每个块的源码起点，切换后按块定位 | 块级精度即可，与 tui 一致 |
| 内联 widget 保存原文 | 与 corra 现有 `rawRef` 同构 | tui 把原文放在节点内容里，corra 放在 `Attachment.Data`。两者都在导出时还原原文 | 无新风险。tui 的做法在文本模式里也能看到原文，corra 的私有区字符方案只在内存里存在 |
| 自定义块 `$$name` | 可做 | 在 `classify.go` 增加块类型，渲染函数挂在 `blockParagraph` 的 `at.Draw` | 与现有 `$$` 数学块冲突，要约定名字前缀 |
| 规范化输出 | 不要照搬 | tui 的 WYSIWYG 往返会改写 `-` 为 `*`。corra 的 `stable()` 检查加 `rawRef` 已经更保守 | 若放宽为 tui 的做法，用户文件会产生无意义的 diff |
| 撤销 | 可做，但 tui 没有答案 | tui 是两份 ProseMirror 历史。corra 只有一份源码快照栈，对「文稿模式」撤销要靠快照 | 见第 9 节 |

## 9. 对 corra 的启发

1. 纯文本为唯一真相的模式，tui 的 Markdown 模式已经验证可行：编辑只改文本，解析器增量更新，高亮是附加层。corra 若想在 Windows/Linux 做「源码内联渲染」，这一路径比「Markdown 转 `richtext.Text` 再转回」稳。
2. tui 的 WYSIWYG 到 Markdown 是有损规范化。corra 现在靠 `stable(t)` 在加载时判定「这一块能否安全进文稿模型」，不安全的块降级为 `rawRef` 附件。这比 tui 保守，应继续保留。
3. tui 的 widget 节点把原文放在节点内容里，导出时 `unwrapWidgetSyntax`。corra 的 `rawRef` 与之等价，但 corra 的私有区字符（`rawBase` 起）是一个隐式协议。可考虑改为在保存时遍历 run 的 `Attachment.Data`，不依赖字符占位（需要先读 `richtext.Text.Markdown()` 的实现确认是否可行：【未核对】）。
4. 位置映射应按「块」做，不要追求字符级。tui 只在切换时映射一个聚焦节点，用户体验可接受。
5. `editor/undo.go` 的整份快照栈（200 条，1 秒合并）对大文档占内存。tui 用 ProseMirror 的 step 反向历史，内存与变更量成正比。补充一点：MyGo 的 `richtext.Document` 自带撤销栈（上限 200，1 秒合并），每步保存前后两份持久化 `Text`，共享未变部分（`mygo@v0.7.3/ui/richtext/document.go`，已读源码）。所以文稿模式的撤销已经有一套，corra 自己的快照栈只服务源码模式。ChangeSet 方案见 `codemirror6.md`。
6. 查找替换：tui 的查找作用于哪个模型【未核对】。corra 的 `editor/search.go` 与 `views/rich_mygo.go` 里 `refreshMatches` 各有一份实现，且后者在 `doc.Text().String()` 上查找，附件字符会参与匹配，这是 corra 自己的问题，不是 tui 的启发。

## 10. 没能核对的点

- ToastMark 版本号与发布版的对应关系。
- `prosemirror-history` 的具体合并与反向 step 行为，以及模式切换时历史是否保留。
- goldmark 和 `richtext.ParseMarkdown` 是否提供源码位置。
- tui 查找功能的模型归属。
- WYSIWYG 里新增 mark（高亮、上标）的实际插件做法，未读 `plugins/` 下的相关实现。
