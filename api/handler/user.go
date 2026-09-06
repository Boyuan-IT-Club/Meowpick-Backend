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
// @Description 返回当前登录用户的昵称、头像引用、累计贡献值、按中国时区自然日统计的今日提案使用量、按贡献值等级计算的每日上限，以及当前能否设置或修改非空昵称。昵称未设置和头像未设置均返回空字符串
// @Tags user
// @Produce json
// @Success 200 {object} Response[dto.GetUserProfileResp]
// @Router /api/user/profile [get]
func GetUserProfile(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().UserService.GetUserProfile(c)
	PostProcess(c, nil, resp, err)
}

// GetUsernameByUserID godoc
// @Summary 根据用户 ID 获取昵称
// @Description 登录后查询指定用户昵称。查询本人或管理员查询任意用户时无需 proposalId；普通用户跨用户查询必须提供属于目标用户且 showUsername=true 的未删除提案ID，否则拒绝，避免绕过匿名提案设置。目标用户尚未设置昵称时成功返回空字符串
// @Tags user
// @Produce json
// @Param userId path string true "用户 ID"
// @Param proposalId query string false "跨用户查询时必填，且该提案必须允许展示昵称；本人和管理员可省略"
// @Success 200 {object} Response[dto.GetUsernameByUserIDResp]
// @Router /api/user/{userId}/username [get]
func GetUsernameByUserID(c *gin.Context) {
	var req dto.GetUsernameByUserIDReq
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().UserService.GetUsernameByUserID(c, c.Param("userId"), req.ProposalID)
	PostProcess(c, nil, resp, err)
}

// UpdateUserProfile godoc
// @Summary 更新当前用户资料
// @Description 原子更新当前登录用户的昵称和头像，任一字段校验或写入失败时两者都不改变。字段省略或为 null 表示保持原值，空字符串表示清空。非空昵称去除首尾空白后最多15个 Unicode 字符，不允许控制字符、不得与其他用户昵称重复，设置或改名后30天内不能再次设置非空昵称；清空昵称不受冷却限制。头像仅保存引用字符串，不负责上传、下载或 URL 格式校验
// @Tags user
// @Accept json
// @Produce json
// @Param body body dto.UpdateUserProfileReq true "昵称和头像的局部更新；字段省略/null 保持，空字符串清空"
// @Success 200 {object} Response[dto.UpdateUserProfileResp]
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
