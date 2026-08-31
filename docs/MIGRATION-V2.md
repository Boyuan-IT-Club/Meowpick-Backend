# 后端 v2 数据库迁移手册

> JDCloud `test-mongo` 的实际服务器命令统一维护在仓库根目录 `MEOWPICK-V2-改造与迁移说明.md`。本文保留通用迁移原理、冲突调查和数据库检查细节，不应单独作为 JDCloud 操作清单。
>
> 如果只想理解为什么要迁移、提案通过后映射如何保存，请先看 [提案映射与数据库迁移说明](./USER-PROPOSAL-CONSISTENCY.md)。本文是实际操作生产数据库时使用的详细手册。

## 先用 30 秒理解迁移

- 旧数据库不一定有错误，迁移不能被理解为“修复所有旧字段”。
- 迁移也不只是添加索引。它会把静态映射落到 MongoDB、建立新编号计数器、检查编号冲突、修复能明确判断的历史脏数据，并创建必要索引。
- 默认执行是试运行（`dry-run`）：只读取和生成 JSON 计划，不修改数据库。
- 只有无冲突、人工审核计划且已完成备份后，才执行 `--apply-plan migration-v2-plan.json` 正式修改数据库。
- 测试用旧 JSON 不能导入或覆盖线上数据库；线上后来新增的数据必须保留。

本次迁移将 MongoDB 设为校区、院系和课程分类映射的唯一真源，Redis 仅作为可重建缓存。禁止将旧 JSON 数据导入生产环境：该文件只是测试输入，不是生产数据库备份。

## 迁移内容

- 将 `mapping._id` 转换为 ObjectID。
- 为 `mapping` 增加 `canonical: bool`。历史上同名但不同编号的数据会保留全部 `code -> name` 别名；其中最小的历史静态编号作为 `name -> code` 的规范编号。新建映射均为规范映射。
- 使用唯一索引保护 `(type, code)` 以及规范记录的 `(type, name)`。
- 使用 `mapping_counter` 保存每种映射类型当前已分配的最大序号；下一次分配会在该值基础上递增。
- 将旧评论的 ObjectID 主键转换为十六进制字符串，使新接口返回的评论 ID 能直接用于点赞接口。
- 仅当能从来源提案唯一推导出正确编号时，才修复编号为 `0` 的课程字段；否则迁移会停止并报告课程 ID。
- 在可以从有效 ObjectID 时间戳恢复时，修复受 1970 年时间转换错误影响的教师时间字段。
- 为 `course.proposalId` 创建部分唯一索引，保证一条提案最多对应一门正式课程。
- 为非空 `user.username` 创建忽略大小写的部分唯一索引，防止并发创建重复昵称。
- 使用 `proposal_guard` 保存每日发布额度和相同课程防重所需的轻量串行化记录。该集合不保存权威业务数据，无需回填；停止活动 48 小时后记录会自动过期。

提案审批现在依赖 MongoDB 事务，因此生产 MongoDB 必须是副本集或分片集群。迁移预检会拒绝独立部署的 MongoDB。

## 必须遵循的执行顺序

1. 备份前进入全应用写维护窗口。从备份开始到试运行和应用阶段完成期间，停止所有应用发起的 MongoDB 写入。若试运行报告冲突，应用必须继续保持关闭，只允许获得批准的运维人员执行下文已经审核的修复写入；每次修复后必须废弃旧计划并重新执行试运行。
2. 确认待部署应用来自 `Eagle233` 分支对应版本。
3. 备份当前线上数据库。
4. 执行试运行并审核 JSON 报告。
5. 解决线上数据库中报告的每一项冲突，禁止通过编辑迁移计划来隐藏冲突。
6. 再次执行试运行，只有 `conflicts` 为空才能继续。
7. 应用这份未经修改的原始计划。工具会验证数据库快照自试运行后没有发生变化。
8. 重启后端。启动过程会原子重建全部六个 Redis 哈希键。
9. 执行下文验证，全部通过后才能重新开放写入。

## 备份

在运维终端中设置环境变量，禁止将数据库凭据写入仓库。

