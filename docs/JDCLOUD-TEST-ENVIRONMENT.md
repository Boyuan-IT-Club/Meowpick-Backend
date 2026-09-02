# JDCloud 测试环境迁移约束

本文供后续 Agents 和开发人员维护迁移逻辑时使用。项目负责人只需阅读仓库根目录的 `MEOWPICK-V2-改造与迁移说明.md`。

## 已通过只读 SSH 确认的环境

- SSH Host：`Eagle233-JDCloud`；
- 已核验的服务器 OS hostname：`Eagle233-JDCloud`；
- 脚本同时校验 `/etc/machine-id` 的 SHA-256 和本机 `unix:///var/run/docker.sock`，并拒绝 `DOCKER_HOST`/`DOCKER_CONTEXT` 覆盖；
- 迁移二进制必须携带 `--replica-set rs0 --require-host test-mongo --require-port 27017 --require-db meowpick`，以应用自身的 YAML 解析结果做连接前最终保护；
- Meowpick 后端容器：`test-meowpick-backend`；
- 后端配置：`/home/eagle233/repos/test/meowpick/config.yaml`；
- 后端当前连接：`test-mongo:27017/meowpick`；
- MongoDB 容器：`test-mongo`，镜像 `mongo:7.0.23-jammy`；
- Docker 网络：`test-net`；
- 宿主机端口：`27015 -> 27017`；
- 数据目录：`/home/eagle233/repos/test/mongo/data`；
- 配置目录：`/home/eagle233/repos/test/mongo/config`；
- Portainer Compose：`/home/eagle233/portainer/srv/compose/1/v1/docker-compose.yml`；
- 后端部署 Compose：`/home/eagle233/repos/test/meowpick/docker-compose.yml`；
- 服务器没有 Go，但有 Docker Compose v5 和 Git。

不得把 `prod-mongo` 当作本次 Meowpick 迁移目标。

## MongoDB 拓扑前置条件

2026-08-31 的只读 `db.hello()` 结果没有 `setName`，说明 `test-mongo` 当前是 standalone。新版提案审批和迁移 apply 使用 MongoDB 事务，必须先转换为带 keyFile 的单节点副本集 `rs0`。

迁移操作必须由用户手动执行。Agent 不得仅凭本文对 JDCloud 执行停止容器、修改 Compose、初始化副本集、迁移或恢复。

`scripts/jdcloud-migrate-v2.sh` 是纯数据库运维脚本：不得停止、启动或部署后端，不得修改后端配置，不得构建后端镜像。它可以只读现有后端配置取得 MongoDB URI，并检查 `test-meowpick-backend` 已由操作者停止。迁移镜像必须使用 `Dockerfile.migrate-v2` 独立构建；正常后端 `Dockerfile` 不得包含迁移二进制。

在 standalone 阶段备份前，操作者必须手动停止 `test-meowpick-backend`，确认它是 `meowpick` 的唯一写入者，并停止其他可能写入该库的程序。回滚只恢复 MongoDB Compose 和数据库，再进行临时库恢复验证；所需空闲空间下限由备份时记录的 `storageSize` 计算。后端部署、配置和回滚由操作者另行完成。

MongoDB 管理命令从容器自身的环境变量取得账户信息，脚本和文档不保存明文凭据。Database Tools 在执行时仍会把展开后的认证参数短暂放入容器内进程参数，因此迁移窗口必须限制服务器和 Docker 访问权限；后续如果切换到已验证的权限文件认证，可再消除此残余风险。维护此脚本时不得把真实 URI、密码、完整 Compose 环境变量或配置 diff 输出到日志。

## 文档分层

- 根目录 `MEOWPICK-V2-改造与迁移说明.md`：唯一用户主文档，包含修改解释和可复制的服务器命令；
- `scripts/jdcloud-migrate-v2.sh`：固定目标、分阶段、失败即停的 JDCloud 实际迁移脚本；
- `docs/MIGRATION-V2.md`：通用迁移实现、冲突调查和数据库检查细节；
- `docs/USER-PROPOSAL-CONSISTENCY.md`：映射及提案一致性的开发说明；
- `docs/USER-PROPOSAL-API-FIXES.md`：用户资料和提案接口修复明细。

修改迁移参数、映射结构、容器名称、数据库名称或服务器部署方式时，必须同步更新根目录主文档和本文，避免操作命令失效。
