# Relay

类别：规范化缓存（normalized cache）

## 数据模型

Relay 的客户端缓存是扁平的 `RecordSource`，把 data ID 映射到记录。每条记录保存：

- 自己的 data ID
- GraphQL 类型
- 已经取回的字段（只存取到的部分）

关联对象单独存成记录，父记录通过 `__ref` 引用它们，不嵌套。例如地址字段在父记录中表现为 `address: {__ref: 'client:842472:address'}`。

没有 `id` 字段的对象使用基于路径的 data ID。这是 ID 生成的细节，不影响整体模型。

## 可信度

- 扁平 RecordSource、记录结构和字段级存储：高，来自 Relay 官方架构文档，3-0 通过。
- `__ref` 引用的结构：中。原文引语没有逐字找到，但验证者通过文档中的规范化示例确认了同样的结构。

## 对 corra 或 ArcMD 的启发

“记录只存已知字段、关联用引用”的做法，和 ArcMD 的 block 按 ID 引用的思路接近。区别在于 Relay 的记录是按需部分填充的，而 ArcMD 的 block 是完整的结构化数据。

## 来源

- https://relay.dev/docs/principles-and-architecture/runtime-architecture/
- https://relay.dev/docs/api-reference/store/
