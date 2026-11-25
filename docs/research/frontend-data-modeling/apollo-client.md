# Apollo Client

类别：规范化缓存（normalized cache）

## 数据模型

Apollo Client 把 GraphQL 响应拆成扁平的对象表。每个带标识的对象按 `__typename` 加 `id`（或 `_id`）字段生成缓存 ID，格式为 `类型名:id`。所有规范化对象放在同一张查找表里，对象之间用引用连接，而不是嵌套。

没有 `id` 或 `_id` 的对象不会被规范化，会嵌在父对象里。

## 合并规则

同一缓存 ID 的新对象到达时，Apollo 按字段合并到已缓存的对象上，而不是整体替换。可以通过 `dataIdFromObject`、`keyFields` 或 `typePolicies` 自定义 ID；通过 `merge` 函数自定义某个字段的合并方式。

## 可信度

- 默认 ID 规则：高。来自当前官方文档和 v3 文档，两者一致。
- 合并行为：高。
- 来源中有一篇 2020 年的博客，但 v3 文档重复了同样的默认规则。

## 对 corra 或 ArcMD 的启发

按类型和 ID 扁平化存储、引用代替嵌套，这个思路可以用在结构化文档里（例如每个 block 用稳定 ID 引用）。Apollo 的字段级合并说明：更新应该是局部的，而不是整体替换。

## 来源

- https://www.apollographql.com/docs/react/caching/overview
- https://www.apollographql.com/docs/react/caching/cache-configuration
- https://www.apollographql.com/blog/demystifying-cache-normalization（2020 年）
