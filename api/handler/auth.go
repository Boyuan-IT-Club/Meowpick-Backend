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

package handler

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// SignIn godoc
// @Summary 登录
// @Description 使用微信临时登录 code 换取 OpenID，查找或首次创建用户并返回 Bearer 访问令牌。请求若同时携带仍有效、属于同一用户且尚无需续期的令牌，服务端会复用原令牌；否则签发新令牌。authId 和 authType 当前为兼容必填字段，实际身份以 verifyCode 换取的 OpenID 为准
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.SignInReq true "登录参数；authId、authType、verifyCode 均必填"
// @Success 200 {object} Response[dto.SignInResp]
// @Security
// @Router /api/auth/sign_in [post]
func SignIn(c *gin.Context) {
	var err error
	var req dto.SignInReq
	var resp *dto.SignInResp

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	tokenStr, _ := token.ExtractToken(c.Request.Header)
	c.Set(consts.CtxToken, tokenStr)

	resp, err = provider.Get().AuthService.SignIn(c, &req)
	PostProcess(c, &req, resp, err)
}

// IsAdmin godoc
// @Summary 是否管理员
// @Description 校验 Bearer 登录态并返回当前用户实时管理员状态；用户不存在或查询失败时返回用户查询错误
// @Tags auth
// @Produce json
// @Success 200 {object} Response[dto.IsAdminResp]
// @Router /api/auth/is_admin [get]
func IsAdmin(c *gin.Context) {
	var err error
	var resp *dto.IsAdminResp

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().AuthService.IsAdmin(c)
	PostProcess(c, nil, resp, err)
}

// GrantAdmin godoc
// @Summary 切换管理员权限
// @Description 登录用户凭服务端配置的管理员授权密钥切换指定用户的管理员状态：普通用户变为管理员，管理员再次操作则撤销权限。目标用户必须存在；操作会写入管理员变更日志。该接口不是“只授予不撤销”，响应 isAdmin 表示切换后的状态
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.GrantAdminReq true "目标用户ID和管理员授权密钥"
// @Success 200 {object} Response[dto.GrantAdminResp]
// @Router /api/auth/grant_admin [post]
func GrantAdmin(c *gin.Context) {
	var err error
	var req dto.GrantAdminReq
	var resp *dto.GrantAdminResp

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().AuthService.GrantAdmin(c, &req)
	PostProcess(c, &req, resp, err)
}
