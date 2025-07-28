package resp

// 登录时 返回给前端的包含AccessToken的response
type LoginResp struct {
	AccessToken string `json:"accessToken"` // 前端需要的字段
	// TODO 完善字段
}
