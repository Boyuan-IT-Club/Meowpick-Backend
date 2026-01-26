# Copyright 2025 Boyuan-IT-Club
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# 使用官方 Go 1.25.5 Alpine 镜像
FROM golang:1.25.6-alpine AS builder

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

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./meowpick-backend"]
