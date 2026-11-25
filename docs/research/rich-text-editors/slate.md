# Slate 的数据建模，及对 corra 的启发

## 0. 版本与可信度标记

读取的版本如下。

| 对象 | 版本 / commit |
| --- | --- |
| slate | 0.147.4，ianstormtaylor/slate `7e6e576`（2026-11-20），`packages/slate/src` |
| slate-history | 0.144.0，同一 commit |
| 官方文档 | 同一 commit 下的 `docs/concepts/*.md`、`docs/api`，即 docs.slatejs.org 的源文件 |
| @slate-yjs/core | 1.0.2，slate-yjs/slate-yjs `6779497`（2026-11-24） |
| remark-slate-transformer | 0.12.2，inokawa `c6c2a29`（2026-11-20） |
| MyGo | v0.7.3（corra `go.mod` 所用版本，读 Go 模块缓存中的源码） |
| corra | `733d0fe` |

每条结论后的标记含义如下。

- [源码]：已在 Slate 官方仓库源码核对。
- [文档]：已在 Slate 官方文档核对。
- [三方源码]：已读第三方库源码（slate-yjs、remark-slate-transformer），不是 Slate 官方。
- [三方]：只来自第三方 README、issue 或搜索摘要。
- [推断]：由已读代码推出，没有实测。
- [未核对]：没有读到来源，不当作事实。

## 1. 数据结构

**节点。** `Node = Editor | Element | Text`，`Descendant = Element | Text`，`Ancestor = Editor | Element`，`NodeEntry<T> = [T, Path]` [源码 `interfaces/node.ts`]。

```ts
interface BaseText    { text: string }
interface BaseElement { children: Descendant[] }
```

`Text` 与 `Element` 都是 `ExtendedType<'Text' | 'Element', Base...>`，应用通过 TypeScript 的 `CustomTypes` 声明合并加字段，例如 `type`、`url` [源码 `interfaces/text.ts`、`element.ts`]。运行时没有类标签。有 `text: string` 就是 Text，有 `children` 就是 Element [源码 `Text.isText`]。

**位置。**

- `Path = number[]`，从根逐层给出子节点下标 [源码 `interfaces/path.ts`]。
- `BasePoint { path, offset }`，只能指向 Text [源码]。
- `BaseRange { anchor, focus }` [源码]。
- `Location = Path | Point | Range` [源码 `interfaces/location.ts`]。
- `Span = [Path, Path]` [源码]。

`Editor` 自身也是节点，持有 `children`、`selection`、`operations`、`marks` [源码 `BaseEditor`]。

**格式不是 mark 对象。** 格式直接作为 Text 的额外属性存放，例如 `{ text: 'x', bold: true }`。文档称这些属性“有时被叫作 marks” [文档 `02-nodes.md`]。`EditorMarks = Omit<Text, 'text'>` [源码]。后果如下。

- 格式是无序的键值集合。`**_a_**` 与 `_**a**_` 在模型里相同，嵌套顺序不存在。
- `Editor.addMark`：选区展开时调用 `Transforms.setNodes(..., { split: true })` 把 Text 切开再设属性；选区折叠时只写 `editor.marks`，这是待用格式，不进文档，`set_selection` 时清空 [源码 `editor/add-mark.ts`、`core/apply.ts`]。
- 相邻 Text 是否合并由 `Text.equals(a, b, { loose: true })` 决定，即忽略 `text` 后做深比较 [源码]。

**Element 的 children 与“schema”。** 没有声明式 schema。结构约束由 editor 上可覆盖的函数表达 [源码 `BaseEditor`]。

- `isInline(element)`、`isVoid(element)`，`createEditor` 默认都返回 false。
- `isElementReadOnly`、`isSelectable`、`isIsolated`、`markableVoid`、`inlineEdgeAffinity`、`leafEdgeAffinity`。
- `normalizeNode(entry, options)`、`shouldNormalize(...)`、`getDirtyPaths(op)`、`getSplitProperties(node, path)`。

覆盖方式是包装旧函数，例如 `const { isVoid } = editor; editor.isVoid = el => el.type === 'editable-void' || isVoid(el)` [源码 `site/examples/ts/editable-voids.tsx`]。

## 2. 操作

九种操作，字段如下 [源码 `interfaces/operation.ts`]。

