// Copyright 2025 Boyuan-IT-Club
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

package wechatsecurity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestContentSecurityClientCachesTokenAndSendsV2Request(t *testing.T) {
	var tokenCalls atomic.Int32
	var checkCalls atomic.Int32
	client := testClient(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case "/stable_token":
			tokenCalls.Add(1)
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode token request: %v", err)
			}
			if req["appid"] != "wx-app" || req["secret"] != "secret" || req["force_refresh"] != false {
				t.Fatalf("unexpected token request: %#v", req)
			}
			return jsonResponse(t, map[string]any{"access_token": "token-1", "expires_in": 7200})
		case "/msg_sec_check":
			checkCalls.Add(1)
			if got := r.URL.Query().Get("access_token"); got != "token-1" {
				t.Fatalf("access_token = %q, want token-1", got)
			}
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode check request: %v", err)
			}
			if req["content"] != "昵称" || req["nickname"] != "昵称" || req["openid"] != "openid-1" || req["version"] != float64(2) || req["scene"] != float64(1) {
				t.Fatalf("unexpected check request: %#v", req)
			}
			return jsonResponse(t, map[string]any{
				"errcode":  0,
				"errmsg":   "ok",
				"result":   map[string]any{"suggest": "pass", "label": 100},
				"trace_id": "trace-1",
			})
		default:
			return jsonResponse(t, map[string]any{"errcode": 404, "errmsg": "not found"})
		}
	})

	req := TextCheckRequest{OpenID: "openid-1", Scene: SceneProfile, Content: "昵称", Nickname: "昵称"}
	for range 2 {
		result, err := client.CheckText(context.Background(), req)
		if err != nil {
			t.Fatalf("CheckText() error = %v", err)
		}
		if result.Suggest != "pass" || result.Label != 100 || result.TraceID != "trace-1" {
			t.Fatalf("result = %#v", result)
		}
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("token calls = %d, want 1", got)
	}
	if got := checkCalls.Load(); got != 2 {
		t.Fatalf("check calls = %d, want 2", got)
	}
}

func TestContentSecurityClientRefreshesCredentialOnce(t *testing.T) {
	var tokenCalls atomic.Int32
	var checkCalls atomic.Int32
	client := testClient(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case "/stable_token":
			call := tokenCalls.Add(1)
			var req struct {
				ForceRefresh bool `json:"force_refresh"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode token request: %v", err)
			}
			if (call == 1 && req.ForceRefresh) || (call == 2 && !req.ForceRefresh) {
				t.Fatalf("call %d force_refresh = %v", call, req.ForceRefresh)
			}
			return jsonResponse(t, map[string]any{"access_token": "token-" + string(rune('0'+call)), "expires_in": 7200})
		case "/msg_sec_check":
			call := checkCalls.Add(1)
			if call == 1 {
				return jsonResponse(t, map[string]any{"errcode": 42001, "errmsg": "access_token expired"})
			}
			if got := r.URL.Query().Get("access_token"); got != "token-2" {
				t.Fatalf("retry access_token = %q, want token-2", got)
			}
			return jsonResponse(t, map[string]any{
				"errcode":  0,
				"errmsg":   "ok",
				"result":   map[string]any{"suggest": "pass", "label": 100},
				"trace_id": "trace-2",
			})
		default:
			return jsonResponse(t, map[string]any{"errcode": 404, "errmsg": "not found"})
		}
	})

	_, err := client.CheckText(context.Background(), TextCheckRequest{
		OpenID: "openid-1", Scene: SceneComment, Content: "评论",
	})
	if err != nil {
		t.Fatalf("CheckText() error = %v", err)
	}
	if tokenCalls.Load() != 2 || checkCalls.Load() != 2 {
		t.Fatalf("token calls = %d, check calls = %d; want 2 each", tokenCalls.Load(), checkCalls.Load())
	}
}

func TestContentSecurityClientDoesNotRetryNonCredentialError(t *testing.T) {
	var tokenCalls atomic.Int32
	var checkCalls atomic.Int32
	client := testClient(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case "/stable_token":
			tokenCalls.Add(1)
			return jsonResponse(t, map[string]any{"access_token": "token-1", "expires_in": 7200})
		case "/msg_sec_check":
			checkCalls.Add(1)
			return jsonResponse(t, map[string]any{"errcode": 45009, "errmsg": "rate limit"})
		default:
			return jsonResponse(t, map[string]any{"errcode": 404, "errmsg": "not found"})
		}
	})

	_, err := client.CheckText(context.Background(), TextCheckRequest{
		OpenID: "openid-1", Scene: SceneComment, Content: "评论",
	})
	if err == nil {
		t.Fatal("CheckText() error = nil, want error")
	}
	if tokenCalls.Load() != 1 || checkCalls.Load() != 1 {
		t.Fatalf("token calls = %d, check calls = %d; want 1 each", tokenCalls.Load(), checkCalls.Load())
	}
}

func testClient(handler func(*http.Request) *http.Response) *ContentSecurityClient {
	return &ContentSecurityClient{
		appID:     "wx-app",
		appSecret: "secret",
		httpClient: &http.Client{
			Timeout: time.Second,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return handler(req), nil
			}),
		},
		tokenURL:     "https://wechat.test/stable_token",
		textCheckURL: "https://wechat.test/msg_sec_check",
		now:          time.Now,
	}
}

func jsonResponse(t *testing.T, value any) *http.Response {
	t.Helper()
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body.String())),
	}
}
