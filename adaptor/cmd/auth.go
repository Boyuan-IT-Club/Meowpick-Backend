package cmd

// SignInRequest 前端传来的登录请求
type SignInRequest struct {
	AuthID   string `json:"authId" binding:"required"`    // 微信开放平台ID
	AuthType string `json:"authType" binding:"required"`  // 认证类型(wechat/phone等)
	OpenID   string `json:"wx-openid" binding:"required"` // 微信openID
	//AppID      int    `json:"appId" binding:"required"`      // 应用ID 用于区分多个应用 暂未使用
}

// SignInResponse 返回给前端的响应 包含了accessToken
type SignInResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
}