| type | 字段 |
| --- | --- |
| `insert_node` | `path`, `node` |
| `remove_node` | `path`, `node` |
| `set_node` | `path`, `properties`, `newProperties` |
| `split_node` | `path`, `position`, `properties` |
| `merge_node` | `path`, `position`, `properties` |
| `move_node` | `path`, `newPath` |
| `insert_text` | `path`, `offset`, `text` |
| `remove_text` | `path`, `offset`, `text` |
| `set_selection` | `properties`, `newProperties`（三种形态：`null→Range`、`Partial→Partial`、`Range→null`） |

要点如下。

- `set_node` 不能改 `children` 和 `text`（`NON_SETTABLE_NODE_PROPERTIES`）[源码 `transforms/general.ts`]。`newProperties` 里值为 `null` 表示删除该属性，所以属性值本身不能是 `null` [文档 `11-normalizing.md` 约束 7]。
- `merge_node` 把 `path` 处的节点并入前一个兄弟，结果沿用前者的属性。`properties` 保存被并入节点的属性，供反转用。`split_node` 的 `properties` 是新右半节点的属性，来源是 `editor.getSplitProperties`，默认 `Node.extractProps` [源码；changelog 0.143.0]。
- Transforms 与 Operation 的关系：`Transforms.insertNodes/setNodes/wrapNodes/unwrapNodes/liftNodes/splitNodes/mergeNodes/moveNodes/delete/setChildren` 是高层命令，最终都调用 `editor.apply(op)`。`GeneralTransforms.transform(editor, op)` 是唯一的 reducer [源码；文档 `05-operations.md`]。`wrap/unwrap/lift` 由上述九种组合而成。
- `apply` 的顺序 [源码 `core/apply.ts`]：先变换所有 path/point/range ref，再更新 dirty paths，再 `GeneralTransforms.transform`，再推入 `editor.operations`，再 `Editor.normalize`，最后在微任务里调用 `onChange` 并清空 `operations`。

**反转。** 因为操作带着旧数据，`Operation.inverse` 是纯函数 [源码]。

- `insert_node ↔ remove_node`、`insert_text ↔ remove_text`：只换 type。
- `set_node`、`set_selection`：互换 `properties` 与 `newProperties`。
- `split_node → merge_node`，`path` 取 `Path.next`。`merge_node → split_node`，`path` 取 `Path.previous`。
- `move_node`：兄弟间移动直接互换 `path/newPath`。跨父节点时用 `Path.transform` 算出被移走节点的原位置和插入点前一位置。

## 3. 规范化

**脏路径收集。** `editor.getDirtyPaths(op)` 按操作类型返回 [源码 `core/get-dirty-paths.ts`]：

- `insert_text`、`remove_text`、`set_node`：该路径的全部祖先层级 `Path.levels`。
- `insert_node`：祖先层级加新节点所有后代。
- `remove_node`：祖先。
- `merge_node`：祖先加前一个兄弟。
- `split_node`：层级加下一个兄弟。
- `move_node`：新旧祖先加目标路径。
- `set_selection`：空。

`updateDirtyPaths` 先把已有脏路径按本次操作变换位置，再合并新路径。路径多于 8 条时改用字符串 key 的 Set 去重 [源码 `core/update-dirty-paths.ts`，`SCAN_LIMIT = 8`]。

**执行。** `Editor.normalize` 在 `withoutNormalizing` 里循环，每次 `pop` 一条脏路径调用 `editor.normalizeNode`。开头先单独过一遍“空 children 的 Element”，补空 Text。迭代上限是 `初始脏路径数 × 42`，超出抛错，用来拦截不收敛的自定义规则 [源码 `editor/normalize.ts`、`core/should-normalize.ts`]。`{ force: true }` 会把整棵树都标脏 [源码]。

**默认规则** [文档 `11-normalizing.md`；源码 `core/normalize-node.ts`]。

1. Element 至少有一个子节点，空则插入 `{ text: '' }`（void 也一样）。
2. 相邻 Text 属性相同则合并。其中空 Text 与另一个 Text 相邻时直接删掉空的。
3. Element 的子节点要么全是块，要么全是行内/文本，由第一个子节点决定。不符合的块被 `unwrapNodes`，行内/文本被删除或包进 `fallbackElement`。
4. 行内元素前后必须是 Text，否则补空 Text。
5. 根节点只能含块。
6. 节点必须可 JSON 序列化，属性值不能为 `null`。这两条是约定，不由代码强制。

