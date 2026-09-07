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
	"errors"
	"io"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// CreateProposal godoc
// @Summary 新增提案
// @Description 登录用户创建待审核课程提案。标题、课程名称、课程开课院系、课程分类及至少一个已有校区必填，课程代码和补充说明可空；教师列表可为空，教师项姓名必填，id 有值时复用已有教师、为空时审批通过后创建新教师，教师 department 可为空且不会用课程开课院系推断。系统还会检查同课程重复提案、已有课程及当前用户每日额度
// @Tags proposal
// @Accept json
// @Param req body dto.CreateProposalReq true "创建提案的请求参数"
// @success 200 {object} Response[dto.CreateProposalResp]
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
// @Description 登录后分页查询提案。管理员可按 status 查询 pending、approved、rejected，不传 status 时查询全部；普通用户无论传什么 status 都只返回 approved。返回的 contribution 仅提案创建者可见，其他用户看到 -1
// @Tags proposal
// @Produce json
// @Param status query string false "提案状态：pending/approved/rejected；管理员不传时查询全部，普通用户传值会被忽略"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.ListProposalResp]
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

// FilterProposals godoc
// @Summary 分页筛选提案列表
// @Description 登录后按状态、校区、课程开课院系和课程分类组合筛选提案。status 与 campus 支持重复 query 参数或 JSON 数组字符串；普通用户的状态条件固定为 approved。校区必须是系统已有名称，院系和分类按名称精确匹配；所有条件均为空时管理员查询全部、普通用户查询全部已通过提案
// @Tags proposal
// @Produce json
// @Param status query []string false "提案状态，可多选，不传则不按状态过滤" collectionFormat(multi)
// @Param campus query []string false "校区，可多选，不传则不按校区过滤" collectionFormat(multi)
// @Param department query string false "开课院系，精确匹配"
// @Param category query string false "课程分类，精确匹配"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.ListProposalResp]
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
// @Description 登录后根据提案ID查询完整信息及当前用户点赞状态。已删除提案仅创建者本人可见；contribution 仅创建者可见。已通过提案的 finalCourse 仅创建者或管理员可见，关联正式课程已删除或查询失败时省略
// @Tags proposal
// @Produce json
// @Param proposalId path string true "提案ID"
// @Success 200 {object} Response[dto.GetProposalResp]
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
// @Description 管理员将 pending 提案审批为 approved，并在同一数据库事务内处理映射、教师、正式课程、贡献值及操作日志。finalCourse 可省略、传 null，或直接使用空请求体，此时以用户原始 course 为准；传入时以管理员确认内容创建或恢复正式课程，但不覆盖提案原文。已有教师传 id 后直接复用；id 为空则创建新教师，department 可为空并保存为未维护状态，不创建空院系映射，也不从课程开课院系推断
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "提案ID"
// @Param req body dto.ToggleProposalReq false "可选审批参数；finalCourse 省略或为 null 时使用提案原始课程"
// @Success 200 {object} Response[dto.ToggleProposalResp]
// @Router /api/proposal/{proposalId}/approve [post]
func ApproveProposal(c *gin.Context) {
	var req dto.ToggleProposalReq
	var resp *dto.ToggleProposalResp
	var err error

	// finalCourse 为可选参数，兼容无请求体的历史调用（空 body 视为未传入）
	if err = c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		PostProcess(c, &req, nil, err)
		return
	}
	req.ProposalID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.ApproveProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// RevokeProposal godoc
