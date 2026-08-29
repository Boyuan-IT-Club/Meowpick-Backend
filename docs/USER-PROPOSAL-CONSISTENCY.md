# 用户资料与提案一致性说明

本文档记录用户资料和提案修复后的运行逻辑、数据一致性保证及生产上线检查流程。MongoDB 始终是业务数据的唯一真源，Redis 和 `monc` 对象缓存均为可丢弃、可重建的读取缓存。

## 已修复的接口行为

- `GET /api/proposal/list` 现在会从查询参数绑定 `status`。普通用户始终只能获得已通过提案；管理员可以指定一种状态，也可以不传状态以查询全部提案。
- `GET /api/proposal/filter` 允许省略 `status` 和 `campus`。普通用户仍会被强制限制为已通过提案。校区合法性使用 MongoDB/Redis 中的运行时映射校验，不再依赖旧的编译期静态映射表。
- 创建提案、更新提案和通过提案时，会拒绝空白标题、空白课程名、空白院系、空白课程分类、空校区列表、格式错误的教师信息以及未知校区。管理员传入的最终课程也会在审批时重新校验。
- `POST /api/proposal/:proposalId/delete` 支持空请求体，因为提案 ID 已由路径参数提供。
- `GET /api/user/:userId/username` 允许用户查询自己，也允许管理员查询任意用户。普通用户跨用户查询时必须提供 `proposalId`，且该提案必须属于目标用户并满足 `showUsername=true`。

本次涉及的完整路由如下：

- `GET /api/user/profile`、`POST /api/user/profile/update`、`GET /api/user/:userId/username`；
- `POST /api/proposal/add`、`GET /api/proposal/list`、`GET /api/proposal/filter`、`POST /api/proposal/:proposalId/update`、`POST /api/proposal/:proposalId/delete`；
- `POST /api/proposal/:proposalId/approve`、`POST /api/proposal/:proposalId/reject`、`POST /api/proposal/:proposalId/revoke`。

## 一致性保证

- 通过提案和撤回通过操作会在 MongoDB 事务中执行课程、映射、教师、贡献值、提案状态和审计日志写入。两个并发撤回请求不能重复扣减贡献值，贡献值也不能变成负数。
- 更新、删除、通过、拒绝和撤回均使用“预期状态”作为数据库更新条件。若另一个请求已先完成状态变更，后到请求会返回提案更新失败，不会使用旧状态覆盖新数据。
- 创建提案时使用两条 `proposal_guard` 协调记录：一条对应用户和 UTC+8 自然日，另一条对应标准化后的课程指纹。事务内更新这些记录可以串行化每日额度检查和重复课程检查。协调记录只保存 ID、版本号和时间戳，不保存权威业务数据。
- 昵称更新会将读取到的旧 `usernameUpdatedAt` 加入 MongoDB 更新条件，因此两个并发请求不能同时绕过 30 天修改限制。
- MongoDB 事务内的用户读取会绕过 `monc` 缓存。贡献值写入仅在事务成功提交后失效用户对象缓存，因此 `/api/user/profile` 能立即读到通过或撤回后的最新贡献值。

## 数据库变化

本次修复不会替换现有业务集合，也不会重命名已有字段。新版本会创建或使用以下对象：

- `user.idx_user_username_unique`：针对非空 `username` 的忽略大小写、唯一、部分索引；
- `proposal_guard`：非权威的并发协调集合，使用默认唯一 `_id` 索引。TTL 索引会在协调记录停止活动 48 小时后自动删除；删除协调记录是安全的，因为每个事务仍会以 MongoDB 业务集合为真源重新检查数据。

上线前必须先以试运行（`dry-run`）模式运行 `cmd/migrate-v2`。如果旧昵称存在忽略大小写后的冲突，或者已有用户贡献值为负数，工具会停止并生成冲突报告。必须解决全部冲突并重新得到无冲突的试运行报告后，才能启动新后端。备份、应用、验证和回滚命令见 `docs/MIGRATION-V2.md`。

## 上线验证

1. 备份 MongoDB，并执行强制迁移试运行。
2. 按 `docs/MIGRATION-V2.md` 的要求，在全应用写维护窗口内应用未经修改的原始迁移计划。
3. 启动后端，确认 MongoDB 为副本集或分片集群。
4. 使用 `POST /api/proposal/add` 创建一条待审核提案，然后在不传 `status` 和 `campus` 的情况下分别调用 `GET /api/proposal/list` 与 `GET /api/proposal/filter`；普通用户的响应中不得出现该待审核提案。
5. 通过一条测试提案，随后立即调用 `/api/user/profile`；无需重启后端或清空 Redis，就必须能看到新的贡献值。
6. 同时发送两个撤回请求，只允许一个成功；课程状态、提案贡献值、用户贡献值和提案状态必须相互一致。
7. 验证普通用户不能通过隐藏昵称的提案查询提案作者昵称。

## 回滚

仅回滚应用版本时不需要删除 `proposal_guard`，旧版本会忽略该集合。如果还需要回滚数据库迁移，应按照 `docs/MIGRATION-V2.md` 恢复已验证的迁移前备份，只清理本应用使用的 Redis 逻辑库，然后启动旧版后端。任何生产冲突都不得通过直接覆盖动态映射编号或用户昵称来处理，必须先由运维和业务负责人审核冲突处置方案。
