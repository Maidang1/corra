# Jotai

类别：原子化状态（atomic state）

## 数据模型

Jotai 把状态拆成许多小的独立 atom，再组合起来，而不是放进一个大 store。派生 atom 读取其他 atom，所以 atom 之间是组合关系，并不完全独立。

## 可信度

- atom 模型和 atom 组合：中。官方文档支持（2-1 通过）。
- 来自 Better Stack 和 dev.to 的说法是转述，不是原文引语。
- “组件只在所读的具体 atom 变化时重渲染”这一说法被 0-3 反驳，没有写入结论。

## 对 corra 或 ArcMD 的启发

按块拆分的 atom 很适合做按 block 订阅：每个 block 一个 atom，只有它变化时相关视图才更新。这和 ArcMD 的稳定 block ID 配合起来，思路相通。

## 来源

- https://jotai.pmnd.rs/docs/core/atom
- https://jotai.pmnd.rs/docs/guides/composing-atoms
- https://jotai.pmnd.rs/docs/basics/comparison
