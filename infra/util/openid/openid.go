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

package openid

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const wechatAPI = "https://api.weixin.qq.com/sns/jscode2session"

type WeChatSessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// GetWeChatOpenID 通过code获取openid
func GetWeChatOpenID(appID, appSecret, code string) string {
	params := url.Values{}
	params.Add("appid", appID)
	params.Add("secret", appSecret)
	params.Add("js_code", code)
	params.Add("grant_type", "authorization_code")

	resp, err := http.Get(wechatAPI + "?" + params.Encode())
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var sessionResp WeChatSessionResp
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return ""
	}

	if sessionResp.ErrCode != 0 {
		return ""
	}

	return sessionResp.OpenID
}
