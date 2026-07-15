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

// CreateProposal godoc
// @Summary 新增提案
// @Description 创建一个新的提案
// @Description 提案创建完成后，管理员能够通过分页筛选接口从提案列表中获取提案
// @Tags proposal
// @Accept json
// @Param req body dto.CreateProposalReq true "创建提案的请求参数"
// @success 200 {object} Response[dto.CreateProposalResp]
// @Security Bearer
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

// FilterProposals godoc
// @Summary 分页筛选提案列表
// @Description 基于提案状态、校区、开课院系、课程分类筛选 proposal 表中的提案
// @Description 原本的提案列表接口 /list 已被整合进本接口，**所有发往 /list 的前端请求需要改向此处**
// @Description (可筛选的) 提案列表功能不对用户开放，仅管理员可见
// @Tags proposal
// @Produce json
// @Param status query []string false "提案状态，可多选" collectionFormat(multi)
// @Param campus query []string false "校区，可多选" collectionFormat(multi)
// @Param department query string false "开课院系，精确匹配"
// @Param category query string false "课程分类，精确匹配"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} Response[dto.ListProposalResp]
// @Security Bearer
// @Router /api/proposal/filter [get]
func FilterProposals(c *gin.Context) {
	var req dto.FilterProposalReq
	var resp *dto.ListProposalResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.FilterProposals(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetProposal 获取提案详情
// @Summary 获取提案详情
// @Description 根据提案ID查询提案完整信息（如果筛选接口**执意**返回所有信息，那这个接口也可以删）
// @Tags proposal
// @Produce json
// @Param id path string true "提案ID"
// @Success 200 {object} Response[dto.GetProposalResp]
// @Security Bearer
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

// ApproveProposal godoc
// @Summary 审批提案
// @Description 管理员通过提案并创建课程，创建的课程是经管理员修改后的版本
// @Description 原始提案将被标注为 Approved 并重新写入数据库，方便追溯
// @Tags proposal
// @Produce json
// @Param proposalId path string true "提案ID"
// @Success 200 {object} Response[dto.ToggleProposalResp]
// @Security Bearer
// @Router /api/proposal/{proposalId}/approve [post]
// @Accept json
// @Param req body dto.ToggleProposalReq true "审批通过参数"
func ApproveProposal(c *gin.Context) {
	// Repeated code with RejectProposal
	var req dto.ToggleProposalReq
	var resp *dto.ToggleProposalResp
	var err error
	// 读取 JSON
	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	// Read the ProposalID in path
	req.ProposalID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.ApproveProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// RejectProposal godoc
// @Summary 拒绝提案
// @Description 管理员操作：将状态为 pending（待审核）的提案变更为 rejected（已拒绝）
// @Description 使用场景：课程提案审核流程中，管理员认为提案不符合要求，驳回该提案
// @Description 注意事项：
// @Description - 仅管理员可操作（需先调用 /api/auth/is_admin 确认权限）
// @Description - 仅状态为 pending 的提案可以拒绝，已 approved/rejected 的提案无法再次操作
// @Description - 拒绝后不会创建课程记录，仅更新提案状态
// @Tags proposal
// @Produce json
// @Param proposalId path string true "提案ID"
// @Success 200 {object} Response[dto.RejectProposalResp]
// @Security Bearer
// @Router /api/proposal/{proposalId}/reject [post]
func RejectProposal(c *gin.Context) {
	var req dto.RejectProposalReq
	var resp *dto.RejectProposalResp
	var err error
	// 不读取 Json 只读取 Path
	req.ProposalID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.RejectProposal(c, &req)
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
// @Success 200 {object} Response[dto.GetProposalSuggestionsResp]
// @Security Bearer
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

// GetProposalFieldSuggestions godoc
// @Summary 获取提案字段建议
// @Description 根据字段类型和关键词获取建议列表，支持学院、类别、校区、课程名称、课程代码、教师姓名
// @Tags proposal
// @Produce json
// @Param field query string true "字段类型: department/category/campus/courseName/courseCode/teacherName"
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码" default(0)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.GetProposalFieldSuggestionsResp
// @Security Bearer
// @Router /api/proposal/field-suggestions [get]
func GetProposalFieldSuggestions(c *gin.Context) {
	var req dto.GetProposalFieldSuggestionsReq
	var resp *dto.GetProposalFieldSuggestionsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.GetProposalFieldSuggestions(c, &req)
	PostProcess(c, &req, resp, err)
}
