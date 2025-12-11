# Meowpick-Backend

![logo.png](https://s2.loli.net/2025/11/05/lBgG3iYP1MkhwnX.png)

**花狮选课猫**是一个基于微信小程序的课程评价平台，学生可以在这里匿名分享对课程的真实评价和体验。

## 项目概述

选课猫后端采用 Go 语言开发，使用 Gin 框架构建 RESTful API，数据存储在 MongoDB 中。该项目主要为微信小程序提供后端服务支持，实现课程评价、搜索、点赞等核心功能。

## 技术栈

- **语言**: Go 1.24.5
- **框架**: Gin
- **数据库**: MongoDB
- **缓存**: Redis
- **依赖注入**: Google Wire
- **API 文档**: Swagger 注释（待完善）
- **部署**: Docker

## 功能特性

### 用户认证
- 微信小程序用户登录
- JWT Token 认证机制

### 课程相关
- 搜索课程（按名称模糊搜索）
- 查看课程详情
- 获取课程分类、部门、校区等元信息

### 评论系统
- 发布课程评论
- 查看课程下的评论列表
- 查看我的历史评论
- 获取平台总评论数

### 教师相关
- 搜索教师
- 查看教师教授的课程

### 点赞功能
- 对评论进行点赞/取消点赞
- 使用 Redis 缓存优化点赞计数性能

### 搜索功能
- 课程和教师模糊搜索
- 搜索历史记录（每个用户保留最近15条）
- 搜索建议（输入关键字时实时推荐）

## 项目架构

```
.
├── adaptor          # 适配器层（控制器、路由、命令等）
│   ├── cmd          # API 请求/响应结构体
│   ├── controller   # 控制器层
│   ├── router       # 路由配置
│   └── token        # 认证相关
├── application      # 应用服务层
│   ├── dto          # 数据传输对象
│   └── service      # 业务逻辑服务
├── infra            # 基础设施层
│   ├── cache        # 缓存实现
│   ├── c       # 配置管理
│   ├── consts       # 常量定义
│   ├── mapper       # 数据访问层
│   └── util         # 工具类
└── provider         # 依赖注入配置
```

## API 接口

### 认证接口
- `POST /api/sign_in` - 用户登录

### 评论接口
- `POST /api/comment/add` - 发布评论
- `GET /api/comment/query` - 分页获取课程下的评论
- `POST /api/comment/history` - 获取我的评论历史

### 搜索接口
- `GET /api/search/recent` - 获取搜索历史
- `POST /api/search` - 搜索课程或教师
- `GET /api/search/total` - 获取平台总评论数
- `GET /api/search/suggest` - 获取搜索建议

### 点赞接口
- `POST /api/action/like/:id` - 为评论点赞

### 课程接口
- `GET /api/course/:courseId` - 获取特定课程详情

## 部署说明

### 环境要求
- Go 1.24.5 或更高版本
- MongoDB
- Redis

### 配置文件
项目需要一个 `etc/c.yaml` 配置文件，包含以下内容：
```yaml
Name: meowpick.backend
Host: 0.0.0.0
Port: 8080
Auth:
  SecretKey: "your-secret-key"
  PublicKey: "your-public-key"
  AccessExpire: 86400
Mongo:
  URL: "mongodb://localhost:27017"
  DB: "meowpick"
Redis:
  Host: "localhost:6379"
  Type: "node"
WeApp:
  AppID: "your-weapp-appid"
  AppSecret: "your-weapp-secret"
```

### 使用 Docker 部署
```bash
# 构建镜像
docker build -db meowpick-backend .

# 运行容器
docker run -d -p 8080:8080 meowpick-backend
```

### 本地开发
```bash
# 克隆项目
git clone https://github.com/Boyuan-IT-Club/Meowpick-Backend.git

# 进入项目目录
cd Meowpick-Backend

# 下载依赖
go mod tidy

# 运行项目
go run main.go
```

## 项目成员

- [博远信息技术社](https://official.boyuan.club/)

## License

[Apache License 2.0](LICENSE)