Text 节点本身没有核心规范化规则。规则是多遍的：修一个问题就返回，修改又会把该节点重新标脏。

## 4. 不可变更新与 ref

**Immer。** 当前版本的核心代码不再使用 Immer。0.116.0 起“Immer 只用于生成 fragment”，0.120.0 “移除最后的 Immer 痕迹并去掉依赖” [源码 `packages/slate/CHANGELOG.md`；`package.json` 无 immer 依赖]。现在的做法是手写结构共享。`modifyDescendant` 沿路径向上重建祖先，`replaceChild` 对 children 数组做 `slice` 后改一项，未涉及的子树引用不变 [源码 `utils/modify.ts`]。所以每次 `insert_text` 的成本是路径深度乘各层兄弟数量的数组拷贝。

**Ref。** `Editor.pathRef/pointRef/rangeRef` 创建对象 `{ current, affinity, unref() }` 并登记在 editor 的 Set 里。`apply` 对每个 ref 调 `PathRef.transform(ref, op)`，内部是 `Path.transform(path, op, { affinity })`。节点被删则 `current = null` 并自动 `unref` [源码 `editor/path-ref.ts`、`interfaces/path-ref.ts`、`path.ts`]。Point 的 `transform` 处理文本偏移，affinity 决定在插入点上向前还是向后 [源码 `point.ts`]。ref 不用时必须手动 `unref`，否则每次 `apply` 都要多变换一个对象 [推断]。

## 5. 历史与协同

**slate-history** [源码 `with-history.ts`、`history.ts`、`rebase.ts`]。

- `history = { undos: Batch[], redos: Batch[] }`，`Batch = { operations, selectionBefore }`。栈上限 100，超出从队首丢弃。
- `set_selection` 不入栈（`shouldSave`），但每个批次保存开始时的 `selectionBefore`。
- 合并判定，按顺序：显式的 `withMerging/withoutMerging` 优先；没有上一批则新开批；上一批最后一个操作仍在 `editor.operations` 里（同一次 flush，即同一个微任务周期）则合并；否则 `shouldMerge`：同一 path 的 `insert_text` 偏移首尾相接，或 `remove_text` 向前连续删除；另外，规范化尚未结束（`!isSettled`）且上一批最后一个操作仍在 `editor.operations` 里时也并入，使规范化产生的操作与触发它的操作同属一步 [源码；changelog 0.140.0]。没有时间窗口。
- undo：取最后一批，`operations.map(Operation.inverse).reverse()` 在 `withoutSaving` 与 `withoutNormalizing` 下依次 `apply`，然后恢复 `selectionBefore`。redo 是按原顺序重放。新的可保存操作会清空 `redos`。
- 0.144.0 新增 `rebaseHistory`：对 `withoutSaving` 应用的外部操作，用 `transformOperations` 把栈中批次平移到新坐标，内容被删或无法应用的批次丢弃 [源码 `rebase.ts`；changelog]。同版本新增 `Transforms.setChildren`，把外部文档按 diff 转成操作，整体记为一步撤销 [changelog 0.144.0]。

**slate-yjs** [三方源码 `packages/core/src`]。

- 共享根是一个 `Y.XmlText`（`sharedRoot`）。每个 Slate Element 是父级 delta 里嵌入的 `Y.XmlText`；Text 叶子是字符串片段，格式作为 delta 的 attributes；Element 属性是 `XmlText` 的 attribute。
- 本地方向：`applySlateOp` 按 `op.type` 分发到 `insertText`、`removeText`、`insertNode`、`removeNode`、`setNode`、`splitNode`、`mergeNode`、`moveNode`，`set_selection` 是空操作。对 Text 做 `split_node/merge_node` 实际变成对 Yjs 文本做 `format`，也就是改 attributes，不产生真正的节点边界。
- 远端方向：`applyRemoteEvents` 把 Yjs 事件在 `withoutNormalizing` 里转成 Slate 操作；连接时会做一次规范化。
- 撤销用 `Y.UndoManager`（`withYHistory`），追踪本地 origin，不使用 `Operation.inverse`。
- 该库开发依赖 slate `^0.82.1`，peer 要求 `>=0.70.0`。与 0.147 是否兼容 [未核对]。

## 6. Markdown

