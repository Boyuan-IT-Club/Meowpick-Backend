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

package dto

// SignInReq 前端传来的登录请求
type SignInReq struct {
	AuthID     string `json:"authId" binding:"required"`     // 微信开放平台ID
	AuthType   string `json:"authType" binding:"required"`   // 认证类型(wechat/phone等)
	VerifyCode string `json:"verifyCode" binding:"required"` // res.code
}

// SignInResp 返回给前端的响应 包含了accessToken
type SignInResp struct {
	*Resp
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	UserID      string `json:"userId"`
	IsAdmin     bool   `json:"isAdmin"`
}

// IsAdminResp 判断用户是否是管理员
type IsAdminResp struct {
	*Resp
	IsAdmin bool `json:"isAdmin"`
}

// GrantAdminReq 授予管理员权限的请求体
type GrantAdminReq struct {
	UserID     string `json:"userId"`
	VerifyCode string `json:"verifyCode"`
}

// GrantAdminResp 授予管理员权限的响应体
type GrantAdminResp struct {
	*Resp
}

// IsAdminResp 判断用户是否是管理员
type IsAdminResp struct {
	*Resp
	IsAdmin bool `json:"isAdmin"`
}
