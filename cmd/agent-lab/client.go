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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxResponseBytes = 1 << 20

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type completionResponse struct {
	Choices []completionChoice `json:"choices"`
}

type completionChoice struct {
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type chatClient struct {
	endpoint string
	apiKey   string
	model    string
	http     *http.Client
}

func newChatClient(endpoint, apiKey, model string) *chatClient {
	return &chatClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		http: &http.Client{
			Timeout: requestTimeout,
			// Do not forward credentials or requests to a redirected endpoint.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *chatClient) complete(ctx context.Context, messages []message) (string, error) {
	body, err := json.Marshal(completionRequest{Model: c.model, Messages: messages, Stream: false})
	if err != nil {
		return "", errors.New("无法编码模型请求")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("无法构造模型请求")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// Provider/transport errors can contain secrets. Never print their bodies.
		return "", errors.New("模型请求失败或超时；未自动重试")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("模型接口返回 HTTP %d；未输出响应正文，未自动重试", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New("读取模型响应失败")
	}
	if len(data) > maxResponseBytes {
		return "", errors.New("模型响应超过 1 MiB")
	}
	var result completionResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", errors.New("模型响应不是有效的 Chat Completions JSON")
	}
	if len(result.Choices) != 1 {
		return "", errors.New("模型响应应包含一条 choice")
	}
	choice := result.Choices[0]
	if choice.FinishReason != "stop" {
		return "", errors.New("模型未正常完成文本回复；本实验尚未处理工具调用或截断输出")
	}
	if choice.Message.Role != "assistant" || strings.TrimSpace(choice.Message.Content) == "" {
		return "", errors.New("模型未返回有效的 assistant 文本")
	}
	return choice.Message.Content, nil
}