```bash
export MEOWPICK_MONGO_URI='mongodb://...'
export MEOWPICK_DB='meowpick'
export MEOWPICK_BACKUP="meowpick-before-v2-$(date +%Y%m%d-%H%M%S).archive.gz"
mongodump --uri "$MEOWPICK_MONGO_URI" --db "$MEOWPICK_DB" \
  --archive="$MEOWPICK_BACKUP" --gzip
shasum -a 256 "$MEOWPICK_BACKUP" > "$MEOWPICK_BACKUP.sha256"
mongosh "$MEOWPICK_MONGO_URI/$MEOWPICK_DB" --quiet --eval '
  for (const name of db.getCollectionNames().sort()) {
    print(name + " " + db.getCollection(name).countDocuments({}))
  }' > "$MEOWPICK_BACKUP.counts.txt"
```

记录备份文件校验和，并将备份、校验和及集合计数文件保存到应用服务器之外的位置。

## 强制试运行

```bash
go run ./cmd/migrate-v2 \
  --uri "$MEOWPICK_MONGO_URI" \
  --db "$MEOWPICK_DB" \
  --report migration-v2-plan.json
```

退出码 `0` 表示报告中没有冲突。退出码 `2` 表示没有进行任何数据库写入，必须解决报告 `conflicts` 数组中的问题。报告中不包含 MongoDB 凭据。

以下情况会阻止迁移，包括但不限于：

- 线上动态编号与名称不同的旧静态编号发生碰撞；
- 线上名称使用的编号与旧静态定义不一致；
- 课程包含编号 `0`，但无法从来源提案唯一确定正确编号；
- 同一 `proposalId` 对应多门正式课程；
- 评论 ObjectID 与已有字符串 ID 冲突；
- MongoDB 为独立部署而不是副本集或分片集群；
- 非空昵称在忽略大小写后重复；
- 任意用户的 `contributionPoints` 为负数。

## 冲突调查模板

使用迁移报告中的 ID 和数值执行以下只读命令：

```javascript
// mongosh，先切换到生产数据库
db.mapping.find({type: 1, code: 1})
db.course.find({$or: [{department: 0}, {category: 0}, {campuses: 0}]})
db.course.find({proposalId: {$type: "string", $gt: ""}})
  .sort({proposalId: 1})
db.proposal.findOne({_id: "PROPOSAL_ID"}, {course: 1})
db.user.aggregate([
  {$match: {username: {$type: "string", $gt: ""}}},
  {$group: {_id: {$toLower: {$trim: {input: "$username"}}}, ids: {$push: "$_id"}, count: {$sum: 1}}},
  {$match: {count: {$gt: 1}}}
])
db.user.find({contributionPoints: {$lt: 0}}, {_id: 1, contributionPoints: 1})
db.course.aggregate([
  {$match: {proposalId: {$type: "string", $gt: ""}}},
  {$group: {_id: "$proposalId", courseIds: {$push: "$_id"}, count: {$sum: 1}}},
  {$match: {count: {$gt: 1}}}
])
db.comment.find({_id: {$type: "objectId"}}, {_id: 1})
```

对于没有来源提案的零编号课程，运维人员必须根据权威业务资料确定正确的已有编号。显式修复示例如下；必须替换全部占位符，并保留旧值匹配条件：

```javascript
db.course.updateOne(
  {_id: "COURSE_ID", department: 0, category: 0},
  {$set: {department: NumberInt(DEPARTMENT_CODE), category: NumberInt(CATEGORY_CODE)}}
)
```

对于线上动态编号与静态编号的冲突，禁止只修改 `mapping.code`。必须先找到所有使用该歧义编号的课程和教师，判断每条记录实际对应的含义，然后同时迁移这些引用。如果无法根据权威数据作出判断，应保留该冲突并停止部署。

每次显式修复后，都必须废弃旧迁移计划并重新执行试运行。

昵称冲突必须由产品或运维负责人决定哪个用户保留昵称，禁止自动选择。负贡献值必须先审计该用户的审批和撤回日志。重复正式课程必须先选择权威课程并修复全部引用，之后才能删除或软删除重复课程。评论 ID 冲突必须比较两份评论记录及其点赞引用。每一项决定和修改前后值都必须记录在变更工单中；迁移工具只负责报告冲突，不会猜测修复方案。

## 应用迁移

```bash
go run ./cmd/migrate-v2 \
  --uri "$MEOWPICK_MONGO_URI" \
  --db "$MEOWPICK_DB" \
  --apply-plan migration-v2-plan.json
```

如果报告包含冲突、版本或数据库不匹配，或者数据库快照与试运行时不同，工具会拒绝应用。映射、计数器、评论 ID、课程和时间戳修改会在同一个 MongoDB 事务中完成。

