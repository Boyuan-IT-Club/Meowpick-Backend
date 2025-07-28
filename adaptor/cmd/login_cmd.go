package cmd

type LoginCMD struct {
	// 前端传来的登录请求
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	OpenID string `json:"open_id"`
	// TODO 完善字段
}