**没有官方往返。** 官方文档只给纯文本和 HTML 的手写 `serialize` 示例，说 JSON 数据“容易序列化成 HTML、Markdown 等格式” [文档 `10-serializing.md`]。官方站点有三个 Markdown 相关示例 [源码 `site/examples/ts`]。

- `markdown-shortcuts`：输入 `#`、`>` 等触发 `Transforms.setNodes` 改块类型。
- `markdown-preview`：用 Prism 做 decoration，文档里始终是原始 Markdown 文本，不改节点，因此无需往返。
- `markdown-sync`：左 Slate、右 textarea，借 `remark-slate-transformer` 双向转换。

**remark-slate-transformer** [三方源码，`src/transformers/mdast-to-slate`、`editor/with-mdast.ts`]。

- 元素的 `type` 直接用 mdast 类型名；`bold/italic/delete/inlineCode` 在叶子上是布尔属性 `strong/emphasis/delete/inlineCode`，可通过 `marks` 选项改名。
- 只保留指定字段，没有 `position`。例如 `heading` 只留 `depth`，`list` 留 `ordered/start/spread`，`listItem` 留 `checked/spread`，`table` 留 `align`，`link` 留 `url/title`，`code` 留 `lang/meta`。
- 表格是 `table/tableRow/tableCell` 三层元素，单元格里是行内内容。
- `code`、`math`、`inlineMath`、`yaml`、`toml`、`html` 都是 Element，内容放在唯一的 Text 子节点里，仍然可编辑。
- `withMdast` 设置 void 的类型只有 `thematicBreak, break, image, imageReference, footnoteReference, definition`。行内类型是 `link, linkReference, break, image, imageReference, footnoteReference, inlineMath`，加带 `inline: true` 的 `html`。
- 丢失的信息 [推断，由上面的字段清单得出]：原文的分隔符与写法（`*` 与 `_`、`-` 与 `*`、setext 与 ATX 标题、围栏字符）、表格列对齐空白、转义写法、源码位置。回写时由 `remark-stringify` 按默认风格重新生成。叶子上的格式无序，所以 `**_a_**` 回写成固定的一种嵌套。
- `htmlMarks` 选项把成对的 `<u>`、`<sup>`、`<sub>` 转成叶子属性。开标签到闭标签之间的文本被标记，孤立的闭标签被丢弃 [三方，README]。

**hanford/remark-slate**：README 称其 `serialize` 是“opinionated”，列表有“assumptions”；issue #1、#68 反映输出不完整 [三方，来自搜索摘要，未打开 issue 正文]。**slate-md**：[未核对]。

## 7. 在 MyGo / Go 里能否实现

### 7.1 corra 与 MyGo 的现状（已读代码）

- 读：`Split` 按空行切块，围栏代码与 `$$` 内部空行保留；`classify` 判块类型；普通块先 `tokenize`，把行内数学、脚注、`==mark==` 等换成私有区字符，交给 `richtext.ParseMarkdown`，再 `expand` 还原成附件或带 `Tag` 的 run。用 `stable()`（解析、`Markdown()`、再解析后形状相等）判断无损，不通过就整块降级为 `blockOther` 的受保护块。
- 受保护块是一个只含一个 `ObjectReplacement` 的段落，附件的 `Data` 是 `rawRef{raw, block}`。`Save` 把私有区字符换回原文。
- `editor/undo.go`：整份源码快照栈，上限 200，1 秒合并。它由 `app.Hist` 使用，推入点在 `SetSource`、`Refresh`。
- 富文本模式的编辑走 `views/rich_editor.go` 的 `syncEditor`，每次 `Snapshot()` 发现 `Document.Version()` 变化就 `codec.Save` 整份文档，再调 `MarkSourceDirty`，这条路径不推 `Hist`。
- 富文本模式的撤销在 MyGo 内部：`richtext.Document` 的 `undoStep{before, after Text, selBefore, selAfter, start, oldEnd, newEnd}`，上限 200，输入 1 秒内合并。`Text` 是 rope 加段落序列，快照之间共享未改动部分 [MyGo v0.7.3 `ui/richtext/document.go`、`text.go`]。macOS 的 NSTextView 撤销行为 [未核对]。

### 7.2 逐项判断

**A. Path + Operation + inverse 能否替代整份源码快照。**

