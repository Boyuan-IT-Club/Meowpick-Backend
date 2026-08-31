# Meowpick 后端 v2 改造与迁移说明

> 这是给项目负责人阅读的唯一主文档。`docs/` 目录中的其他 Markdown 是给 Agents、开发和运维排查细节使用的。
>
> 如果只想理解并手写“这次改了什么”，阅读第一至第五部分和第九部分；真正上线时再阅读第六至第八部分。

## 一、先说最终结论

这次改造解决的核心问题是：原来校区、院系、课程分类主要依赖后端代码里的静态映射，无法可靠支持运行期间新增数据；现在改为 MongoDB 永久保存、Redis 加速查询。

- 院系可以在提案审批通过时新增；
- 课程分类可以在提案审批通过时新增；
- 校区不能通过提案新增，只能选择已经存在的校区；
- 未知校区会在创建、修改或审批提案时被拒绝；
- 新院系和新分类写入 MongoDB 后，不会因为 Redis 被清空或后端重启而丢失；
- 提案审批使用 MongoDB 事务，院系、分类、课程、教师、贡献值和提案状态要么全部成功，要么全部回滚；
- MongoDB 是权威数据来源，Redis 和 `monc` 都只是缓存。

## 二、修改前存在什么问题

### 1. 映射主要写死在后端代码中

旧版本把校区、院系和课程分类保存在 Go 代码的映射表里。查询速度快，但运行期间新增院系或分类后，很难可靠保存；服务重启后也可能重新回到代码中原有的映射。

### 2. MongoDB 中的旧映射结构不适合动态新增

测试发现旧数据可能存在以下问题：

- 多条映射使用错误或旧格式的 `_id`；
- 动态编号可能和代码里的静态编号冲突；
- 映射创建失败后，课程可能留下编号 `0`；
- 没有专门的编号计数器，多实例同时新增时可能产生重复编号；
- 缺少阻止重复名称、重复编号的唯一索引。

这不代表线上数据库的所有字段都有错误。迁移工具会先检查，只有确实存在的问题才会报告或修复。

### 3. 提案审批可能产生不一致

旧逻辑中的映射、课程、教师、提案状态和贡献值不是一个完整原子操作。中间步骤失败时，理论上可能出现一部分成功、一部分失败。

### 4. 用户资料和提案接口存在边界问题

本次检查还发现并修复了以下问题：

- 用户贡献值更新后，`monc` 对象缓存可能继续返回旧值；
- 两个并发撤回请求可能重复扣减贡献值；
- 普通用户可能通过旧列表接口看到未通过提案；
- 提案筛选接口把本应可选的条件当成必填；
- 删除提案时，空请求体可能错误返回 `EOF`；
- 部分空白课程字段能够进入数据库；
- 隐藏昵称的提案可能被绕过隐私限制查询作者昵称；
- 评论 ID、教师时间和部分路由文档存在历史问题。

## 三、修改后是怎么工作的

### 1. MongoDB、Redis 和 monc 分别负责什么

| 组件 | 作用 | 数据丢失后的结果 |
| --- | --- | --- |
| MongoDB | 保存用户、提案、课程、教师和映射，是唯一真源 | 不能随意丢失，必须备份 |
| Redis | 保存共享查询缓存，例如映射的名称和编号互查 | 可以清空，后端会从 MongoDB 重建 |
| `monc` | 后端进程内的 MongoDB 对象读取缓存 | 可以失效或重建，它不是 Redis |

Redis 和 `monc` 没有上下级关系。Redis 是独立服务，多个后端实例可以共享；`monc` 是每个后端进程内部的对象缓存。

### 2. 校区、院系和课程分类如何查询

Redis 为每种映射保存两个 Hash：

- `name -> code`：名称转换为编号；
- `code -> name`：编号转换为名称。

查询顺序是：先查 Redis，未命中时查询 MongoDB，再把结果回填 Redis。批量组装课程时使用 `HMGET` 一次读取多个编号，避免产生大量 Redis 请求。

### 3. 提案通过后如何新增院系和分类

```text
管理员提交通过操作
        ↓
校验所有校区必须已经存在
        ↓
开启 MongoDB 事务
        ↓
创建缺少的院系、课程分类和教师
        ↓
写入或恢复课程
        ↓
更新提案状态、贡献值和审计日志
        ↓
提交事务
        ↓
从 MongoDB 刷新 Redis 映射
```

如果事务中的任何一步失败，全部数据库写入都会回滚。MongoDB 已经提交、但 Redis 刷新暂时失败时，MongoDB 中的数据仍然安全；缓存未命中或服务重启后会重新加载。

### 4. 后端重启时如何预热 Redis

后端启动时从 MongoDB 读取全部映射，先写入临时 Redis Hash。六个 Hash 全部构建完成后，再一次性切换为正式键，因此其他实例不会读到只加载了一半的数据。

