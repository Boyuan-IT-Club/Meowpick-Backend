// Copyright 2026 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command agent-lab demonstrates explicit multi-turn Chat Completions requests.
// It has no connection to the application's database, routes, or configuration.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	schoolEndpoint  = "https://chat.ecnu.edu.cn/open/api/v1/chat/completions"
	defaultModel    = "ecnu-plus"
	requestTimeout  = 45 * time.Second
	maxInputBytes   = 8 << 10
	maxContextBytes = 64 << 10
	systemPrompt    = "你是选课猫问答接入实验助手。当前没有接入真实课程和评论，不能编造课程推荐或引用。可以围绕用户明确提供的信息对话，缺少资料时说明。请简洁回答。"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "agent-lab:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, input io.Reader, output io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("agent-lab", flag.ContinueOnError)
	flags.SetOutput(output)
	live := flags.Bool("live", false, "调用学校真实模型；默认运行本机模拟演示")
	model := flags.String("model", defaultModel, "真实模式的学校对话模型名称")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("不接受位置参数；使用 -h 查看帮助")
	}
	if !*live {
		return runDemo(ctx, output)
	}
	key := strings.TrimSpace(getenv("MEOWPICK_AGENT_API_KEY"))
	if key == "" {
		return errors.New("真实模式需要本机环境变量 MEOWPICK_AGENT_API_KEY；不要将 Key 作为命令行参数或提交到仓库")
	}
	if strings.TrimSpace(*model) == "" {
		return errors.New("模型名称不能为空")
	}
	client := newChatClient(schoolEndpoint, key, strings.TrimSpace(*model))
	session := newSession(client)
	fmt.Fprintln(output, "真实模式：使用你的学校 Key；聊天只保存在本进程内。输入 /exit 结束。")
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), maxInputBytes+1)
	for {
		fmt.Fprint(output, "你 > ")
		if !scanner.Scan() {
			break
		}
		question := strings.TrimSpace(scanner.Text())
		if question == "/exit" {
			return nil
		}
		if question == "" {
			continue
		}
		printContext(output, session.messages, question)
		turnCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		answer, err := session.ask(turnCtx, question)
		cancel()
		if err != nil {
			return err
		}
		fmt.Fprintln(output, "助手 >", answer)
	}
	if scanner.Err() != nil {
		return errors.New("读取输入失败或单条输入超过 8 KiB")
	}
	return nil
}

// printContext exposes the message shape without printing credentials or bodies.
func printContext(output io.Writer, history []message, question string) {
	roles := make([]string, 0, len(history)+1)
	for _, msg := range history {
		roles = append(roles, msg.Role)
	}
	roles = append(roles, "user")
	fmt.Fprintf(output, "请求：messages=%d，roles=%s，当前问题=%d 字节\n", len(roles), strings.Join(roles, ","), len(question))
}

// runDemo uses a loopback HTTP fixture, not an LLM or production course data.
func runDemo(ctx context.Context, output io.Writer) error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req completionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		answer := "演示确认：你提供的顺序是 A、B、C。这些是虚构代号，不是真实课程推荐。"
		if len(req.Messages) == 4 {
			answer = "第二个是 B；这次请求包含上一轮用户消息与助手回复，所以能够定位。此回复由模拟服务预设，不代表真实模型效果。"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(completionResponse{
			Choices: []completionChoice{{Message: message{Role: "assistant", Content: answer}, FinishReason: "stop"}},
		})
	}))
	defer server.Close()
	fmt.Fprintln(output, "本机模拟演示：不读取 Key，不调用学校模型，不连接业务数据库。")
	session := newSession(newChatClient(server.URL, "demo-only", defaultModel))
	for _, question := range []string{"本实验的三个虚构代号依次为 A、B、C，请确认。", "第二个是什么？"} {
		fmt.Fprintln(output, "你 >", question)
		printContext(output, session.messages, question)
		answer, err := session.ask(ctx, question)
		if err != nil {
			return err
		}
		fmt.Fprintln(output, "助手 >", answer)
	}
	return nil
}

type chatSession struct {
	client   *chatClient
	messages []message
}

func newSession(client *chatClient) *chatSession {
	return &chatSession{client: client, messages: []message{{Role: "system", Content: systemPrompt}}}
}

func (s *chatSession) ask(ctx context.Context, question string) (string, error) {
	if strings.TrimSpace(question) == "" || len(question) > maxInputBytes {
		return "", errors.New("问题不能为空且不能超过 8 KiB")
	}
	total := len(question)
	for _, msg := range s.messages {
		total += len(msg.Content)
	}
	if total > maxContextBytes {
		return "", errors.New("实验上下文超过 64 KiB；本实验暂不实现正式的 Token 裁剪或摘要")
	}
	// Keep the previous successful history unchanged if this request fails.
	pending := append([]message(nil), s.messages...)
	pending = append(pending, message{Role: "user", Content: question})
	answer, err := s.client.complete(ctx, pending)
	if err != nil {
		return "", err
	}
	s.messages = append(pending, message{Role: "assistant", Content: answer})
	return answer, nil
}
