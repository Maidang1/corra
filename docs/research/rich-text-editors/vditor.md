# Vditor 与 Lute 数据建模调研

## 0. 版本与来源

- Vditor：`package.json` 版本 4.0.1，commit `0732967e1bd16de7716399b6926bf75ad9bfc5d8`（2026-11-09），`git clone --depth 1`。许可证 MIT（仓库 `LICENSE`，`package.json` 的 `license` 字段）。
- Lute：`go.mod` 声明 `module github.com/88250/lute`、`go 1.26`，commit `a654af9ca125ecc701734d8b320900c26ab34a6e`（2026-11-23）。许可证见第 8 节。
- 来源：
  - https://github.com/Vanessa219/vditor
  - https://github.com/88250/lute （`README_en_US.md`、`LICENSE`、`ast/node.go`、`vditor_ir.go`、`render/vditor_ir_renderer.go`）
- 可信度标记：【源码】已读源码核对；【文档】已读仓库内文档核对；【推断】由源码推断，未运行；【未核对】。
- 说明：本次没有在本机运行 Lute（`go get` 与 `go run` 被环境的权限分类器拒绝，没有绕过），所以所有「Lute 行为」都来自读源码，没有实测输出。

## 1. 三种模式与技术底座

- `types/index.d.ts` 第 761 行：`mode?: "wysiwyg" | "sv" | "ir"`。【源码】
- 所见即所得（wysiwyg）和即时渲染（ir）都是 `<pre ... contenteditable="true">`（`src/ts/wysiwyg/index.ts` 第 50 行，`src/ts/ir/index.ts` 第 61 行）。分屏预览（sv）是一个 `<textarea class="vditor-sv">`，另配预览区（`src/ts/sv/index.ts` 第 13 行）。【源码】
- Vditor 不基于 ProseMirror。`package.json` 的运行时依赖只有 `diff-match-patch`。【源码】
- Markdown 解析、渲染、DOM 与 Markdown 互转都由 Lute 完成。Vditor 在运行时加载 `${cdn}/dist/js/lute/lute.min.js`（`src/index.ts` 第 553 行），Lute 的 JS 版由 GopherJS 编译（`lute/javascript/build.sh`：`gopherjs build --tags javascript`）。仓库的 `dist/` 里没有这个文件，是从 CDN 或 npm 包取。【源码】
- 所以结论成立：Vditor 直接操作 contenteditable DOM，用 Go 写的 Lute 解析。

## 2. 数据结构

### 2.1 Vditor 一侧：DOM 就是模型

- 在 wysiwyg 和 ir 模式下没有独立的文档对象。浏览器里的 DOM 是唯一的状态，取 Markdown 时调 `lute.VditorDOM2Md(element.innerHTML)` 或 `lute.VditorIRDOM2Md(...)`（`src/ts/markdown/getMarkdown.ts`）。【源码】
- sv 模式的状态是 textarea 的 `value`，即纯文本。【源码】
- 切换模式时都经过 Markdown 字符串：先 `getMarkdown`，再对目标模式调 `Md2VditorIRDOM`、`Md2VditorDOM` 或写入 textarea（`src/ts/toolbar/EditMode.ts` 第 27、59 行）。【源码】

### 2.2 即时渲染（IR）的 DOM 表示

IR 的做法是把 Markdown 标记符保留为 DOM 里的真实文本节点。以 `**加粗**` 为例，`render/vditor_ir_renderer.go` 的 `renderStrongA6kOpenMarker` 输出：

- `<span class="vditor-ir__marker vditor-ir__marker--bi">**</span><strong data-newline="1">`
- 外层节点由 `renderSpanNode` 加 `data-type="strong"` 等属性（em、s、mark、sup、sub、a 同理）。块级节点带 `data-block="0"`。【源码】

光标移入时，`src/ts/ir/expandMarker.ts` 给节点加 `vditor-ir__node--expand` 类，标记符才显示出来，移出后隐藏。也就是说，源码一直在 DOM 里，是否可见由 CSS 控制。【源码】

### 2.3 Lute 一侧：AST