- 富文本模式：不需要。MyGo 已有基于持久化 `Text` 的撤销，成本接近增量。corra 若自建 Slate 式树，MyGo 的 `Text` 不能直接从树构造（只能经 `TextOf(paragraphs...)` 或 `Document.Edit`），等于多一层树到 `Text` 的转换，收益不明。
- 源码模式：可以做，而且不需要 Path。源码是一维字符串，Slate 的 `insert_text/remove_text` 退化为 `{Offset, Removed, Inserted}`，反转即互换。
- 改动点：`editor/undo.go` 的 `snapshot{src, caret}` 改为 `edit{off, removed, inserted, caretBefore, caretAfter}`。`Push(src, caret)` 的调用点（`app/state.go` 约第 178、206、268 行）可以保持签名，内部对上一状态求公共前后缀，得到一条 edit。合并规则可借 `shouldMerge`：按偏移首尾相接合并，保留 1 秒窗口作为上限。
- 收益：200 步的内存从 O(文档大小 × 200) 降到 O(改动量 × 200)。
- 风险：每次 Push 要对上一状态求前后缀，大文档上是 O(n) 的字符串比较，但比保存整份 O(n) 拷贝便宜。外部变更（文件冲突重载、`PushBase`、模式切换后重新解析）使操作序列失效，必须清栈并重设基线。Slate 为此在 0.144 引入了 `rebaseHistory` 与 `setChildren`，corra 没有同等需求，直接清栈更简单。`Undo(current, caret)` 里的 `current` 参数现在没被使用，改造时可顺手去掉。

**B. 用 Go 接口实现 Node 树的代价。**

- 写法：`type Node interface{ node() }`，`Text` 与 `Element` 两个结构体。Slate 的属性是任意键值，Go 里用 `map[string]any` 会失去类型检查，也使相等比较变成逐键比较。MyGo 的 `Attributes` 是可比较结构体，相邻 run 相等判断是 O(1) 的结构体比较（`Attributes` 的注释写明它们可比较，属性相同的 run 算作一个）。如果自建树，应沿用固定字段加 `Tag string`，不要照搬开放属性。
- 不可变更新：沿路径重建祖先、`slices.Clone` 子切片，Go 里写起来直接。每次编辑的拷贝量是路径上各层的子节点数。顶层有上千个块时，每次按键拷贝上千个接口值（每个 16 字节），大约十几 KB [推断]，可接受，但不如 MyGo 的持久化序列。
- ref：Go 没有对象生命周期钩子，`PathRef.Unref` 必须显式调用，忘记就泄漏并拖慢每次 `Apply`。
- 序列化与协同：Slate 要求 JSON 可序列化是为协同服务。corra 不做协同，这条约束可以不要。
- 结论：不引入通用节点树。如果要用，只用在 `mgdoc` 内部作为块级中间结构，用 `Kind` 枚举加结构体字段，不用接口加 map。

**C. 规范化对 Markdown 往返保真的帮助或危害。**

- 已有的：MyGo 的 `Text` 注释写明“相邻且属性相同的 run 合并”，等价于 Slate 规则 2，不用再做。
- 不需要的：规则 1、4（空 Text 垫位、行内元素两侧补 Text）是为了让 DOM 选区总能落在某个 Text 里。MyGo 的偏移是整段文本的 rune 偏移，附件占一个 rune，这两条没有对应需求。
- 危害：规范化会改写用户内容，对“字节级还原”有风险。例如合并相邻同格式 run 会抹掉 `**a****b**` 的原始边界。remark-slate-transformer 的丢失清单（见 6）就是这类归一化的结果。corra 现有做法是 `stable()` 门槛加受保护块兜底，保真优先，应保留。
- 可借鉴的：（1）迭代上限，防自定义规则不收敛；（2）脏路径思想。`Snapshot` 每次变化都 `Save` 整份文档，O(文档)。若按段落缓存其 Markdown 片段，只重算被改的段落，就是 Slate 脏路径的对应物。MyGo 的 `Document` 有 `newID` / `IdentifyParagraphs` 的注释说明段落带 ID，但外部能否读取 ID [未核对]。
- 规则放在哪里：只对 corra 自己拥有的结构做不变量检查，例如“受保护块段落只含一个附件”。`Save` 已经在做（块附件独占段落，同段其他文本被拆出去）。

**D. void 元素与受保护块的对应。**

