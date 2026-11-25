# 阶段 0 记录：MyGo 升级与编辑器原型

计划：`2026-11-24-document-model.md` 阶段 0。

## 0.1 MyGo 升级

- `github.com/egoist/mygo` 从 v0.7.3 升到 v0.10.0（当时的最新版本），`go mod tidy`。
- 不需要改任何代码：Linux 上 `go build ./...`、`go vet ./...`、`go test ./...` 全部通过；`GOOS=windows`、`GOOS=darwin CGO_ENABLED=0` 交叉编译通过。
- `nativerich` 不导入 MyGo，只通过 `main.go` 拿窗口句柄，所以升级对它的影响只在 `main.go`，而 `main.go` 已在 darwin 交叉编译中通过。

**没有验证的**（这台机器是 Linux arm64，没有 clang，也没有 macOS 和 Windows 图形环境）：

- [ ] macOS 上带 cgo 构建和 `nativerich` 测试
- [ ] 三平台手动冒烟：打开文档、富文本编辑、源码模式、导出 PDF
- [ ] `go tool mygo build` 打包

## 0.2 编辑器原型

代码曾在 `internal/editor/spike/`，阶段 3 完成后按计划删除（正式实现在 `internal/editor`）。多段落编辑器，直接实现 `ui.TextInputClient`，用 `ui.ShapeText` 排版，不接文档模型。

用 MyGo 的无窗口 `ui.Tester` 自动验证（`spike_test.go`，全部通过）：

| 项目 | 测试 |
|---|---|
| 输入、回车拆段 | `TestTypingAndNewlines` |
| 输入法组字：组字文本带标记、输入法拿得到光标位置、提交后标记清除 | `TestCompositionCommitsOnce` |
| 跨段落选区被输入替换，段落合并 | `TestSelectionAcrossParagraphsReplacedByInput` |
| Shift+方向键跨段落扩展选区 | `TestShiftArrowsExtendAcrossParagraphs` |
| emoji 是一个光标步长，删除不会留下半个代理对 | `TestEmojiIsOneCaretStep` |
| 行首退格合并段落 | `TestBackspaceJoinsParagraphs` |
| 上下移动保持列位置 | `TestVerticalMoveKeepsColumn` |
| 点击定位光标 | `TestClickPlacesCaret` |

**性能数据**（Linux arm64，无窗口）：1000 段、每段约 200 字（中英混合）的文档，原型每次按键都重新排版全部段落，耗时约 **390 ms/次**，也就是排一个这样的段落约 **0.39 ms**。结论：

- 必须按节点缓存排版结果（设计 §5.3）。只重排被改的段落时，按键的排版开销在 1 ms 以内。
- 打开一篇 1000 段的文档，全部排版约 0.4 秒，所以首屏只排可见区域是必要的。

**MyGo v0.10.0 里可以直接用的能力**（读 `docs/ui/text-input-client.md`）：

- `ShapeRichText` / `ShapeRichTextWith`：带样式的 span、对齐、首行缩进；`ObjectWidth` 让 U+FFFC 成为行内对象（行内公式、图片），`BaselineShift` 用于上下标。
- `TextLayout`：`CaretAt`、`PositionAt`、`MoveCaret`（视觉移动，处理双向文本）、`SelectionRanges`、`SelectionRects`、`Lines`、`Objects`、`PreviousBoundary`、`NextBoundary`。
- `ui.TextBuffer`：带 UTF-16 索引的持久化 rope，源码模式可以用。
- `Tester.Compose`：在无窗口测试里模拟输入法组字。

**没有验证的**（需要在真实平台上手动检查）：

- [ ] macOS：拼音输入法、日文输入法、VoiceOver 读出段落
- [ ] Windows：微软拼音、讲述人
- [ ] Linux：fcitx5 或 ibus

闸门判断：无窗口测试覆盖到的部分全部可行，没有发现需要回到设计讨论的问题。真实平台的输入法检查列在上面，在阶段 3 接入应用之前需要有人完成。
