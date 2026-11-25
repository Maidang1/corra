# Zustand

类别：集中式状态（single store）

## 数据模型

Zustand 把客户端状态组织成一个集中式 store，支持 slice 模式，用 selector 读取状态。文档把单 store 当作推荐做法，不是强制要求。Zustand 也支持多个 store。

## 可信度

- 单 store 推荐、slice 模式和 selector：中。Zustand 官方文档支持。
- “单 store 是推荐结构”这一表述：中。只有一篇观点性博客明确这样描述，验证结果为 2-1。

## 对 corra 或 ArcMD 的启发

Zustand 的 store 很轻，不强制规范化。它适合放 UI 状态（如当前选中的 block、面板是否展开），而不是承载文档内容本身。

## 来源

- https://zustand.docs.pmnd.rs/learn/guides/flux-inspired-practice
- https://zustand.docs.pmnd.rs/learn/guides/slices-pattern
- https://dev.to/locionic/zustand-vs-jotai-in-2026-the-practical-decision-tree-i-use-after-shipping-both-200e（观点性文章）
