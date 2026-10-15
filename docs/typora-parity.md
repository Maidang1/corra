# Corra × Typora 能力对照（2026-10-14）

对照 https://typora.io/ 与 support.typora.io 文档列出的全部能力。状态说明：

- ✅ 已有（本次任务前）
- 🆕 本次新增
- 🔶 框架限制（MyGo/无 webview/原生栈），下方有说明
- ❌ 不实现，附理由

## 编辑器核心

| Typora 能力 | Corra 状态 |
|---|---|
| 无缝所见即所得（live preview 单栏） | ✅ macOS 原生 NSTextView 富文本编辑；🔶 Windows/Linux 无原生富文本控件（MyGo 限制），保留源码+预览 |
| 源码模式切换 | ✅ Cmd+E |
| 标题 1–6（快捷键 Cmd+1..6） | 🆕 格式菜单 + 快捷键 |
| 粗体/斜体/删除线/行内代码 | ✅（粗体 Cmd+B、斜体 Cmd+I） |
| 下划线 `<u>` | 🆕 Cmd+U，三端（原生/预览/PDF 文本/HTML）一致 |
| 高亮 `==x==` | 🆕 Cmd+Shift+H |
| 上标 `^x^` / 下标 `~x~`（Typora 语义，含 `<sup>/<sub>`） | 🆕 |
| 行内 HTML（`<u>` `<mark>` `<kbd>` `<sub>` `<sup>` `<br>`） | 🆕 预览/导出解释渲染；未知标签按源码保护 |
| 硬换行（行尾两空格 / `\` / `<br>`） | 🆕 原生编辑器内作为同段落内的软视觉换行可见可编辑；Shift+Return 直接插入硬换行（正文/引用），往返保持 Markdown 原文 |
| 清除格式 Cmd+\ | 🆕 |
| 链接插入对话框 Cmd+K（文本+URL） | 🆕 |
| 链接可点击：浏览器打开 / `#标题` 页内跳转 / 相对 .md 互跳 | 🆕（预览模式） |
| 自动链接 linkify | ✅ GFM |
| 列表：无序/有序/任务 | ✅ 原生编辑器内；Enter 续项且保持紧凑列表（`- a\n- b`），空项 Enter 退出列表，有序列表插入/删除后自动重编号，Shift+Tab 退回正文 |
| 列表 Tab 嵌套缩进 | 🔶 嵌套列表仍以源码行编辑；Tab 在列表项内不再插入 `&#9;` 字符 |
| 块首退格（标题/列表/引用 → 正文） | 🆕 与 Typora 一致，单步撤销 |
| 引用内 Enter 续段（`>` 空行分段），空行 Enter 退出引用 | 🆕；多段引用 `> a\n>\n> b` 可直接富文本编辑 |
| 最小转义（`.` `(` `!` `#` 等正文标点保持原样，仅在会改变语义时转义） | 🆕 |
| 选中文字输入 `*` `_` `~` `` ` `` 直接加格式 | 🆕 |
| 列表类型互相切换（菜单/快捷键） | 🆕 格式菜单 + Cmd+Alt+U/O/L |
| 引用块 | ✅ |
| 水平分割线 | ✅（插入菜单 🆕） |
| 任务列表点击勾选 | ✅ |
| 代码块语法高亮（~100 语言） | ✅ 200+ 语言（chroma） |
| 代码块行号开关 | 🆕 视图菜单 |
| 自动配对（括号/引号/ Markdown 符号） | 🆕 原生编辑器内实现；源码编辑器由 MyGo editor 自带配对 |
| 智能标点 SmartyPants（弯引号/破折号/省略号） | 🆕 视图菜单开关（预览与导出，不源码） |
| 拼写检查 | 🆕 macOS 原生（NSTextView 连续拼写检查），视图菜单开关 |
| 文本替换（macOS 系统文本替换） | 🆕 随系统设置（NSTextView automaticTextReplacement） |
| 专注模式（淡化非当前段） | 🆕 原生编辑器，视图菜单开关 |
| 打字机模式（光标行垂直居中） | 🆕 原生编辑器 |
| 文档内查找替换 Cmd+F | ✅ 源码模式已有；🆕 富文本模式走 NSTextView 系统查找栏 |
| 字数统计（词/字符/行/阅读时长） | 🆕 编辑栏底部状态条，中英文混排规则同 Typora |
| 大纲面板（点击跳转） | 🆕 第四栏大纲（Cmd+Alt+T）；预览滚动定位、富文本滚动定位、源码光标定位 |
| [TOC] 目录块 | 🆕 预览渲染为可点击目录；HTML 导出为带锚点链接的列表 |
| YAML Front Matter | 🆕 预览显示为信息卡；富文本中为受保护源码块，逐字节往返 |
| 脚注 | ✅ 渲染与文末列表；🆕 Cmd+Alt+F 插入脚注；🔶 悬停显示脚注内容未做（MyGo 文本流无 hover 富文本气泡），脚注已在文末完整列出 |
| Emoji `:code:` 输入与渲染 | 🆕 预览/导出渲染为 Unicode；原生编辑器输入 `:code:` 后随空格自动替换；源码保留 `:code:` 亦可在预览正确显示 |
| CJK 排版 | ✅ |

## 图片与媒体

| Typora 能力 | Corra 状态 |
|---|---|
| 插入图片 | ✅ 文件对话框；🆕 现在插入到光标处（原先是追加到文末） |
| 粘贴剪贴板图片 | 🆕 富文本模式 Cmd+V 直接粘贴（存 assets/ 或上传图床） |
| 拖放图片进编辑器 | 🆕 拖到窗口即插入当前备忘录（Markdown 文件拖入仍是打开） |
| 图片缩放（`<img width>` / `=300x200` / `=50%`） | 🆕 预览、PDF 导出、原生编辑器图示均生效 |
| 相对路径与 base path | ✅ SetBaseDir |
| 图床上传（Typora 用 iPic） | ✅ GitHub 图床 + jsDelivr CDN（等价方案） |
| 嵌入视频/iframe/网页 | 🔶 无 webview，HTML 块按源码卡片展示（安全且刻意） |

## 数学与图表

| Typora 能力 | Corra 状态 |
|---|---|
| 行内/块级数学公式实时排版 | ✅ go-latex 排版；🔶 TeX 上下标等复杂宏有限（go-latex 引擎子集，README 已知限制）；mhchem 化学方程式 ❌ 依赖 MathJax 扩展生态，纯 Go 无等价物 |
| 公式自动编号 | ❌ go-latex 无编号支持；编号引用属排版引擎层能力 |
| mermaid（流程图/时序图/甘特图等） | ✅（WASM 渲染器） |
| flowchart.js / sequence.js 独立渲染器 | ❌ 不引入 JS 引擎；同类图用 mermaid（flowchart/sequence 全覆盖）+ d2 + goat 替代 |
| 数据图表 | ✅ Corra 独有 ```chart |
| 图表错误提示 | ✅ 渲染失败显示源码卡片（等价 Error Hint） |

