# Redux Toolkit

类别：集中式状态（single store）

## 数据模型

Redux Toolkit 把应用状态放在一个集中式 store 里，按 slice 组织，用 selector 读取和派生状态。这是官方推荐的模式。Redux 并不在技术上强制只有一个 store。

对关系型数据，官方建议规范化状态。`createEntityAdapter` 提供 `{ ids, entities }` 的结构：`ids` 保存顺序，`entities` 用 ID 索引对象。

## 可信度

- 单 store、slice 和 selector：中到高。官方文档 `three-principles` 页面支持这一点。
- 规范化建议和 `createEntityAdapter` 结构：中。官方的 normalizing-state 和 createEntityAdapter 文档支持结论。Better Stack 文章只有一条相关说明，验证者认为其内容单薄。
- 其中一条主张是 2-1 通过的，另一条 3-0。

## 对 corra 或 ArcMD 的启发

`{ ids, entities }` 是“顺序列表加按 ID 索引的表”的经典写法，可以用于 block 列表。它把顺序和内容分开存，移动、删除、替换都比较直接。

## 来源

- https://redux.js.org/understanding/thinking-in-redux/three-principles
- https://redux.js.org/usage/structuring-reducers/normalizing-state-shape
- https://redux.js.org/toolkit/api/createEntityAdapter
- https://betterstack.com/community/guides/scaling-nodejs/zustand-vs-redux-toolkit-vs-jotai/（补充来源）