## 四、数据库增加或调整了什么

### `mapping`

永久保存校区、院系和课程分类。`type` 区分类型，`code` 保存编号，`name` 保存名称，`canonical` 标记同名映射中的规范编号。

### `mapping_counter`

保存每种映射已经分配到的最大编号。新增院系或分类时通过 MongoDB 原子递增获得新编号，避免多个实例分配相同编号。

### `proposal_guard`

协调每日提案额度和相同课程防重复的并发请求。它不是业务真源，停止活动 48 小时后可以自动过期。

### 新增或加强的索引

- `mapping` 的类型加编号唯一索引；
- `mapping` 的规范名称唯一索引；
- `course.proposalId` 的部分唯一索引；
- 非空用户昵称的忽略大小写部分唯一索引；
- `proposal_guard` 的默认唯一主键和 TTL 索引。

## 五、迁移工具到底做什么

迁移工具不是把旧 JSON 导入服务器，也不只是添加索引。它会：

1. 把代码中的静态映射和线上已有映射合并为 MongoDB `mapping` 数据；
2. 建立 `mapping_counter`；
3. 检查线上动态编号与旧静态编号是否冲突；
4. 修复能够从权威数据唯一判断的课程零编号、评论 ID 和教师时间问题；
5. 检查重复课程、昵称大小写冲突和负贡献值；
6. 创建必要的唯一索引；
7. 发现无法确定的冲突时停止，不猜测、不覆盖。

`dry-run` 是试运行：只读取数据库、生成 JSON 计划和冲突报告，不修改数据库。

`apply-plan` 是正式迁移：读取已经审核的试运行计划，确认数据库从试运行后没有变化，再通过事务写入数据库。

## 六、已确认的 JDCloud 环境

以下信息已通过只读 SSH 检查确认：

| 项目 | 当前值 |
| --- | --- |
| SSH 别名 | `Eagle233-JDCloud` |
| 后端容器 | `test-meowpick-backend` |
| 后端配置 | `/home/eagle233/repos/test/meowpick/config.yaml` |
| MongoDB 容器 | `test-mongo` |
| MongoDB 数据库 | `meowpick` |
| Docker 网络 | `test-net` |
| MongoDB 镜像 | `mongo:7.0.23-jammy` |
| MongoDB 宿主机端口 | `27015` |
| MongoDB 数据目录 | `/home/eagle233/repos/test/mongo/data` |
| MongoDB 配置目录 | `/home/eagle233/repos/test/mongo/config` |
| Portainer Compose | `/home/eagle233/portainer/srv/compose/1/v1/docker-compose.yml` |
| 后端镜像 | `boyuanclub/meowpick-backend:latest` |
| 服务器 Go 环境 | 未安装，迁移脚本通过 Docker 构建和运行 |

当前 `test-mongo` 是单机模式。MongoDB 事务要求副本集，所以必须先转换成单节点副本集 `rs0`。转换会重启 MongoDB，因此脚本强制先停止后端并完成备份。

转换和回滚都会短暂重启整个 `test-mongo` 容器。如果还有其他程序使用这个容器中的其他数据库，也会受到短暂停机影响；正式操作前需要一并确认维护窗口。

本轮没有在服务器执行迁移、备份、停止容器、修改配置、创建副本集或写数据库。此前 SSH 操作全部是只读检查。

## 七、迁移脚本怎么使用

实际脚本是 `scripts/jdcloud-migrate-v2.sh`。它会校验 JDCloud 主机名、机器指纹、本机 Docker socket、Compose 标签、MongoDB 镜像、数据挂载、端口、网络、URI 和数据库，并拒绝重复的 `Mongo.URL`/`Mongo.DB` 配置。迁移程序还会用 Go 实际解析出的连接结果再次强制核对 host、port 和 DB，固定拒绝除当前 JDCloud `test-mongo:27017/meowpick` 之外的目标；每个阶段失败都会立即退出。

### 1. 脚本提供的阶段

| 命令 | 是否修改服务器 | 作用 |
| --- | --- | --- |
| `preflight` | 否 | 只读检查容器、网络、配置和 MongoDB 拓扑 |
| `init` | 只创建权限为 `700` 的迁移目录 | 建立本次操作记录目录 |
| `backup` | 会停止后端并写备份文件 | 备份数据库、配置和旧镜像 ID，并验证备份可恢复性 |
| `prepare-replica` | 会修改配置并重启 MongoDB | 原子生成 keyFile，把 `test-mongo` 转为单节点 `rs0` |
| `build` | 会构建本地 Docker 镜像 | 用同一个 Git commit 构建同时包含迁移工具和新后端的固定镜像 |
| `dry-run` | 只读数据库 | 生成迁移计划；有冲突时退出码为 `2` |
| `apply` | 会写 MongoDB | 应用已经审核且数据库快照未变化的计划 |
| `postcheck` | 只读数据库 | 检查迁移后仍有无冲突或待修复数据 |
| `deploy` | 会重建后端容器 | 部署与迁移工具完全相同 commit 的后端镜像 |
| `rollback` | 会删除并恢复 `meowpick` 数据库 | 从迁移前备份恢复数据库、配置和旧后端镜像 |

