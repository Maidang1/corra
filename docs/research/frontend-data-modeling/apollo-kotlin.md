# Apollo Kotlin

类别：规范化缓存（normalized cache）

## 数据模型

Apollo Kotlin 的规范化缓存与 Apollo Client 类似，但默认 ID 不同。没有声明 ID 时，对象的缓存 ID 默认是它在 GraphQL 响应中的字段路径，这个 ID 在去重上效果不好。

团队可以在 schema 中用指令声明 ID（`@typePolicy` 配合 `keyFields`）。声明式 ID 会带上类型名前缀。

## 注意点

- 字段路径式 ID 不适合去重。
- 只用裸 `id` 作为 ID 时，它必须在所有对象间唯一。带类型名前缀后，只有在同一类型内 `id` 唯一时才全局唯一。
- 本次没有核对 Kotlin v2 到 v4 之间默认 key 格式的具体差异。

## 可信度

- 字段路径默认 ID、声明式 ID 和类型名前缀：高，来自官方文档，两条主张都是 3-0 通过。

## 对 corra 或 ArcMD 的启发

声明式的 ID 规则比隐式默认更好维护。对文档结构来说，应该让稳定 ID 由模型显式给出，不依赖位置。

## 来源

- https://www.apollographql.com/docs/kotlin/caching/normalized-cache
- https://www.apollographql.com/docs/kotlin/caching/declarative-ids
