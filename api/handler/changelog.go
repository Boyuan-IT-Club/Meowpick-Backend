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

// ListChangeLogs godoc
// @Summary 分页查询变更记录
// @Description 按目标类型+ID分页查询变更记录
// @Tags changelog
// @Accept json
// @Produce json
// @Param req body dto.ListChangeLogReq true "查询参数"
// @Success 200 {object} Response[dto.ListChangelogResp]
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

// ListProposalLogsGrouped 按提案聚合的日志列表
// @Summary 按提案聚合的日志列表
// @Description 以提案为维度的分页列表，包含提案基础信息、提议者信息、审核操作信息
// @Tags ChangeLog
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} Response[dto.ListProposalLogsGroupedResp]
// @Router /api/changelog/proposal/grouped [get]
func ListProposalLogsGrouped(c *gin.Context) {
	var err error
	var req dto.ListProposalLogsGroupedReq
	var resp *dto.ListProposalLogsGroupedResp

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ChangeLogService.ListProposalLogsGrouped(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListProposalLogsTimeline 扁平化时间线日志
// @Summary 扁平化时间线日志
// @Description 一条记录代表一次独立动作的扁平化分页，严格按时间倒序排列
// @Tags ChangeLog
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} Response[dto.ListProposalLogsTimelineResp]
// @Router /api/changelog/proposal/timeline [get]
func ListProposalLogsTimeline(c *gin.Context) {
	var err error
	var req dto.ListProposalLogsTimelineReq
	var resp *dto.ListProposalLogsTimelineResp

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ChangeLogService.ListProposalLogsTimeline(c, &req)
	PostProcess(c, &req, resp, err)
}
