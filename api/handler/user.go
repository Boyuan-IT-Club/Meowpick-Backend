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

// GetUserProfile godoc
// @Summary 获取当前用户资料
// @Description 获取当前登录用户的昵称、头像、贡献值、今日已创建提案数、每日提案上限和昵称可编辑状态
// @Tags user
// @Produce json
// @Success 200 {object} Response[dto.GetUserProfileResp]
// @Security Bearer
// @Router /api/user/profile [get]
func GetUserProfile(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().UserService.GetUserProfile(c)
	PostProcess(c, nil, resp, err)
}

// GetUsernameByUserID godoc
// @Summary 根据用户 ID 获取昵称
// @Description 根据用户 ID 获取该用户设置的昵称
// @Tags user
// @Produce json
// @Param userId path string true "用户 ID"
// @Success 200 {object} Response[dto.GetUsernameByUserIDResp]
// @Security Bearer
// @Router /api/user/{userId}/username [get]
func GetUsernameByUserID(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().UserService.GetUsernameByUserID(c, c.Param("userId"))
	PostProcess(c, nil, resp, err)
}

// UpdateUserProfile godoc
// @Summary 更新当前用户资料
// @Description 更新当前登录用户的昵称和头像；字段未传或为 null 时保持原值，空字符串用于清空
// @Tags user
// @Accept json
// @Produce json
// @Param body body dto.UpdateUserProfileReq true "UpdateUserProfileReq"
// @Success 200 {object} Response[dto.UpdateUserProfileResp]
// @Security Bearer
// @Router /api/user/profile/update [post]
func UpdateUserProfile(c *gin.Context) {
	var req dto.UpdateUserProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().UserService.UpdateUserProfile(c, &req)
	PostProcess(c, &req, resp, err)
}
