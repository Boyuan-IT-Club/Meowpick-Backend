package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func GetWXOpenID(appId, secret, code string) (string, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appId, secret, code,
	)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		OpenID string `json:"openid"`
		ErrMsg string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.OpenID == "" {
		return "", fmt.Errorf("get openid failed: %s", result.ErrMsg)
	}
	return result.OpenID, nil
}