- 节点结构 `ast.Node`（`ast/node.go`）是一个胖结构体，不用接口：`Type`、`Parent/Previous/Next/FirstChild/LastChild`、`Tokens []byte`、`Data string`，再加按类型使用的字段（`ListData`、`HeadingLevel`、`CodeBlockInfo`、`TableAligns`、`CalloutType` 等）。注释明确说这样设计是为了转换类型方便和性能。【源码】
- 语法标记符本身是节点：`NodeStrongA6kOpenMarker`（`**`）、`NodeStrongU8eOpenMarker`（`__`）、`NodeEmA6kOpenMarker`、`NodeHeadingC8hMarker`（`#`）、`NodeBlockquoteMarker`、`NodeCodeBlockFenceOpenMarker` 等。写回 Markdown 时按这些节点输出，所以 `**` 与 `__` 的区别在 AST 里保留。【源码】
- 内置节点类型包含 corra 目前靠私有区字符和 `tag` 处理的语法：`NodeMark`、`NodeSup`、`NodeSub`、`NodeInlineMath`、`NodeMathBlock`、`NodeFootnotesDef`、`NodeCallout`、`NodeCustomBlock`、`NodeTable`、`NodeYamlFrontMatter`。这些语法要开对应解析选项（Vditor 的 `setLute` 里有 `SetMark`、`SetSup`、`SetSub`、`SetCallout`、`SetFootnotes`、`SetInlineMath`）。【源码】
- 节点里夹带了不少思源笔记（SiYuan）专用字段：块引用 ID、属性视图、闪卡挖空、`KramdownIAL`。【源码】

## 3. 位置与选区

- Vditor 用浏览器 `Range`。块内位置用文本偏移：`getSelectPosition(blockElement, editorElement, range)` 返回 `{start, end}`，值是从块首到光标的 `toString().length`（`src/ts/util/selection.ts` 第 105 行）。【源码】
- 重绘后恢复光标靠一个 `<wbr>` 元素：输入时把 `<wbr>` 插入 DOM，重绘后用 `setRangeByWbr` 找到它并还原 Range。【源码】
- 光标穿过 Lute 的方式值得注意：`SpinVditorIRDOM` 先把 `<wbr>` 换成字符 `‸`（`editor.Caret`），转成 Markdown、再解析、再渲染，最后把 `‸` 换回 `<wbr>`（`vditor_ir.go` 第 28 到 37 行）。也就是把光标当成一个普通字符塞进文本，让它随往返一起搬运。【源码】
- Lute 的 AST 没有源码位置。在 `ast` 和 `parse` 目录里搜索 `sourcepos`、`StartLine`、`LineNum` 均无结果；`Node` 里只有 `CodeBlockFenceOffset`、`MathBlockDollarOffset`、`MarkerOffset` 这类局部偏移。【源码】
- 因此不能用「AST 节点到源码区间」做选区映射，只能靠上面的光标字符法，或者自己按块切分后记录偏移。

## 4. 变更表示

没有变更表示。Vditor 不记录 delta。每次输入后的处理是「重画一块」（`src/ts/ir/input.ts`）：【源码】

1. 找到光标所在的块级元素。列表取最顶层 `ul/ol`，引用取 `blockquote`，脚注取整个脚注块，链接引用定义取整个文档。
2. 取该块的 `outerHTML`，列表还会并入前后相邻的列表。回车时并入上一段。
3. 调 `lute.SpinVditorIRDOM(html)`：DOM 转 Markdown（`vditorIRDOM2Md`，先 `adjustVditorDOM` 整理浏览器生成的结构，再 `genASTByVditorIRDOM` 建 AST，再用 `FormatRenderer` 输出文本），再解析，再用 `VditorIRRenderer` 渲染回 DOM。
4. 用返回的 HTML 替换原块，恢复光标。

这套机制叫「自旋」（Spin）。它的代价是每次输入都做一次整块的往返；好处是 DOM 永远由 Markdown 重新生成，不会漂移。

## 5. 撤销

`src/ts/undo/index.ts`：【源码】

