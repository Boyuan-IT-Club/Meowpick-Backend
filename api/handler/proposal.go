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

// CreateProposal 新建一个提案
// @router /api/proposal/add [POST]
func CreateProposal(c *gin.Context) {
	// TODO: not implemented
}

// ListProposals godoc
// @Summary 分页获取提案列表
// @Description 分页查询提案列表数据
// @Tags proposal
// @Produce json
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Success 200 {object} dto.ListProposalResp
// @Router /api/proposal/query [get]
func ListProposals(c *gin.Context) {
	var (
		err  error
		req  dto.ListProposalReq
		resp *dto.ListProposalResp
	)

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.ListProposals(c, &req)
	PostProcess(c, &req, resp, err)

}

// GetProposal 获取提案详情
// @router /api/proposal/:id [GET]
// @Summary 获取提案详情
// @Description 根据提案ID查询提案完整信息
// @Tags proposal
// @Produce json
// @Param id path string true "提案ID"
// @Success 200 {object} dto.GetProposalDetailResp
func GetProposal(c *gin.Context) {
	var (
		err  error
		resp *dto.GetProposalResp
	)

	proposalId := c.Param(consts.CtxProposalID)

	c.Set(consts.CtxUserID, token.GetUserID(c))

	req := &dto.GetProposalReq{
		ProposalID: proposalId,
	}

	resp, err = provider.Get().ProposalService.GetProposal(c, req)
	PostProcess(c, req, resp, err)
}

// ApproveProposal 审批提案
// @router /api/proposal/:id/approve [POST]
func ApproveProposal(c *gin.Context) {
	// TODO: not implemented
}

// UpdateProposal 修改提案
// @router /api/proposal/:id/update [POST]
func UpdateProposal(c *gin.Context) {
	// TODO: not implemented
}

// DeleteProposal 删除提案
// @router /api/proposal/:id/delete [POST]
func DeleteProposal(c *gin.Context) {
	// TODO: not implemented
}

// GetProposalSuggestions 获取提案搜索建议
// @router /api/proposal/suggest [GET]
func GetProposalSuggestions(c *gin.Context) {
	// TODO: not implemented
}

// ToggleProposal 翻转投票状态
// @Summary 投票或取消投票
// @Description 对提案进行投票或取消投票
// @Tags proposal
// @Produce json
// @Param id path string true "提案ID"
// @Success 200 {object} dto.ToggleProposalResp
// @Router /api/proposal/{id} [post]
func ToggleProposal(c *gin.Context) {
	var req dto.ToggleProposalReq
	var resp *dto.ToggleProposalResp
	var err error

	req.TargetID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.ToggleProposal(c, &req)
	PostProcess(c, req, resp, err)
}
