# 用户资料与提案接口修复明细

本文档记录用户资料和提案接口修复后的行为与一致性保证。映射存储和迁移的简明解释见 [提案映射与数据库迁移说明](./USER-PROPOSAL-CONSISTENCY.md)。

## 已修复的接口行为

- `GET /api/proposal/list` 会从查询参数绑定 `status`。普通用户始终只能获得已通过提案；管理员可以指定一种状态，也可以不传状态查询全部提案。
- `GET /api/proposal/filter` 允许省略 `status` 和 `campus`。普通用户仍被强制限制为已通过提案。校区合法性使用 MongoDB/Redis 中的运行时映射校验。
- 创建、更新和通过提案时，会拒绝空白标题、空白课程名、空白院系、空白课程分类、空校区列表、格式错误的教师信息和未知校区。管理员传入的最终课程会在审批时重新校验。
- `POST /api/proposal/:proposalId/delete` 支持空请求体，因为提案 ID 已由路径参数提供。
- `GET /api/user/:userId/username` 允许用户查询自己，也允许管理员查询任意用户。普通用户跨用户查询时必须提供 `proposalId`，且提案必须属于目标用户并满足 `showUsername=true`。
- 用户资料的局部更新先直接更新 MongoDB，再删除完整用户对象缓存。只更新头像不会再把缓存中的昵称覆盖为空；管理员状态的局部更新采用相同策略。
- 撤回“拒绝提案”时，状态恢复为 `pending`，并同步清空已经失效的 `rejectReason`。
- 点赞只接受 `proposal` 和 `comment`，且写入前必须确认目标存在且未删除。非法类型和悬空目标不会产生点赞记录。
- 课程详情、课程搜索、教师/分类/院系搜索和课程建议统一排除 `deleted=true` 的课程。撤回已通过提案后，关联课程不会继续出现在搜索结果中。
- 撤回已通过提案时，会在同一个 MongoDB 事务中软删除关联课程和该课程的全部评论，并删除这些评论的点赞，避免产生悬空点赞。
- 由提案创建的正式课程在详情和搜索结果中返回 `contributor`。其中包含来源 `proposalId` 和 `showUsername`；只有提案允许展示昵称时才返回 `userId`、`username`，否则不暴露作者身份。
- 请求参数校验错误返回统一业务错误，不向调用方暴露 Gin validator、JSON 解码器或其他内部错误文本。

涉及的路由包括：

- `GET /api/user/profile`、`POST /api/user/profile/update`、`GET /api/user/:userId/username`；
- `POST /api/proposal/add`、`GET /api/proposal/list`、`GET /api/proposal/filter`、`POST /api/proposal/:proposalId/update`、`POST /api/proposal/:proposalId/delete`；
- `POST /api/proposal/:proposalId/approve`、`POST /api/proposal/:proposalId/reject`、`POST /api/proposal/:proposalId/revoke`。

## 一致性保证

- 通过和撤回提案会在 MongoDB 事务中执行课程、评论、评论点赞、映射、教师、贡献值、提案状态和审计日志写入。两个并发撤回请求不能重复扣减贡献值，贡献值也不能变成负数。
- 更新、删除、通过、拒绝和撤回均使用预期状态作为数据库更新条件。若另一个请求已先完成状态变更，后到请求会失败，不会用旧状态覆盖新数据。
- 创建提案时使用两条 `proposal_guard` 协调记录，分别对应用户每日额度和标准化课程指纹。协调记录只保存 ID、版本号和时间戳，不保存权威业务数据。
- 昵称更新会把读取到的旧 `usernameUpdatedAt` 加入更新条件，因此两个并发请求不能同时绕过 30 天修改限制。
- MongoDB 事务中的用户读取会绕过 `monc` 缓存。贡献值仅在事务成功提交后失效用户对象缓存，因此 `/api/user/profile` 能立即读到最新贡献值。
- 后端日志不记录请求体、响应体、验证码或访问令牌。提案事务使用 MongoDB 原生 session，避免 `monc` 把含凭据的 MongoDB URI 写入事务失败日志。

## 教师院系说明

旧教师记录中 `department` 为 `null` 或 `0` 表示当前没有维护教师所属院系，这是允许的数据状态，不属于本次迁移错误。迁移工具不会用课程开课院系猜测教师所属院系；接口在此情况下继续返回“未知开课院系”。

## 数据库变化

- `user.idx_user_username_unique`：针对非空 `username` 的忽略大小写、唯一、部分索引；
- `proposal_guard`：非权威并发协调集合，使用默认唯一 `_id` 索引；TTL 索引会在记录停止活动 48 小时后自动删除。
- `comment.idx_comment_course_id_deleted_created_at`：支持按课程查询和撤销审批时批量软删除评论。新版后端启动时会幂等创建，不需要手工执行迁移脚本。
- 新版后端只保证部署后的撤销操作联动删除评论。上线前仍应 dry-run 检查历史上“课程已删除或不存在、评论仍未删除”的记录；若存在，应由迁移工具同时软删除评论并删除对应点赞，不能在应用启动时静默修改历史业务数据。

上线前必须先运行迁移试运行。如果存在昵称大小写冲突、负贡献值或其他阻塞问题，工具会停止并生成报告。完整步骤见 [MIGRATION-V2.md](./MIGRATION-V2.md)。

## 接口上线验证

1. 创建一条待审核提案，不传 `status` 和 `campus` 分别调用提案列表和筛选接口，普通用户响应中不得出现待审核提案。
2. 通过测试提案后立即调用 `/api/user/profile`，无需重启或清空 Redis 就应看到新的贡献值。
3. 同时发送两个撤回请求，只允许一个成功；课程、提案、用户贡献值和提案状态必须一致。
4. 验证普通用户不能通过隐藏昵称的提案查询作者昵称。
5. 通过提案后确认课程搜索结果包含 `contributor`；`showUsername=false` 时不得返回作者 ID 和昵称。
6. 为新课程创建评论后撤销审批，确认课程搜索结果为 0、评论查询结果为 0、用户贡献值已回扣且本人“我的提案”仍能看到待审核提案。

## 回滚

只回滚应用版本时无需删除 `proposal_guard`，旧版本会忽略该集合。如果还需回滚数据库迁移，应恢复已验证的迁移前备份，只清理本应用使用的 Redis 逻辑库，再启动旧后端。生产冲突不得通过覆盖动态映射编号或昵称处理，必须先审核处置方案。