- 每个模式（wysiwyg、ir、sv）各一套栈，`stackSize = 50`。
- 记录对象是编辑区的 HTML 字符串（sv 是 textarea 文本），内含 `<wbr>` 光标标记；用 `diff-match-patch` 的 `diff_main` 和 `patch_make` 生成补丁，栈里存补丁列表，不存整份快照。
- 撤销是对 `lastText` 应用补丁，再整体写回 `innerHTML` 并重新渲染预览块，最后用 `<wbr>` 恢复光标。重做是反转补丁。
- 写入栈由 `afterRenderEvent` 里的 `setTimeout` 触发（`src/ts/wysiwyg/afterRenderEvent.ts` 第 15 行），代码注释称已防抖。【源码，注释为作者说法】
- 切换模式时在新模式的栈上追加记录（`EditMode.ts` 里 `enableAddUndoStack: true`），旧模式的栈保留。【源码】

对比 corra：同样是「文本级」历史，但 Vditor 存的是补丁而不是快照，且记录的是 HTML 而非 Markdown 源码。

## 6. Markdown 往返保真

- 在 sv 模式下文本不经转换，字节不变。【推断】
- 在 ir 和 wysiwyg 模式下，每次编辑都经过 `FormatRenderer`。AST 里保留了强调符号种类（`*` 对 `_`）和列表 `BulletChar`，所以这些写法能保住；但空行、缩进、表格对齐、中西文空格等会被规范化（`README_en_US.md` 的 Format 一节明说格式化目的是统一风格）。【源码+文档】
- Lute 有对相邻强调的专门处理：`formatVditorTree` 在重新解析后发现 HTML 结果与原树不一致时，改用 `<em>`、`<strong>` 标签输出以保边界（`vditor_emphasis.go`）。这说明作者碰到过往返不稳定的情况并打了补丁。【源码】
- `RenderOptions` 默认 `AutoSpace: false`（`render/renderer.go` 第 140 行），Vditor 的 `setLute` 按选项开启；开启后会改写用户文本。【源码】
- 未实测任意 Markdown 文件的往返差异：【未核对】。

## 7. Lute 的 AST 与 IR 设计小结

- 解析：`parse.Parse(name, []byte, *parse.Options) *parse.Tree`，`Tree.Root` 是 `NodeDocument`。遍历用 `ast.Walk(root, func(n, entering) WalkStatus)`。【源码】
- 渲染：同一棵树可以用不同的 `Renderer`：`NewHtmlRenderer`、`NewFormatRenderer`（回 Markdown）、`NewVditorIRRenderer`、`NewVditorSVRenderer`、`NewVditorRenderer`（wysiwyg）、`NewJSONRenderer` 等，都在 `render/`。【源码】
- 反向：`html.Node` 到 AST 的 `genASTByVditorIRDOM`、`genASTByVditorDOM`，即「DOM 到 AST」也由 Lute 提供。【源码】
- IR 不是独立的中间数据结构，而是「带 `data-type`、`data-block` 属性和标记符 span 的 HTML」。它既是显示物也是状态。

## 8. 能否作为 Go 依赖直接用

| 项 | 判断 | 依据 |
| --- | --- | --- |
| Go 版本 | 兼容 | Lute `go 1.26`；corra `go.mod` 为 `go 1.27.3`【源码】 |
| 依赖冲突 | 低 | Lute 要求 `chroma/v2 v2.27.0`，corra 已用同一版本【源码】。另有 `x/text`、`gopherjs v1.17.2`、`regexp2/v2` |
| 原生编译 | 未核对 | `lute.go` 无构建标签，直接 import `github.com/gopherjs/gopherjs/js`（用于 `SetJSRenderers`）。这个包在 darwin、windows、linux 上能否正常编译：【未核对】。需要在新目录跑 `go build` 验证 |
| 许可证 | Mulan PSL v2 | `LICENSE` 为木兰宽松许可证第 2 版，`README_en_US.md` 末尾和 Go 文件头同样写 v2。英文条款要求再分发时附带许可证副本并保留版权、专利、商标和免责声明；含专利许可，且在被许可人发起专利诉讼时终止。【源码】`javascript/build.sh` 头部仍写 v1，与其他文件不一致，应以 `LICENSE` 为准。corra 仓库根目录目前没有 `LICENSE` 文件，如要分发二进制，需要先决定 corra 自己的许可证并在第三方声明里列出 Lute。法律解释未核对 |
| 源码位置 | 缺失 | 见第 3 节，AST 无位置信息 |
| 维护活跃度 | 活跃 | 最近 commit 2026-11-23【源码】；但 API 随思源笔记需求变化，节点结构体里含思源专用字段 |

