# TinyBase

类别：客户端数据库（本地 store）

## 数据模型

TinyBase 的一个 store 同时包含两类数据：

- 结构化的表（tables），行列组织
- 键值数据（values）

它提供细粒度的响应式：监听器（listeners）和响应式查询引擎。它还有可选的 schema，写入不符合 schema 的值时会被修正。持久化内置在库中。

## 可信度

- 数据模型、响应式、schema 和持久化的组成：中。3-0 通过，文档覆盖了 Store API、schema 和持久化指南。
- 来源全部是 TinyBase 官方文档，没有独立评价。
- 搜索结果没有标注发布日期，因此文档的时效性没有确认。

## 对 corra 或 ArcMD 的启发

“表加键值放在一个 store 里，schema 负责约束写入”的组合，对结构化文档的属性（如 frontmatter）可能有参考价值。它的表模型是行列式的，和文档树形结构不同。

## 来源

- https://tinybase.org/guides/the-basics/why-tinybase/
- https://tinybase.org/api/store/interfaces/store/store/
- https://tinybase.org/guides/schemas/using-schemas/
- https://tinybase.org/guides/schemas-and-persistence/database-persistence/
- https://github.com/tinyplex/tinybase/blob/main/readme.md
