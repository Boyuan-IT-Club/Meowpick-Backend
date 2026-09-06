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
// @Description 仅管理员可用。按可选目标类型和日志正文关键词分页查询全部变更记录，type 支持 course、proposal、teacher、user，keyword 使用大小写不敏感模糊匹配；两者为空时查询全部，结果按时间倒序
// @Tags changeLog
// @Accept json
// @Produce json
// @Param req body dto.ListChangeLogsReq true "可选类型、内容关键词和分页参数"
// @Success 200 {object} Response[dto.ListChangeLogsResp]
// @Router /api/changelog/list [post]
func ListChangeLogs(c *gin.Context) {
	var req dto.ListChangeLogsReq
	var resp *dto.ListChangeLogsResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ChangeLogService.ListChangeLogs(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListProposalLogsGrouped 按提案聚合的日志列表
// @Summary 按提案聚合的日志列表
// @Description 仅管理员可用。以提案为分页单位返回提案原始内容、课程、创建者信息和最近一次管理员操作；同一提案存在多次审核、撤回或更新日志时 adminAction 只保留时间最新的一条，尚无管理员操作时省略
// @Tags changeLog
// @Accept json
// @Produce json
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，默认20；大于100按10处理" default(20)
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
// @Description 仅管理员可用。以每次独立操作为一条记录分页返回全部变更日志，严格按操作时间倒序，包含操作者、标准化动作名、可选关联提案当前摘要及日志正文；管理员权限变更等非提案操作可能没有 proposalId 和 proposalSnapshot
// @Tags changeLog
// @Accept json
// @Produce json
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，默认20；大于100按10处理" default(20)
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