可行的用法，按侵入程度由低到高：

1. 只用 `Format`/`FormatStr`。作为「源码模式一键格式化」功能。几乎无风险。
2. 替换 `richtext.ParseMarkdown` 之上的 `tokenize`（`mgdoc/classify.go` 里的私有区占位）。用 `parse.Parse` 后遍历 AST，直接识别 `NodeMark`、`NodeSup`、`NodeSub`、`NodeInlineMath`、`NodeFootnotesDef`、`NodeCallout`，生成 `richtext.Paragraph`。需要把每个节点类型映射到 `richtext.Attributes`。
3. 保留 `Split` 切块，不依赖 Lute 给位置：每个块自己记录源码起止，Lute 只负责块内解析。这是避开「无源码位置」的现实办法。
4. 照搬 Vditor 的 IR（DOM 里留标记符）：不可行，MyGo 没有 DOM，见第 9 节。

## 9. 在 MyGo / Go 里能否实现

| 项 | 判断 | 改动点 | 风险 |
| --- | --- | --- | --- |
| 「DOM 即模型」 | 不可行 | MyGo 无 DOM。`richtext.Text` 是模型，但没有「标记符 span 可显隐」的概念 | 若要在文稿模式显示标记符，只能把标记符作为带特殊 `Tag` 的 run 放进文本，再由 `TagStyle` 控制颜色或隐藏。`Tag` 是否支持零宽或隐藏：【未核对】 |
| 块级「自旋」 | 可行 | 对应 `Codec.Load(Save(block))`：只对光标所在块做保存再加载，替换该段落。`Split` 已按块切分 | 光标要跨往返保持。可借用 `‸` 字符法：保存前在光标处插入哨兵字符，加载后找回。哨兵字符必须在 Markdown 里不被转义；`richtext.Text.Markdown()` 对未知字符的处理未核对 |
| 光标字符法 | 可行 | 与 corra 现有私有区方案同一思路，`rawBase` 起的字符已经在用 | 与 `rawBase` 区间不能重叠 |
| 补丁式撤销 | 可行 | 把 `editor/undo.go` 的 `snapshot{src}` 改为存 diff。Go 有 `sergi/go-diff`（diff-match-patch 的 Go 移植，未核对其许可证） | 只存文本补丁不含 `richtext` 的附件状态；文稿模式的撤销以源码为准即可，因为 `Load` 能重建附件 |
| Lute 作 Go 依赖 | 部分可行 | 见第 8 节方案 2 和 3 | 原生编译、许可证声明、API 变动，均需验证 |
| 保留原文的「受保护块」 | corra 已优于 Vditor | Vditor 会规范化所有块，corra 的 `rawRef` 保证表格、HTML 等逐字节还原 | 不要为换 Lute 而放弃 `rawRef` 与 `stable()` 检查 |

## 10. 对 corra 的启发

1. 「光标作为字符穿过往返」是低成本的位置保持办法。corra 的块往返（`Load`/`Save`）可以用同样思路，把光标变成块内一个哨兵字符，不需要 AST 位置。
2. Vditor 的 IR 本质是「源码一直在，显隐靠样式」。MyGo 中对应的做法是源码模式文本加装饰层，细节见 `codemirror6.md`。Vditor 自己的做法依赖 DOM，不能直接搬。
3. Lute 的 AST 无位置信息，意味着它适合「整块解析成结构」，不适合「编辑时增量映射」。corra 若引入，应只用在块内解析，并继续由 `Split` 掌握块边界。
4. 撤销用补丁而非快照能减小内存，Vditor 每模式 50 步。corra 现在 200 步整份快照，文档大时占内存明显。
5. 许可证与原生编译两件事要先验证再决定引入，不要先改代码。

## 11. 没能核对的点

- Lute 在 darwin、windows、linux 的原生编译与实际输出（未运行）。
- 任意 Markdown 往返差异的实测。
- `gopherjs/js` 在非 JS 平台是否可编译。
- Vditor 切换模式时光标位置是否保持（`EditMode.ts` 未细读选区部分）。
- `richtext.Attributes.Tag` 能否表达「隐藏的标记符」，以及 `Text.Markdown()` 对哨兵字符的处理。
