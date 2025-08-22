package cmd

// SignInRequest 前端传来的登录请求
type SignInRequest struct {
	AuthID   string `json:"authId" binding:"required"`     // 微信开放平台ID
	AuthType string `json:"authType" binding:"required"`   // 认证类型(wechat/phone等)
	Code     string `json:"verifyCode" binding:"required"` // res.code
}

// SignInResponse 返回给前端的响应 包含了accessToken
type SignInResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	UserID      string `json:"userId"`
}
