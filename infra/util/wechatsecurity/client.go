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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/google/wire"
)

const (
	stableTokenEndpoint = "https://api.weixin.qq.com/cgi-bin/stable_token"
	textCheckEndpoint   = "https://api.weixin.qq.com/wxa/msg_sec_check"
	responseBodyLimit   = 1 << 20
)

type TextScene int

const (
	SceneProfile TextScene = 1
	SceneComment TextScene = 2
)

type TextCheckRequest struct {
	OpenID   string
	Scene    TextScene
	Content  string
	Nickname string
}

type TextCheckResult struct {
	Suggest string
	Label   int
	TraceID string
}

type IContentSecurityClient interface {
	CheckText(ctx context.Context, req TextCheckRequest) (*TextCheckResult, error)
}

type APIError struct {
	Code int
	Msg  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("wechat API error: code=%d msg=%s", e.Code, e.Msg)
}

type ContentSecurityClient struct {
	appID        string
	appSecret    string
	httpClient   *http.Client
	tokenURL     string
	textCheckURL string
	now          func() time.Time

	tokenMu        sync.Mutex
	accessToken    string
	tokenExpiresAt time.Time
}

var ContentSecurityClientSet = wire.NewSet(
	NewContentSecurityClient,
	wire.Bind(new(IContentSecurityClient), new(*ContentSecurityClient)),
)

func NewContentSecurityClient(cfg *config.Config) *ContentSecurityClient {
	return &ContentSecurityClient{
		appID:        cfg.WeApp.AppID,
		appSecret:    cfg.WeApp.AppSecret,
		httpClient:   &http.Client{Timeout: 3 * time.Second},
		tokenURL:     stableTokenEndpoint,
		textCheckURL: textCheckEndpoint,
		now:          time.Now,
	}
}

func (c *ContentSecurityClient) CheckText(ctx context.Context, req TextCheckRequest) (*TextCheckResult, error) {
	token, err := c.getAccessToken(ctx, false)
	if err != nil {
		return nil, err
	}

	result, err := c.checkTextWithToken(ctx, token, req)
	if !isCredentialError(err) {
		return result, err
	}

	c.invalidateAccessToken(token)
	token, err = c.getAccessToken(ctx, true)
	if err != nil {
		return nil, err
	}
	return c.checkTextWithToken(ctx, token, req)
}

func (c *ContentSecurityClient) getAccessToken(ctx context.Context, forceRefresh bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if !forceRefresh && c.accessToken != "" && c.now().Before(c.tokenExpiresAt) {
		return c.accessToken, nil
	}

	payload := struct {
		GrantType    string `json:"grant_type"`
		AppID        string `json:"appid"`
		Secret       string `json:"secret"`
		ForceRefresh bool   `json:"force_refresh"`
	}{
		GrantType:    "client_credential",
		AppID:        c.appID,
		Secret:       c.appSecret,
		ForceRefresh: forceRefresh,
	}
	var response struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := c.postJSON(ctx, c.tokenURL, payload, &response); err != nil {
		return "", err
	}
	if response.ErrCode != 0 {
		return "", &APIError{Code: response.ErrCode, Msg: response.ErrMsg}
	}
	if strings.TrimSpace(response.AccessToken) == "" || response.ExpiresIn <= 0 {
		return "", errors.New("wechat stable token response is incomplete")
	}

	cacheFor := time.Duration(response.ExpiresIn) * time.Second
	if cacheFor > 5*time.Minute {
		cacheFor -= 5 * time.Minute
	} else {
		cacheFor /= 2
	}
	c.accessToken = response.AccessToken
	c.tokenExpiresAt = c.now().Add(cacheFor)
	return c.accessToken, nil
}

func (c *ContentSecurityClient) invalidateAccessToken(token string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.accessToken == token {
		c.accessToken = ""
		c.tokenExpiresAt = time.Time{}
	}
}

func (c *ContentSecurityClient) checkTextWithToken(ctx context.Context, token string, req TextCheckRequest) (*TextCheckResult, error) {
	endpoint, err := url.Parse(c.textCheckURL)
	if err != nil {
		return nil, fmt.Errorf("parse WeChat text-check endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("access_token", token)
	endpoint.RawQuery = query.Encode()

	payload := struct {
		Content  string    `json:"content"`
		Version  int       `json:"version"`
		Scene    TextScene `json:"scene"`
		OpenID   string    `json:"openid"`
		Nickname string    `json:"nickname,omitempty"`
	}{
		Content:  req.Content,
		Version:  2,
		Scene:    req.Scene,
		OpenID:   req.OpenID,
		Nickname: req.Nickname,
	}
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			Suggest string `json:"suggest"`
			Label   int    `json:"label"`
		} `json:"result"`
		TraceID string `json:"trace_id"`
	}
	if err := c.postJSON(ctx, endpoint.String(), payload, &response); err != nil {
		return nil, err
	}
	if response.ErrCode != 0 {
		return nil, &APIError{Code: response.ErrCode, Msg: response.ErrMsg}
	}
	if response.Result.Suggest == "" || response.TraceID == "" {
		return nil, errors.New("wechat text-check response is incomplete")
	}
	return &TextCheckResult{
		Suggest: response.Result.Suggest,
		Label:   response.Result.Label,
		TraceID: response.TraceID,
	}, nil
}

func (c *ContentSecurityClient) postJSON(ctx context.Context, endpoint string, payload, response any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode WeChat request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build WeChat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call WeChat API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, responseBodyLimit))
		return fmt.Errorf("wechat API returned HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, responseBodyLimit))
	if err := decoder.Decode(response); err != nil {
		return fmt.Errorf("decode WeChat response: %w", err)
	}
	return nil
}

func isCredentialError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case 40001, 40014, 42001:
		return true
	default:
		return false
	}
}
