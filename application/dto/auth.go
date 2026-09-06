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
	AuthID     string `json:"authId" binding:"required"`     // 客户端认证标识，当前版本为兼容字段且仍需传入
	AuthType   string `json:"authType" binding:"required"`   // 认证类型标识，当前登录流程使用微信临时登录凭证
	VerifyCode string `json:"verifyCode" binding:"required"` // 微信登录临时 code，用于换取 OpenID，不能为空
}

// SignInResp 返回给前端的响应 包含了accessToken
type SignInResp struct {
	*Resp
	AccessToken string `json:"accessToken"` // Bearer 访问令牌；已有令牌仍有效且无需续期时可能原样返回
	ExpiresIn   int64  `json:"expiresIn"`   // 访问令牌剩余有效秒数
	UserID      string `json:"userId"`      // 当前用户ID；首次登录时自动创建用户
	IsAdmin     bool   `json:"isAdmin"`     // 当前用户是否具有管理员权限
}

// IsAdminResp 判断用户是否是管理员
type IsAdminResp struct {
	*Resp
	IsAdmin bool `json:"isAdmin"` // 当前登录用户是否为管理员
}

// GrantAdminReq 授予管理员权限的请求体
type GrantAdminReq struct {
	UserID     string `json:"userId"`     // 要切换管理员状态的目标用户ID
	VerifyCode string `json:"verifyCode"` // 管理员授权密钥，必须与服务端配置完全一致
}

// GrantAdminResp 授予管理员权限的响应体
type GrantAdminResp struct {
	*Resp
	IsAdmin bool `json:"isAdmin"` // 切换后的管理员状态；true 表示已授予，false 表示已撤销
}
