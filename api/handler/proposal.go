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

// CreateProposal godoc
// @Summary 新增提案
// @Description 创建一个新的提案
// @Tags proposal
// @Accept json
// @Param req body dto.CreateProposalReq true "创建提案的请求参数"
// @success 200 {object} dto.CreateProposalResp
// @Router /api/proposal/add [post]
func CreateProposal(c *gin.Context) {
	var req dto.CreateProposalReq
	var resp *dto.CreateProposalResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.CreateProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListProposals godoc
// @Summary 分页获取提案列表
// @Description 分页查询提案列表数据
// @Tags proposal
// @Produce json
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Success 200 {object} dto.ListProposalResp
// @Router /api/proposal/list [get]
func ListProposals(c *gin.Context) {
	var req dto.ListProposalReq
	var resp *dto.ListProposalResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.ListProposals(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetProposal 获取提案详情
// @Summary 获取提案详情
// @Description 根据提案ID查询提案完整信息
// @Tags proposal
// @Produce json
// @Param id path string true "提案ID"
// @Success 200 {object} dto.GetProposalResp
// @Router /api/proposal/{proposalId} [get]
func GetProposal(c *gin.Context) {
	var req dto.GetProposalReq
	var resp *dto.GetProposalResp
	var err error

	req.ProposalID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.GetProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// ApproveProposal 审批提案
// @router /api/proposal/{proposalId}/approve [POST]
func ApproveProposal(c *gin.Context) {
	// TODO: not implemented
}

// UpdateProposal 更新提案接口
// @Summary 更新提案内容
// @Description 根据提案ID修改提案的标题和内容
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "提案唯一ID"
// @Param body body dto.UpdateProposalReq true "更新参数（标题、内容）"
// @Success 200 {object} dto.UpdateProposalResp "更新成功响应"
// @Router /api/proposal/{proposalId}/update [post]
func UpdateProposal(c *gin.Context) {
	var req dto.UpdateProposalReq
	var resp *dto.UpdateProposalResp
	var err error

	req.ProposalID = c.Param(consts.CtxProposalID)

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.UpdateProposal(c, &req)

	PostProcess(c, &req, resp, err)
}

// DeleteProposal godoc
// @Summary 删除提案
// @Description 根据提案ID软删除提案（标记为已删除状态）
// @Tags proposal
// @Accept json
// @Param proposalId path string true "提案ID"
// @success 200 {object} dto.DeleteProposalResp
// @Router /api/proposal/{proposalId}/delete [POST]
func DeleteProposal(c *gin.Context) {
	var err error
	var req dto.DeleteProposalReq
	var resp *dto.DeleteProposalResp

	req.ProposalID = c.Param(consts.CtxProposalID)

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.DeleteProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetProposalSuggestions godoc
// @Summary 获取提案搜索建议
// @Description 根据关键词模糊分页搜索提案标题，返回匹配的提案建议列表
// @Tags proposal
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码" default(0)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.GetProposalSuggestionsResp
// @Router /api/proposal/suggest [post]
func GetProposalSuggestions(c *gin.Context) {
	var req dto.GetProposalSuggestionsReq
	var resp *dto.GetProposalSuggestionsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.GetProposalSuggestions(c, &req)
	PostProcess(c, &req, resp, err)
}
