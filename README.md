# Meowpick Backend

![花狮选课猫 Logo](https://s2.loli.net/2025/11/05/lBgG3iYP1MkhwnX.png)

花狮选课猫是面向微信小程序的课程评价平台。本仓库提供后端 API，覆盖微信登录、用户资料、课程与教师搜索、评论、点赞、课程新增提案、管理员审批和操作日志。

## 技术栈

- Go 1.25.5
- Gin
- MongoDB（审批和撤回依赖事务，必须使用副本集或分片集群）
- Redis（缓存；MongoDB 是业务数据和基础映射的真源）
- Google Wire
- Swagger / OpenAPI 3.1
- Docker

## 主要能力

- 微信小程序登录和 JWT Bearer 鉴权；
- 用户昵称、头像、贡献值和管理员权限管理；
- 按课程名称、教师、院系、分类和校区查询课程；
- 课程评论、标签、点赞和个人评论历史；
- 搜索历史与课程、教师、院系、分类建议；
- 用户提交课程新增提案，管理员修改最终课程后通过、拒绝或撤回审批；
- 提案贡献值、贡献者展示和管理员操作日志；
- 院系、课程分类和校区映射持久化到 MongoDB，并使用 Redis 加速双向查询。

## 目录结构

```text
.
├── api/
│   ├── handler/          # HTTP 参数绑定和响应
│   └── router/           # 路由注册
├── application/
│   ├── assembler/        # 数据模型与 DTO 转换
│   ├── dto/              # API 请求和响应结构
│   └── service/          # 业务逻辑
├── cmd/migrate-v2/       # 独立数据库迁移工具
├── docs/                 # OpenAPI 与设计、迁移说明
├── infra/
│   ├── cache/            # Redis 缓存
│   ├── config/           # 配置加载
│   ├── model/            # MongoDB 模型
│   ├── repo/             # 数据访问
│   └── util/             # 通用工具与运行时映射
├── provider/             # Wire 依赖注入
├── scripts/              # BSON 迁移编排脚本
└── types/                # 常量、错误码和迁移种子映射
```

## 本地运行

### 环境要求

- Go 1.25.5 或兼容的更新版本；
- MongoDB 副本集或分片集群，不能使用 standalone；
- Redis；
- 生产登录所需的微信小程序 AppID 和 AppSecret。

只在宿主机运行后端时，可以用 Docker 快速启动本地依赖：

```bash
docker run -d --name meowpick-mongo \
  -p 27017:27017 \
  mongo:7.0.23-jammy \
  mongod --replSet rs0 --bind_ip_all
```

使用下面的命令观察启动日志；出现 `Waiting for connections` 后按 `Ctrl-C` 退出日志跟踪：

```bash
docker logs -f meowpick-mongo
```

然后初始化副本集并启动 Redis：

```bash
docker exec meowpick-mongo mongosh --quiet --eval \
  'rs.initiate({_id:"rs0",members:[{_id:0,host:"localhost:27017"}]})'

docker run -d --name meowpick-redis \
  -p 6379:6379 \
  redis:7-alpine
```

上述副本集地址适用于宿主机上的 `go run .`。如果后端也运行在容器中，需要让 MongoDB、Redis 和后端加入同一 Docker 网络，并使用容器可解析的主机名配置和初始化副本集。

### 配置

配置文件默认读取 `etc/config.yaml`，也可以通过 `CONFIG_PATH` 指定其他路径。仓库有意忽略整个 `etc/` 目录，因此全新 clone 后需要先创建目录，并把下面的示例保存为 `etc/config.yaml`；不要提交真实密钥。

```bash
mkdir -p etc
```

```yaml
Name: meowpick-backend
ListenOn: 0.0.0.0:8080
Mode: dev
State: local
Log:
  Level: info

Auth:
  SecretKey: "replace-with-a-long-random-secret"
  PublicKey: "replace-with-a-public-key-or-compatible-placeholder"
  AccessExpire: 86400

Mongo:
  URL: "mongodb://127.0.0.1:27017/?replicaSet=rs0&directConnection=true"
  DB: "meowpick"

Cache:
  - Host: "127.0.0.1:6379"
    Type: node

Redis:
  Host: "127.0.0.1:6379"
  Type: node

WeApp:
  AppID: "your-weapp-appid"
  AppSecret: "your-weapp-appsecret"

AdminGrantKey: "replace-with-a-separate-admin-verification-code"
```

`State: local` 会启用仅供本地调试的登录验证码 `test123`，部署环境不得使用该值。

`Cache` 是各 MongoDB Repository 使用的文档缓存配置；`Redis` 是点赞和基础映射等显式缓存使用的连接配置。当前两者都必须提供，通常指向同一个 Redis 实例。

应用启动会幂等创建运行时所需索引，但不会为全新空数据库自动写入完整的校区、院系和分类种子，仓库目前也没有一键空库种子命令。纯本地可以从空映射启动服务并验证登录等不依赖基础映射的接口，但空库没有校区记录，不能直接创建和审批课程提案；院系、分类可由相应业务流程按需创建，校区必须通过迁移或受控管理方式预置。需要完整业务链路时，应恢复一份已脱敏的 BSON，再按[数据库迁移流程](docs/MIGRATION-V2.md)生成并审核迁移计划。已有旧库升级也必须走该迁移流程，不能直接启动新版覆盖数据。

### 启动服务

```bash
go mod download
CONFIG_PATH=etc/config.yaml go run .
```

当前 `main.go` 固定监听 `8080` 端口；配置中的 `ListenOn` 尚不能改变实际监听地址：

- Swagger UI：<http://localhost:8080/swagger/index.html>
- OpenAPI JSON：<http://localhost:8080/openapi.json>

本地调试登录示例：

```bash
curl -X POST http://localhost:8080/api/auth/sign_in \
  -H 'Content-Type: application/json' \
  -d '{"authId":"local","authType":"wechat","verifyCode":"test123"}'
```

`State: local` 且验证码为 `test123` 时不会调用微信接口，因此本地调试可以使用占位的 WeApp 配置。其他请求把响应中的 `accessToken` 放入 `Authorization: Bearer <token>` 请求头。

## 开发与验证

```bash
# 格式化修改过的 Go 文件
gofmt -w path/to/changed.go

# 单元测试和静态检查
go test ./...
go vet ./...

# 生成依赖注入代码和 OpenAPI 文档
make wire
make swagger
```

修改 Wire 依赖关系后必须运行 `make wire`；修改 Handler 注释或 DTO 后必须运行 `make swagger`，并提交生成文件。

首次执行生成命令前安装对应 CLI：

```bash
go install github.com/google/wire/cmd/wire@v0.7.0
go install github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc5
```

## Docker

```bash
docker build -t meowpick-backend .

docker run --rm \
  -p 8080:8080 \
  -e CONFIG_PATH=/app/etc/config.yaml \
  -v "$PWD/etc/config.yaml:/app/etc/config.yaml:ro" \
  meowpick-backend
```

配置中的 MongoDB 和 Redis 地址必须能从容器内部访问；容器内的 `127.0.0.1` 指向容器本身。

## API 模块

所有业务路由均以 `/api` 开头：

| 模块 | 路径前缀 | 说明 |
| --- | --- | --- |
| 认证 | `/api/auth` | 登录、管理员状态和权限切换 |
| 用户 | `/api/user` | 用户资料与贡献者昵称权限 |
| 课程 | `/api/course` | 课程详情及院系、分类、校区查询 |
| 教师 | `/api/teacher` | 教师创建与建议 |
| 评论 | `/api/comment` | 发布、课程评论列表和个人历史 |
| 点赞 | `/api/like` | 评论或提案点赞切换 |
| 搜索 | `/api/search` | 课程搜索、建议、历史和统计 |
| 提案 | `/api/proposal` | 创建、查询、修改、审批、拒绝和撤回；`/history` 为“我的提案” |
| 变更日志 | `/api/changelog` | 管理操作列表和提案时间线 |

准确的请求参数、权限与响应结构以运行时 Swagger 和 [`docs/swagger.yaml`](docs/swagger.yaml) 为准。

## 数据库升级

从旧版数据库升级时，不要直接启动新版后端，也不要用旧 JSON 覆盖数据库。应先导出 `mongodump` BSON，在隔离 MongoDB 中执行 dry-run、迁移和 postcheck，再部署应用。

- [数据库迁移流程](docs/MIGRATION-V2.md)
- [基础映射运行时设计](docs/REFERENCE-MAPPINGS.md)
- [提案映射与迁移说明](docs/USER-PROPOSAL-CONSISTENCY.md)
- [用户与提案接口修复、验收项和已知限制](docs/USER-PROPOSAL-API-FIXES.md)

## License

[Apache License 2.0](LICENSE)
