package util

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const wechatAPI = "https://api.weixin.qq.com/sns/jscode2session"

type WeChatSessionResponse struct {
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
	var sessionResp WeChatSessionResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return ""
	}

	if sessionResp.ErrCode != 0 {
		return ""
	}

	return sessionResp.OpenID
}
