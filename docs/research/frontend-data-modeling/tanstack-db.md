# TanStack DB

类别：客户端数据库（normalized client store）

## 数据模型

TanStack DB 在客户端维护规范化的内存集合（collection）。集合可以包装已有的查询调用，比如 REST、tRPC、GraphQL 或 WebSocket 的 `useQuery`。后端的记录会变成有类型的集合，组件通过 live query 读取。

持久化是可选的，通过 IndexedDB、RxDB 等适配器实现。它本身不是持久存储，只存在内存里。

## 状态

项目仍处于 1.0 之前，0.x 版本之间 API 有过变化。

## 可信度

- 内存规范化集合、包装查询调用、live query：中。主要来自维护者的博客和官方文档，两篇博客属于推广性质。
- 具体 API（如包装 `useQuery` 的方式）：验证者只在搜索摘要中看到了 Query Collection 的行为，没有读到完整文档，因此结论较弱。
- 文档描述的是架构，没有性能基准。

## 对 corra 或 ArcMD 的启发

“查询结果进入规范化集合，视图通过 live query 订阅”，可以作为结构化数据读取的参考。但它假设数据来自服务端，对本地 Markdown 文档的编辑模型帮助不大。

## 来源

- https://tanstack.com/blog/tanstack-db-0.1-the-embedded-client-database-for-tanstack-query
- https://tanstack.com/blog/tanstack-db-1.0-rc1-going-all-in-on-oracles
- https://tanstack.com/db/latest/docs/overview
- https://tanstack.com/db/latest/docs/guides/live-queries
