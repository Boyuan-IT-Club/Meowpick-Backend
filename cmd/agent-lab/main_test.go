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
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestMultiTurnRequestsAndFailedTurn(t *testing.T) {
	const key = "fake-secret-for-test-only"
	var requests []completionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("unexpected method or headers")
		}
		body, _ := io.ReadAll(r.Body)
		if bytes.Contains(body, []byte(key)) {
			t.Error("credential must not appear in model messages")
		}
		var req completionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests = append(requests, req)
		if len(requests) == 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, key)
			return
		}
		answer := "A、B、C"
		if len(requests) == 3 {
			answer = "第二个是 B"
		}
		_ = json.NewEncoder(w).Encode(completionResponse{
			Choices: []completionChoice{{Message: message{Role: "assistant", Content: answer}, FinishReason: "stop"}},
		})
	}))
	defer server.Close()
	session := newSession(newChatClient(server.URL, key, defaultModel))
	if _, err := session.ask(context.Background(), "请确认 A、B、C 的顺序"); err != nil {
		t.Fatal(err)
	}
	beforeFailure := append([]message(nil), session.messages...)
	if _, err := session.ask(context.Background(), "失败的问题"); err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("expected sanitized HTTP failure, got %v", err)
	}
	if !reflect.DeepEqual(session.messages, beforeFailure) {
		t.Fatal("failed request changed successful conversation history")
	}
	answer, err := session.ask(context.Background(), "第二个是什么？")
	if err != nil || answer != "第二个是 B" {
		t.Fatalf("answer=%q err=%v", answer, err)
	}
	want := []message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "请确认 A、B、C 的顺序"},
		{Role: "assistant", Content: "A、B、C"},
		{Role: "user", Content: "第二个是什么？"},
	}
	if len(requests) != 3 || !reflect.DeepEqual(requests[2].Messages, want) {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	for _, req := range requests {
		if req.Model != defaultModel || req.Stream {
			t.Fatal("unexpected model or streaming option")
		}
	}
	if len(session.messages) != 5 {
		t.Fatalf("expected two complete turns and system message, got %d messages", len(session.messages))
	}
}

func TestRejectIncompleteAndMalformedResponses(t *testing.T) {
	for name, body := range map[string]string{
		"invalid JSON":       `{broken`,
		"missing choice":     `{"choices":[]}`,
		"truncated answer":   `{"choices":[{"message":{"role":"assistant","content":"partial"},"finish_reason":"length"}]}`,
		"unhandled tool":     `{"choices":[{"message":{"role":"assistant","content":null},"finish_reason":"tool_calls"}]}`,
		"wrong role":         `{"choices":[{"message":{"role":"user","content":"bad"},"finish_reason":"stop"}]}`,
		"empty answer":       `{"choices":[{"message":{"role":"assistant","content":" "},"finish_reason":"stop"}]}`,
		"oversized response": strings.Repeat("x", maxResponseBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			session := newSession(newChatClient(server.URL, "test-only", defaultModel))
			if _, err := session.ask(context.Background(), "问题"); err == nil {
				t.Fatal("invalid response accepted")
			}
			if len(session.messages) != 1 {
				t.Fatal("invalid response entered history")
			}
		})
	}
}

func TestNoRedirect(t *testing.T) {
	var forwarded bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		forwarded = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := newChatClient(server.URL, "secret", defaultModel)
	if _, err := client.complete(context.Background(), []message{{Role: "user", Content: "test"}}); err == nil {
		t.Fatal("redirect accepted")
	}
	if forwarded {
		t.Fatal("request forwarded to redirected destination")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCancellationAndTransportErrorRedaction(t *testing.T) {
	client := newChatClient(schoolEndpoint, "secret", defaultModel)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.complete(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	client.http.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("proxy error containing secret")
	})
	if _, err := client.complete(context.Background(), nil); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("expected sanitized transport error, got %v", err)
	}
}

func TestDemoAndLiveConfiguration(t *testing.T) {
	var output bytes.Buffer
	if err := run(context.Background(), nil, strings.NewReader(""), &output, func(string) string {
		t.Fatal("demo attempted to read environment")
		return ""
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"本机模拟演示", "messages=2", "messages=4", "system,user,assistant,user", "模拟服务预设"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing demo explanation %q", want)
		}
	}
	if err := run(context.Background(), []string{"-live"}, strings.NewReader(""), io.Discard, func(string) string { return "" }); err == nil {
		t.Fatal("live mode accepted missing key")
	}
}

func TestInputAndContextLimits(t *testing.T) {
	session := newSession(nil) // Invalid input must be rejected before any client call.
	for _, question := range []string{" ", strings.Repeat("x", maxInputBytes+1)} {
		if _, err := session.ask(context.Background(), question); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	session.messages = append(session.messages, message{Role: "assistant", Content: strings.Repeat("x", maxContextBytes)})
	if _, err := session.ask(context.Background(), "问题"); err == nil {
		t.Fatal("oversized context accepted")
	}
}