## 文件与库管理

| Typora 能力 | Corra 状态 |
|---|---|
| 文件树 + 文章列表侧栏 | ✅ 文件夹 + 备忘录列表（Apple Notes 式） |
| 全局搜索 | ✅ |
| 快速打开 | ✅ 搜索即快速打开 |
| 最近文件 | ✅ |
| 自动保存 | ✅ 600ms 防抖自动保存 |
| 版本恢复 | ❌ 纯文件存储，刻意交给 git / iCloud / Time Machine（README 说明的设计哲学） |
| 外部修改检测与冲突 | ✅ |

## 导入导出与打印

| Typora 能力 | Corra 状态 |
|---|---|
| 导出 PDF | ✅；🆕 现在嵌入正文图片并支持 `<div style="page-break-after">` 分页符；🔶 PDF 书签（outline）未做——gpdf 库暂无书签 API |
| 导出 HTML | 🆕 独立 HTML（内联样式 + KaTeX/mermaid CDN） |
| 导出 Word/EPUB/LaTeX/RTF/ODT/RST | 🆕 pandoc 集成（检测到 pandoc 才可用） |
| 导入 docx/EPUB/HTML/tex/odt/rst/org | 🆕 pandoc 集成 |
| 复制为 HTML / 复制 Markdown | 🆕 / ✅ |
| 智能粘贴（从网页粘贴 HTML → Markdown） | 🆕 富文本模式粘贴 HTML 自动转换（标题/列表/表格/代码/链接/图片） |
| 打印 Cmd+P | 🆕 macOS 原生打印面板（NSPrintOperation）；🔶 Windows/Linux 无打印 API（MyGo 限制），用导出 PDF 替代 |

## 外观

| Typora 能力 | Corra 状态 |
|---|---|
| 深/浅色主题 + 跟随系统 | ✅ |
| 自定义 CSS 主题（themes 生态） | 🔶 无 webview 无 CSS 引擎，不存在承载层；提供缩放（Cmd±）、深浅色、字体族与行高预设作为原生替代 |
| 缩放 | ✅ |
| 写作区宽度/行距自定义 | 🆕 部分（行高已有档位）|
| 标题自动编号（Typora 用 CSS 实现） | 🆕 渲染层实现，视图菜单开关 |
| RTL（从右到左，实验性） | 🔶 预览自绘文本栈不做 BiDi 塑形；原生编辑区由 AppKit 免费获得 RTL 输入与显示 |
| Touch Bar | ❌ Apple 已淘汰 Touch Bar 硬件，新代码不为已停产硬件投入 |

## 已知限制汇总（框架层）

1. **无 webview**：自定义 CSS 主题、iframe 嵌入、MathJax 扩展（mhchem 等）这三类能力在架构上不存在承载层。
2. **MyGo 未暴露的能力**：Windows/Linux 原生富文本编辑、非 macOS 打印、文本流内嵌图片（预览里的行内公式图）、PDF 书签（gpdf）。
3. **表格拖拽调宽/拖拽重排行列**：NSTextTable 自动布局不支持拖拽边框；行列增删经源码模态框完成。