// @Summary 撤回提案操作
// @Description 管理员把已通过或已拒绝提案恢复为 pending。actionType=approve 仅适用于 approved：事务内软删除提案关联课程和评论、删除相关评论点赞、扣回已结算贡献值，并清理无引用的教师、分类和院系；教师仍被未删除课程或提案引用时保留，分类和院系仍被未删除课程或其他未删除提案引用时保留，院系被正式教师引用时也保留。actionType=reject 仅适用于 rejected：清空拒绝理由。状态与 actionType 不匹配时拒绝操作
// @Tags proposal
// @Accept json
// @Param proposalId path string true "提案ID"
// @Param req body dto.RevokeProposalReq true "撤回类型：approve 撤回通过，reject 撤回拒绝"
// @Success 200 {object} Response[dto.RevokeProposalResp]
// @Router /api/proposal/{proposalId}/revoke [post]
func RevokeProposal(c *gin.Context) {
	var req dto.RevokeProposalReq
	var resp *dto.RevokeProposalResp
	var err error

	req.ProposalID = c.Param(consts.CtxProposalID)

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.RevokeProposal(c, &req)
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
// @Accept json
// @Produce json
// @Param proposalId path string true "提案ID"
// @Param body body dto.RejectProposalReq true "拒绝参数（可选理由）"
// @Success 200 {object} Response[dto.RejectProposalResp]
// @Router /api/proposal/{proposalId}/reject [post]
func RejectProposal(c *gin.Context) {
	var req dto.RejectProposalReq
	var resp *dto.RejectProposalResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	req.ProposalID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.RejectProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// UpdateProposal 更新提案接口
// @Summary 更新提案内容
// @Description 管理员更新 pending 提案的标题、补充说明和完整课程信息；已通过、已拒绝或已删除提案不能更新。请求必须提交完整 course，教师规则与创建提案一致：姓名必填，id 为空表示新教师，department 可为空
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "提案唯一ID"
// @Param body body dto.UpdateProposalReq true "完整更新参数；title、content、course 必填"
// @Success 200 {object} Response[dto.UpdateProposalResp] "更新成功响应"
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
// @Description 登录用户按 path 中的提案ID执行软删除，请求体可为空。仅提案创建者本人可删除自己的 pending 或 rejected 提案，管理员也不能代删；approved 提案不能删除。软删除后创建者仍可在个人提案历史中查看
// @Tags proposal
// @Accept json
// @Param proposalId path string true "提案ID"
// @success 200 {object} Response[dto.DeleteProposalResp]
// @Router /api/proposal/{proposalId}/delete [POST]
func DeleteProposal(c *gin.Context) {
	var err error
	var req dto.DeleteProposalReq
	var resp *dto.DeleteProposalResp

	req.ProposalID = c.Param(consts.CtxProposalID)

	if err = c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.DeleteProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetProposalSuggestions godoc
// @Summary 获取提案搜索建议
// @Description 登录后根据关键词模糊分页搜索未删除且已通过的提案标题，返回提案ID和标题；无结果时返回空数组
// @Tags proposal
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.GetProposalSuggestionsResp]
// @Router /api/proposal/suggest [get]
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
// @Description 登录后获取提案表单字段建议。department、category、campus 从当前映射名称中匹配；courseName、courseCode 从未删除课程中分页搜索；teacherName 从教师中分页搜索，每位教师额外返回其按创建时间倒序的最多两门未删除课程，未教授课程时 courses 返回空数组。未知 field 返回无效字段错误
// @Tags proposal
// @Produce json
// @Param field query string true "字段类型: department/category/campus/courseName/courseCode/teacherName"
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} dto.GetProposalFieldSuggestionsResp
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

// GetMyProposals godoc
// @Summary 获取我的提案
// @Description 登录后获取当前用户的完整提案历史，包含 pending、approved、rejected 及已软删除提案，按创建时间倒序分页。updatedAt 是最近一次编辑或审批时间；已通过提案附带关联正式课程 finalCourse，课程查询失败或已删除时省略。此接口中 contribution 对创建者可见
// @Tags proposal
// @Produce json
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.GetMyProposalsResp]
// @Router /api/proposal/history [get]
func GetMyProposals(c *gin.Context) {
	var req dto.GetMyProposalsReq
	var resp *dto.GetMyProposalsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.GetMyProposals(c, &req)
	PostProcess(c, &req, resp, err)
}
