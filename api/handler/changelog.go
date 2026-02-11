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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// ListChangeLogs godoc
// @Summary 分页查询变更记录
// @Description 按目标类型+ID分页查询变更记录
// @Tags changelog
// @Accept json
// @Produce json
// @Param req body dto.ListChangeLogReq true "查询参数"
// @Success 200 {object} dto.ListChangeLogResp
// @Router /api/changelog/list [post]
func ListChangeLogs(c *gin.Context) {
	var req dto.ListChangeLogReq
	var resp *dto.ListChangeLogResp
	var err error

	// 绑定参数
	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	// 设置上下文用户ID
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ChangeLogService.ListChangeLogs(c, &req)
	PostProcess(c, &req, resp, err)
}

// CreateChangeLog godoc
// @Summary 新增变更记录
// @Description 创建一条新的变更记录
// @Tags changelog
// @Accept json
// @Produce json
// @Param req body dto.CreateChangeLogReq true "创建参数"
// @Success 200 {object} dto.CreateChangeLogResp
// @Router /api/changelog/add [post]
func CreateChangeLog(c *gin.Context) {
	var req dto.CreateChangeLogReq
	var resp *dto.CreateChangeLogResp
	var err error

	// 绑定参数
	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	// 设置上下文用户ID
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ChangeLogService.CreateChangeLog(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetChangeLog godoc
// @Summary 获取变更记录详情
// @Description 根据变更记录ID查询详情
// @Tags changelog
// @Produce json
// @Param id path string true "变更记录ID"
// @Success 200 {object} dto.GetChangeLogResp
// @Router /api/changelog/{changeLogId} [get]
func GetChangeLog(c *gin.Context) {
	// TODO:
}
