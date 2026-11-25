# 服务端数据与客户端状态的划分

类别：跨方案结论

## 结论

成熟的前端实践通常把状态按来源分开：

- **服务端派生的数据**放进查询库，如 TanStack Query 或 SWR，由它负责缓存、失效和重新获取。
- **客户端 UI 状态**放进通用状态库，如主题、当前用户、功能开关。

## 依据

- 一篇 2026 年 11 月的博客提出这个划分（3-0 通过）。
- TanStack Query 官方文档说明它不替代客户端状态管理，可以和 Redux 等库一起用。这一点佐证了划分。
- 工作流中“大部分全局状态都是服务端状态”的说法带有未注明来源的 20% 数字，已被剔除。

## 可信度

中。划分的思路有多个来源支持，但证据主要是观点性博客和官方文档的描述，没有独立的量化比较。

## 对 corra 或 ArcMD 的启发

本地 Markdown 没有服务端数据，但同样的划分可以用在编辑器里：文档内容（block 树）是一层，UI 状态（选区、面板、焦点）是另一层，两者不应混在一起。

## 来源

- https://trio.dev/7-top-react-state-management-libraries/
- https://tanstack.com/query/v5/docs/react/guides/does-this-replace-client-state