## 迁移后验证

再次执行试运行。所有修复数量和冲突数量都必须为零：

```bash
go run ./cmd/migrate-v2 \
  --uri "$MEOWPICK_MONGO_URI" \
  --db "$MEOWPICK_DB" \
  --report migration-v2-postcheck.json
```

建议执行以下 `mongosh` 检查：

```javascript
db.mapping.aggregate([
  {$group: {_id: {type: "$type", code: "$code"}, count: {$sum: 1}}},
  {$match: {count: {$gt: 1}}}
])
db.mapping.countDocuments({canonical: true})
db.mapping_counter.find().sort({_id: 1})
db.course.countDocuments({$or: [{department: 0}, {category: 0}, {campuses: 0}]})
db.comment.countDocuments({_id: {$type: "objectId"}})
db.user.find({contributionPoints: {$lt: 0}}, {_id: 1, contributionPoints: 1})
db.user.aggregate([
  {$match: {username: {$type: "string", $gt: ""}}},
  {$group: {_id: {$toLower: {$trim: {input: "$username"}}}, ids: {$push: "$_id"}, count: {$sum: 1}}},
  {$match: {count: {$gt: 1}}}
])
```

所有用于检查重复项、零编号、ObjectID 评论、负贡献值或昵称冲突的查询都必须返回空结果或计数为零。对于每种映射类型，`mapping_counter.seq` 必须等于 `mapping.code` 的最大值；每个不同映射名称必须恰好有一条规范记录。以下只读查询会报告计数器不一致：

```javascript
db.mapping.aggregate([
  {$group: {_id: "$type", maxCode: {$max: "$code"}}},
  {$lookup: {from: "mapping_counter", localField: "_id", foreignField: "_id", as: "counter"}},
  {$match: {$expr: {$ne: ["$maxCode", {$first: "$counter.seq"}]}}}
])
```

后端启动后，Redis 必须包含以下六个哈希键：

```text
mapping:{reference-mappings}:1:name_to_code
mapping:{reference-mappings}:1:code_to_name
mapping:{reference-mappings}:2:name_to_code
mapping:{reference-mappings}:2:code_to_name
mapping:{reference-mappings}:3:name_to_code
mapping:{reference-mappings}:3:code_to_name
```

如果迁移报告中存在新增规范映射和历史别名，应分别选择一条作为样本；若不存在，则从每种映射类型选择任意规范记录，并选择任意可用的额外 `code -> name` 记录。使用 `HGET` 验证规范记录的 `name -> code` 和所有样本编号的 `code -> name`，每个值都必须与 MongoDB 一致。

随后通过一条测试提案，确认课程、映射、提案状态、贡献值和变更日志在同一事务中提交。任何 Redis 值缺失或不匹配，或者 MongoDB 出现部分写入，都表示部署失败；必须继续保持写入关闭并执行回滚。

## 回滚

如果验证失败，停止新后端并恢复迁移前备份。`--drop` 会替换集合，属于破坏性操作，执行前必须确认数据库名称和备份路径：

```bash
mongorestore --uri "$MEOWPICK_MONGO_URI" --db "$MEOWPICK_DB" \
  --archive="PATH_TO_VERIFIED_BACKUP.archive.gz" --gzip --drop
```

恢复前运行 `shasum -a 256 -c "$MEOWPICK_BACKUP.sha256"` 验证备份。恢复后，使用备份阶段相同的 `mongosh` 循环重新生成集合计数，只比较 `$MEOWPICK_BACKUP.counts.txt` 中已有的集合。迁移后新增的 `mapping_counter` 或 `proposal_guard` 可以保留，因为旧版本会忽略它们；这些额外集合不算计数不一致。启动旧版本前，还应抽样确认旧版本依赖的映射、课程、提案、用户、评论、教师和变更日志均可正常读取。

只能清空本应用独占的 Redis 逻辑库，禁止在共享 Redis 上使用 `FLUSHALL`。应从部署配置中确认 Redis 主机和逻辑库编号，由第二名运维人员复核后，再执行 `redis-cli -u "$MEOWPICK_REDIS_URI" -n REDIS_DB FLUSHDB`。随后启动旧版后端，验证用户资料以及提案列表和详情读取正常，最后才能重新开放写入。Redis 映射数据是可丢弃缓存，会根据已恢复的 MongoDB 自动重建。
