# v2 离线 BSON 迁移设计

本文供 Agents、开发和运维维护迁移逻辑。业务行为的简明说明见 [提案映射与数据库迁移说明](./USER-PROPOSAL-CONSISTENCY.md)。

## 决策

- 不在测试或生产 MongoDB 上直接运行迁移工具。
- 使用生产 `mongodump` BSON/archive 作为输入，在 Mac/运维机的隔离 Docker 环境中处理。
- 输入备份只读且永不覆盖；输出是新的 `mongodump` BSON/archive。
- `dry-run` 和 `apply` 必须分阶段。apply 使用相同临时数据卷、相同迁移镜像和未经修改的计划。
- 迁移发现动态/静态编号、昵称、重复课程等冲突时停止，不能自动选择或覆盖。
- BSON 迁移不负责修改后端、Redis、Kubernetes 资源、网络、Secret、PVC 或 MongoDB 副本集拓扑。

## 本地编排

入口为 `scripts/process-mongodb-bson-v2.sh`：

1. `dry-run` 校验输入，构建 `Dockerfile.migrate-v2`，创建隔离网络和不映射端口的 MongoDB 7.0.23 容器。
2. 临时 MongoDB 初始化为单节点 `rs0`，以满足迁移事务要求。
3. 输入目录以只读方式挂载；支持 mongodump 目录、`.archive` 和 `.archive.gz`。
4. `cmd/migrate-v2` 生成带计划哈希和数据库快照哈希的报告。冲突退出码为 2。
5. `apply` 需要固定确认词，工具再次计算数据库快照并核验计划哈希。
6. 事务提交后重新 dry-run。仍有冲突或确定性待修复项时拒绝导出。
7. 输出由 `mongodump` 生成，执行 gzip、SHA-256 和 `mongorestore --dryRun` 检查。
8. `cleanup` 只删除状态文件记录且名称满足固定格式的临时容器、网络和数据卷。

## 迁移内容

- 合并编译期历史映射与备份中已有动态映射，写入 `mapping`。
- 保留所有历史 `code → name`；同名历史静态编号中最小编号作为 canonical `name → code`。
- 初始化三种映射类型的 `mapping_counter` 最大编号。
- 只在能从来源提案唯一推导时修复课程零编号。
- 把旧评论 ObjectID 主键转换为等值十六进制字符串。
- 能从有效 ObjectID 恢复时修复教师 1970 时间。
- 将能够由唯一目标确定类型的旧点赞规范化为字符串 ID 和整数 `targetType`；删除目标不存在或目标已删除的悬空点赞。dry-run 会逐条列出操作和原因，目标同时匹配多种类型时停止并报告冲突。
- 创建 `mapping` 编号唯一索引、canonical 名称唯一索引、`course.proposalId` 部分唯一索引和昵称忽略大小写部分唯一索引。
- `proposal_guard` 没有历史数据，不需要迁移；集合及 TTL 索引由新版后端启动时创建。

教师 `department=null/0` 不在迁移范围内。当前业务允许教师暂未维护所属院系，而且课程开课院系不等于教师所属院系，迁移工具不得据此猜测或批量填充。

## 阻断条件

- MongoDB 不是副本集或分片集群；
- 线上动态编号占用了名称不同的静态编号；
- 已有动态名称使用了不允许的静态编号；
- 同名动态映射对应多个编号；
- 课程有零编号但没有可追溯提案，或提案包含未知校区；
- 一份提案对应多门正式课程；
- 评论 ObjectID 转字符串后与现有 ID 相撞；
- 非空昵称忽略大小写后重复；
- 用户贡献值为负。

冲突处理必须依据权威业务记录，并在另一个工作副本上执行。禁止编辑计划文件隐藏冲突；处理后必须重新创建工作目录并从 dry-run 开始。

## BSON 恢复边界

`mongodump` 保存文档 BSON 类型和索引元数据，但不保存部署拓扑。恢复 archive 不会改变 Service、认证、网络或副本集。

建议先用 `--nsFrom='meowpick.*' --nsTo='meowpick_v2.*'` 恢复到新库验收。若必须完全覆盖原库，先停写并做最终备份，再明确 `dropDatabase()` 后恢复。仅使用 `mongorestore --drop` 不能删除备份中不存在的旧集合。

## 2026-09-03 本地验证记录

- MongoDB 镜像：`mongo:7.0.23-jammy`；
- 从当前测试数据库导出的 BSON 共恢复 203,589 条文档；
- 在隔离副本中清除已确认的测试数据和昵称字段后，dry-run 为 0 冲突、0 警告；
- apply 写入 640 条映射、3 条计数器，转换 5,530 个评论 ID、修复 18 个教师时间，并修复 1 门可从提案唯一推导映射的课程；
- postcheck 为 0 冲突、0 评论修复、0 教师修复、0 课程修复；
- 新 archive 已完整恢复到第二个全新 MongoDB 容器，共 204,209 条文档；
- 关键三个业务唯一索引从 archive 正常恢复；评论 ObjectID 和 1970 教师时间均为 0；
- 三份输入/输出 archive 的 SHA-256 不同且原始输入校验值在前后保持不变。
