# 课程问答接入实验

这是观察 Chat Completions 协议的独立命令，不是已上线的问答服务。
不调用业务 Provider，不读取业务配置，不连接 MongoDB 或 Redis，不注册 HTTP 路由。
仅使用 Go 标准库，暂未引入 Eino。

## 无 Key 演示

在仓库根目录运行：

```sh
go run ./cmd/agent-lab
```

程序启动一个仅监听本机的 HTTP 模拟服务，执行两轮预设对话，并关闭模拟服务。
回复由程序预设，不代表真实模型理解能力或推荐质量。
两轮会输出：

```text
请求：messages=2，roles=system,user，……
请求：messages=4，roles=system,user,assistant,user，……
```

观察第二次请求为什么包含上一轮用户消息与助手回答。
不会读取环境中的模型 Key，也不向学校发送请求。

## 使用自己的学校 Key

在 macOS 的 zsh 终端中隐藏输入 Key，避免把字面值写进命令历史：

```zsh
read -rs 'MEOWPICK_AGENT_API_KEY?学校 API Key: '
export MEOWPICK_AGENT_API_KEY
echo
go run ./cmd/agent-lab -live
unset MEOWPICK_AGENT_API_KEY
```

默认调用 `https://chat.ecnu.edu.cn/open/api/v1/chat/completions`，模型为 `ecnu-plus`。
可以用 `-model ecnu-max` 选择另一学校对话模型。调用消耗由所填 Key 承担。
输入 `/exit` 或结束标准输入退出；Ctrl-C 可取消正在进行的 HTTP 请求。
Key 仅用于 Authorization 请求头，不放入消息，不写入文件。
错误不输出上游响应正文；HTTP 重定向不跟随，失败不自动重试。

建议输入：

```text
本实验中三个虚构代号依次为 A、B、C，请确认。
第二个是什么？
```

程序仅显示消息数量、角色顺序和当前问题的字节数，以及助手回复；不打印认证头。
不要把真实 Key 发进聊天或作为命令行参数。

## 代码阅读路线

1. `main.go` 的 `run`：默认模拟或显式选择真实模式。
2. `chatSession.ask`：复制历史，追加当前问题，成功后追加助手回答。
3. `client.go` 的 `chatClient.complete`：构造 JSON、发送认证请求、检查状态与文本回复。
4. 返回 `ask`：只将正常完成的文本回复纳入本实验的下一轮上下文。

## 验证范围与限制

- 单次请求超时 45 秒，输入最多 8 KiB，响应最多 1 MiB；历史输入以 64 KiB 限制。
- 字节限制是实验保护，不是 Token 计数或正式上下文选择策略。
- 所有聊天仅保存在内存，退出即丢失；没有 Conversation/Message 数据库持久化或用户 Key 加密托管。
- 失败请求不改变本实验的成功历史；正式服务仍需单独持久化失败消息状态。
- 当前未处理流式、工具调用、RAG、索引、真实课程查询或 Eino 编排。
- `finish_reason` 非 `stop`（例如工具调用、截断）明确报错，不当成成功回答。
- 自动化测试使用本机模拟接口，不能证明学校当前的连通性、Key 授权或模型效果。

```sh
go test ./cmd/agent-lab
```

接口依据：[学校通用对话文档](https://developer.ecnu.edu.cn/vitepress/llm/api/completions.html)。