### 2. 进入服务器并取得脚本

在你的电脑上执行：

```bash
ssh Eagle233-JDCloud
```

进入服务器后，第一次执行：

```bash
mkdir -p /home/eagle233/repos/tools
git clone --branch Eagle233 --single-branch https://github.com/Boyuan-IT-Club/Meowpick-Backend.git /home/eagle233/repos/tools/Meowpick-Backend-Eagle233
cd /home/eagle233/repos/tools/Meowpick-Backend-Eagle233
```

以后重新执行迁移前，更新并确认工作区干净：

```bash
git fetch origin Eagle233
git checkout Eagle233
git pull --ff-only origin Eagle233
git status --short
```

`git status --short` 必须没有输出。

### 3. 只读预检

```bash
bash scripts/jdcloud-migrate-v2.sh preflight
```

预期明确显示目标是 `test-mongo:27017/meowpick`。首次执行还会显示当前没有 `setName`，说明仍是单机模式。

### 4. 创建迁移记录并备份

```bash
bash scripts/jdcloud-migrate-v2.sh init
export MEOWPICK_CONFIRM_EXCLUSIVE_WRITER=ONLY-test-meowpick-writes-meowpick
bash scripts/jdcloud-migrate-v2.sh backup
unset MEOWPICK_CONFIRM_EXCLUSIVE_WRITER
```

如果以后需要开始全新一轮迁移，`init` 不会静默覆盖已有状态指针，必须先确认旧记录已归档，再临时设置 `MEOWPICK_CONFIRM_NEW_RUN=NEW-test-mongo-run`。

设置确认变量前，必须先确认没有其他程序会写入 `meowpick`。因为当前 MongoDB 还是 standalone，脚本只能通过“停止唯一写入者”冻结写入，不能替其他程序保证跨集合备份一致性。

`backup` 会停止 `test-meowpick-backend`，然后：

- 使用 `mongodump` 生成压缩备份；
- 使用 `gzip -t` 检查压缩文件；
- 生成并复核 SHA-256；
- 使用 `mongorestore --dryRun` 验证备份可读取；
- 把所有敏感文件保存在权限为 `700` 的目录中，文件权限为 `600`；
- 记录旧后端镜像 ID，供完整回滚使用。
- 记录数据库 `storageSize`，回滚前要求至少保留其 1.5 倍再加 1 GiB 的空闲空间，用于临时恢复验证。

脚本会输出迁移目录。请在自己电脑的新终端中，把备份及校验文件复制到服务器之外：

```bash
scp Eagle233-JDCloud:/home/eagle233/migrations/meowpick-v2-实际时间/meowpick-before.archive.gz .
scp Eagle233-JDCloud:/home/eagle233/migrations/meowpick-v2-实际时间/meowpick-before.archive.gz.sha256 .
```

### 5. 准备单节点副本集

这一步会写配置并重启 `test-mongo`，必须显式确认：

```bash
export MEOWPICK_CONFIRM_PREPARE=PREPARE-test-mongo-rs0
bash scripts/jdcloud-migrate-v2.sh prepare-replica
unset MEOWPICK_CONFIRM_PREPARE
```

脚本会拒绝覆盖已有 keyFile，只接受完整且规范的 MongoDB `command`，并等待 `rs0` 成为主节点。Compose 和后端配置分别使用临时文件原子替换，但“重启 MongoDB、初始化副本集、修改后端 URI”是有顺序的多阶段操作，不属于一个跨文件事务。如果本阶段中途失败，禁止继续 dry-run/apply，应保留后端停止并按第八部分执行回滚。

### 6. 从同一 commit 构建迁移工具和新后端

```bash
bash scripts/jdcloud-migrate-v2.sh build
```

脚本要求当前分支必须是 `Eagle233` 且工作区无修改，并把完整 commit、迁移镜像名和后端镜像名记录到迁移目录。迁移使用镜像中预编译的 `/app/migrate-v2`，服务器不需要安装 Go，也不会在正式操作时临时下载依赖。

### 7. 运行 dry-run

```bash
bash scripts/jdcloud-migrate-v2.sh dry-run
```

- 退出码 `0`：没有阻塞冲突；
- 退出码 `2`：发现冲突，数据库没有被修改；
- 其他退出码：环境、连接或程序错误。

查看脚本输出的 `migration-v2-plan.json`。重点审核 `conflicts`、`warnings`、`courseRepairs`、`commentIdRepairs` 和 `teacherTimeRepairs`。`conflicts` 不为空时必须停止，处理后重新生成计划。

