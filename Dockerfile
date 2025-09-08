# 使用官方 Go 1.24.5 Alpine 镜像
FROM golang:1.24.5-alpine AS builder

# 安装必要工具（可选，根据需求）
RUN apk add --no-cache git bash

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod tidy

# 复制其余项目文件
COPY . .

# 构建可执行文件
RUN go build -o meowpick-backend main.go

# 使用精简镜像运行
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/meowpick-backend ./

# 暴露端口（根据你项目端口修改）
EXPOSE 8080

# 启动命令
CMD ["./meowpick-backend"]