| Slate | corra / MyGo |
| --- | --- |
| 块级 void，`children: [{ text: '' }]`，数据放在元素属性 | 一个 `ObjectReplacement` 段落，`Attachment.Data = rawRef{raw, block: true}` |
| 行内 void，两侧必须有空 Text | 行内附件（行内数学、脚注标记），`rawRef{raw}`，无需垫位 |
| `isVoid` 决定是否可编辑内部 | 附件只能整体选中、删除，编辑要切到源码模式 |
| 删除 void 是 `remove_node`，反转时带回完整节点 | 删除附件是删一个 rune，撤销依赖 `Text` 快照里的附件指针 |
| `isElementReadOnly`、`isSelectable`、`markableVoid` 提供更细的状态 | 没有对应项，目前也不需要 |

- 对应关系成立。Slate 的 void 和 corra 的受保护块在语义上相同：模型内不透明，渲染由应用负责。差别是 Slate 用空 Text 子节点保证选区可落入，MyGo 用一个占位 rune。
- remark-slate-transformer 选了另一条路：表格、代码、数学、HTML 不是 void，原文作为可编辑 Text。corra 选择只读加源码模式编辑，保真更强，编辑体验更弱。这是产品取舍，不是模型缺陷。
- 私有区字符 `0xF0000+i` 只是为了让原文片段穿过 `ParseMarkdown`。树模型可以用“行内 void 元素加 `raw` 属性”直接表达，不需要占位字符。但那意味着自写 Markdown 解析器去替换 `richtext.ParseMarkdown` 和 `Text.Markdown()`，工作量大，不建议为此引入。
- 风险 [推断，未实测]：`mgdoc` 读入时没有清理源文里本来就存在的 U+F0000 起的字符（在 `mgdoc` 中搜索 `rawBase` 只有写入和还原两处）。含这类字符的文档，`expand` 可能把它当成占位符处理。建议在 `tokenize` 前检测，命中则整块降级为受保护块。

### 7.3 建议的改动点与风险汇总

| 序号 | 改动 | 位置 | 收益 | 风险 |
| --- | --- | --- | --- | --- |
| 1 | 源码模式撤销改为 edit 记录（前后缀 diff） | `editor/undo.go` | 内存从 O(n×200) 降到 O(改动×200) | 外部重载须清栈；diff 的 O(n) 比较 |
| 2 | 保存时按段落缓存 Markdown 片段 | `internal/mgdoc/codec.go` `Save`、`views/rich_mygo.go` `Snapshot` | 大文档下每次按键不再全量序列化 | 依赖 MyGo 段落 ID 的可读性 [未核对]；缓存失效条件要写对 |
| 3 | 读入前检测 U+F0000 起的字符 | `internal/mgdoc/classify.go` `tokenize` | 消除占位符冲突 | 极少见，降级为受保护块即可 |
| 4 | 不引入 Slate 式节点树 | 无 | 避免与 MyGo `Text` 并存两套模型 | 失去以 Path 寻址的统一操作语言 |
| 5 | 保留 `stable()` 门槛，不加改写文本的规范化 | `internal/mgdoc/codec.go` | 保真 | 无 |

## 8. 没能核对的点

- slate-yjs 与当前 Slate 0.147 的兼容性（该库测试用 0.82.1）。
- hanford/remark-slate 的 issue 正文，以及 slate-md。
- NSTextView 路径的撤销行为，MyGo `Document` 的段落 ID 是否对外可读。
- 对 `U+F0000` 冲突的实际表现，没有写测试复现。

## 9. 来源

- https://github.com/ianstormtaylor/slate （`7e6e576`；`packages/slate/src`、`packages/slate-history/src`、`docs/concepts`、`site/examples/ts`、各包 `CHANGELOG.md`）
- https://docs.slatejs.org/ （内容与上述 `docs/` 源文件对应）
- https://github.com/BitPhinix/slate-yjs （`6779497`，`packages/core/src`；其 `package.json` 中的仓库地址是 github.com/slate-yjs/slate-yjs）
- https://github.com/inokawa/remark-slate-transformer （`c6c2a29`）
- https://github.com/hanford/remark-slate 、https://www.npmjs.com/package/remark-slate （README 与搜索摘要）
- MyGo v0.7.3：`github.com/egoist/mygo/ui/richtext`（本机 Go 模块缓存）
