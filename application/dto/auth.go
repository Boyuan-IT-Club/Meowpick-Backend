package dto

// SignInReq 前端传来的登录请求
type SignInReq struct {
	AuthID   string `json:"authId" binding:"required"`     // 微信开放平台ID
	AuthType string `json:"authType" binding:"required"`   // 认证类型(wechat/phone等)
	Code     string `json:"verifyCode" binding:"required"` // res.code
}

// SignInResp 返回给前端的响应 包含了accessToken
type SignInResp struct {
	*Resp
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	UserID      string `json:"userId"`
}
