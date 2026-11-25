# 前端数据建模方案调研

本目录每个文档对应一个前端状态或缓存方案，说明它如何组织数据、标识对象、响应变化。

## 调研范围与方法

- 来源：一次 deep-research 工作流（5 个搜索角度、抓取约 15 个来源、每条主张 3 票对抗验证，2/3 反驳则剔除）。最终保留 9 条主张。
- 证据主要是各项目官方文档和维护者博客。少数来源是第三方文章，已在对应文档中标明。
- 工作流中关于性能和重渲染行为的比较，要么被反驳，要么无法验证，因此没有写入这些文档。
- 本目录的内容是调研记录，我没有逐一亲自打开每个来源核对原文。引用的版本和日期以工作流记录为准。

## 范围缺口

本次调研聚焦的是**客户端数据缓存和状态管理库**（Apollo、Relay、TanStack DB、TinyBase、Redux Toolkit、Zustand、Jotai）。
它不包括**富文本编辑器**的文档模型（ProseMirror、Lexical、Slate、Tiptap、Milkdown 等）。如果需要后者，要另外调研。

## 文档索引

| 文档 | 方案 | 类别 |
| --- | --- | --- |
| [apollo-client.md](apollo-client.md) | Apollo Client | 规范化缓存 |
| [apollo-kotlin.md](apollo-kotlin.md) | Apollo Kotlin | 规范化缓存 |
| [relay.md](relay.md) | Relay | 规范化缓存 |
| [tanstack-db.md](tanstack-db.md) | TanStack DB | 客户端数据库 |
| [tinybase.md](tinybase.md) | TinyBase | 客户端数据库 |
| [redux-toolkit.md](redux-toolkit.md) | Redux Toolkit | 集中式状态 |
| [zustand.md](zustand.md) | Zustand | 集中式状态 |
| [jotai.md](jotai.md) | Jotai | 原子化状态 |
| [state-split.md](state-split.md) | 服务端数据与客户端状态的划分 | 跨方案结论 |

## 与 corra、ArcMD 的关系

这些方案解决的是“接口数据如何缓存和被组件订阅”，和 Markdown 文档的结构化编辑不是同一个问题。它们的启发主要在两点：规范化存储（按类型和 ID 扁平化）和“数据层与 UI 状态分开”。