### 8. 正式迁移

确认备份已复制到服务器外、后端仍停止、计划已经人工审核后执行：

```bash
export MEOWPICK_CONFIRM_APPLY=APPLY-test-mongo-meowpick
export MEOWPICK_CONFIRM_BACKUP=BACKUP-COPIED-OFFSERVER
bash scripts/jdcloud-migrate-v2.sh apply
unset MEOWPICK_CONFIRM_APPLY
unset MEOWPICK_CONFIRM_BACKUP
```

脚本会在 apply 前再次硬检查主机、机器指纹、端口、数据库、容器、挂载、网络、副本集、备份 SHA-256、后端状态、计划 SHA-256、当前仓库的 Git commit、工作区状态和迁移镜像 ID。计划、镜像、代码或数据库快照发生变化时都会拒绝执行。

### 9. 迁移后检查并部署

```bash
bash scripts/jdcloud-migrate-v2.sh postcheck
bash scripts/jdcloud-migrate-v2.sh deploy
```

`postcheck` 要求冲突、课程修复、评论修复、教师时间修复和旧映射 ID 数量全部归零。报告中的 `mappings` 是完整映射清单，不要求为空。

`deploy` 使用 `build` 阶段生成的固定镜像，不会重新拉取可能变化的 `latest`；启动后还会核对容器实际 Image ID，因此迁移工具与新后端来自同一个 commit。

## 八、验证和回滚

### 接口验收标准

1. 已有课程、校区、院系和分类可以正常查询；
2. 可以创建使用已有校区、全新院系和全新分类的提案；
3. 管理员通过提案后，可以查询到正式课程和正确的新映射；
4. 重启后端后再次查询，映射仍然存在；
5. 未知校区会被创建、修改或审批接口拒绝；
6. 提案通过后立即查询用户资料，贡献值已经更新；
7. 普通用户看不到待审核提案；
8. `showUsername=false` 时不能绕过限制查询作者昵称。

这些是迁移后的验收标准，不代表目前已经在 JDCloud 上执行过迁移并得到实测结果。执行后应记录日期、Git commit、镜像名和每项结果。

### 完整回滚

只有正式迁移或新后端验证失败，并且已经确认要恢复迁移前状态时才执行：

```bash
export MEOWPICK_CONFIRM_ROLLBACK=ROLLBACK-test-mongo-meowpick
bash scripts/jdcloud-migrate-v2.sh rollback
unset MEOWPICK_CONFIRM_ROLLBACK
```

`rollback` 是破坏性操作，会停止后端，先恢复旧 Mongo Compose 和旧后端配置并确保 MongoDB 能启动，再复核备份校验和，把备份恢复到临时数据库并生成集合计数与索引清单；临时恢复成功后才删除当前 `meowpick`，使用 `--stopOnError` 正式恢复并比较两份清单，最后恢复旧镜像。临时恢复失败时会尝试清理本轮创建的临时库；成功回滚会写入标记并拒绝重复执行。没有准确确认变量时脚本拒绝执行。

回滚会丢弃“迁移前备份生成之后”的所有新写入。如果正式恢复在删除数据库后仍然失败，脚本会立即停止，不会启动任何后端；此时保留着原始备份和已验证的临时恢复库，应先处理磁盘或权限问题，再重新执行回滚，不能开放接口。

## 九、怎样手写“本次修改说明”

可以按照下面的顺序写：

1. **修改背景**：旧版本把校区、院系和分类写在后端静态映射中，不利于动态新增和持久化。
2. **总体方案**：MongoDB 成为唯一真源，Redis 作为可重建缓存，`monc` 继续承担进程内对象缓存。
3. **映射改造**：增加 `mapping` 和 `mapping_counter`，Redis 使用双向 Hash，批量查询使用 `HMGET`。
4. **提案审批**：未知校区禁止使用，新院系和分类在审批事务中创建。
5. **一致性保证**：课程、教师、映射、提案状态、贡献值和日志在同一事务中提交或回滚。
6. **启动恢复**：后端启动从 MongoDB 构建 Redis 临时键，完成后原子切换。
7. **接口修复**：修复用户资料缓存、提案可见性、并发撤回、可选筛选、空请求体和昵称隐私问题。
8. **数据库迁移**：先 dry-run 检查冲突，再审核并 apply；旧 JSON 不能覆盖线上数据。
9. **服务器要求**：MongoDB 必须是副本集，JDCloud 的目标是 `test-mongo` 单节点副本集 `rs0`。
10. **验收标准**：新增院系和分类在审批后可查询，并且清空缓存或重启后仍然存在。

如果能够不查看前文，把这十点用自己的语言讲清楚，就已经理解了本次改造的主要内容。
