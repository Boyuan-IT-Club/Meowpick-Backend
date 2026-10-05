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
// @Description 登录用户创建create_course/update_course/update_teacher三类待审提案。缺失type兼容create_course，新请求不使用title；新增提交完整course，修改提交targetId和suggested，suggested只含拟改字段，后端保存可信原值。课程名称、代码、开课院系、分类及至少一个已有校区必填；已有教师按ID引用，新教师在审批后创建。教师修改只允许name/title/department，title和department显式空字符串可清空；缺席字段不改。校区和教师列表提交完整无序新列表。无实际变化立即返回108000029，不创建提案。不同作者的相同待审建议允许提交并返回pendingDuplicateIds；同一作者重复返回108000008。三类共用UTC+8每日额度。
// @Summary 新增提案
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

// ResubmitProposal godoc
// @Description 作者重新提交自己未删除的rejected提案；使用与add相同的三类输入。事务内删除旧提案、创建新的pending提案并检查共用额度，任一步失败全部回滚。
// @Summary 重新提交被拒绝提案
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "原拒绝提案ID"
// @Param req body dto.ResubmitProposalReq true "重新提交后的完整提案内容"
// @Success 200 {object} Response[dto.ResubmitProposalResp]
// @Router /api/proposal/{proposalId}/resubmit [post]
func ResubmitProposal(c *gin.Context) {
	var req dto.ResubmitProposalReq
	var resp *dto.ResubmitProposalResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	req.ProposalID = c.Param(consts.CtxProposalID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.ResubmitProposal(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListProposals godoc
// @Description 登录分页查询三类提案，按type、displayName、targetId、suggested/before/final或course/finalCourse展示。旧记录缺失type返回create_course。管理员可按status筛选全部状态，普通用户仅approved。匿名提案对其他普通用户不返回userId，作者及管理员可见；contribution仍仅作者可见，其他查看者为-1。Final值为管理员确认的不可变快照。
// @Summary 分页获取提案列表
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

// ListPendingProposals godoc
// @Description 所有登录用户分页查看未删除pending提案，包含三种type。匿名作者ID对其他普通用户隐藏，作者和管理员可见；contribution仅作者可见。
// @Summary 分页获取待审核提案
// @Tags proposal
// @Produce json
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.ListProposalResp]
// @Router /api/proposal/pending [get]
func ListPendingProposals(c *gin.Context) {
	var req dto.PageParam
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().ProposalService.ListPendingProposals(c, &req)
	PostProcess(c, &req, resp, err)
}

// SuggestProposals godoc
// @Description 按displayName、课程名、旧title进行普通文本大小写不敏感模糊搜索；type可选create_course/update_course/update_teacher，status和campus支持多选。管理员可检索所有状态及疑似待审课程，普通用户固定approved。校区、院系和分类兼容课程原建议及修改字段。匿名身份权限与列表相同。
// @Summary 分页搜索并筛选提案列表
// @Tags proposal
// @Produce json
// @Param keyword query string false "课程或教师显示名普通文本模糊搜索关键词，正则特殊字符按字面值处理"
// @Param status query []string false "提案状态，可多选，不传则不按状态过滤" collectionFormat(multi)
// @Param campus query []string false "校区，可多选，不传则不按校区过滤" collectionFormat(multi)
// @Param department query string false "开课院系，精确匹配"
// @Param category query string false "课程分类，精确匹配"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.ListProposalResp]
// @Param type query string false "create_course/update_course/update_teacher"
// @Router /api/proposal/suggest [get]
func SuggestProposals(c *gin.Context) {
	var req dto.SuggestProposalReq
	var resp *dto.ListProposalResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.SuggestProposals(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetProposal 获取提案详情
// @Description 登录获取三类提案详情。approved所有登录用户可见，pending/rejected仅作者和管理员可见；已删除仅作者可见。匿名作者ID对其他普通用户隐藏，contribution仅作者可见。原建议保持不变，final/finalCourse为管理员最终确认的快照，decisionBatchId关联共同审批。
// @Summary 获取提案详情
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
// @Description 管理员确认审批，必须先POST preview，再使用相同参数及previewToken调用本接口。未预览108000030、资料或候选变化108000031、未明确重审冲突108000032、无实际变更108000029、未确认新教师108000034、正式课程精确重复108000009。待审新增课程六项一致自动共同通过；手动选择的proposalIds及其自动组也共同通过，统一最终资料，但每份原建议保持不变，分别结算贡献值，数据库只写一次目标资料。贡献值、所有成员状态、资料历史和不可变操作日志同事务提交；失败全部回滚。修改类型使用final调整suggested中的字段，未传采用管理员已保存草稿或作者建议；不会覆盖未修改字段。
// @Summary 审批提案
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "提案ID"
// @Param req body dto.ToggleProposalReq false "可选审批参数；finalCourse/final及previewToken等共同审批参数"
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
// @Description 管理员撤回操作。actionType=approve在本次共同审批后不足24小时内整批撤回，所有成员回pending，贡献值全部扣回，目标课程或教师仅恢复一次；存在尚未撤回的后续目标修改时108000022，满24小时108000028。撤回新增课程软删课程、评论及评论点赞；只清理本批创建且无引用的教师，绝不删除复用教师或其他课程。撤回后重新通过恢复原课程ID。actionType=reject仅撤回当前拒绝提案，无时间限制；批量拒绝也可以逐份撤回。历史无批次数据兼容旧撤回，但缺少教师所有权来源时保留教师。
// @Summary 撤回提案操作
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
// @Description 管理员手动拒绝path提案及请求proposalIds明确勾选的额外pending提案，统一填写reason。不自动扩展拒绝范围，不创建或修改正式资料、不发积分；状态和日志同事务提交。发现已有正式课程时由管理员使用本接口手动拒绝，可批量填写已有课程ID或链接。
// @Summary 拒绝提案
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
// @Description 管理员保存待审自动组的共同最终资料草稿，不改变各作者原始course/suggested/before/content。新增类型传完整course，修改类型传suggested中拟改字段的最终值，不能扩展原建议的字段范围或更换type/targetId。修改草稿缺席字段保留此前管理员草稿值；教师ID、重复教师及校区须校验，失败不保存草稿或日志。草稿同步至自动组全部成员，后续相同待审提案继承草稿，仍需preview确认审批；改动记录逐份保存不可变日志。
// @Summary 更新提案内容
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "提案唯一ID"
// @Param body body dto.UpdateProposalReq true "新增传完整course，修改传suggested最终字段"
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

// GetProposalFieldSuggestions godoc
// @Description 登录获取表单字段建议。courseName按名称去重后按精确/前缀/子串相关度及名称稳定排序再分页，total为唯一名称数；名称只做普通文本匹配，不解释正则。department/category/campus匹配已有映射，courseCode保留代码建议，teacherName按姓名或姓名职称搜索，返回name(value)、title、label及最近两门课程。
// @Summary 获取提案字段建议
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
// @Description 登录查询自己的三类提案历史，包含所有状态和软删除记录，贡献值可见。每份共同审批提案单独保留原建议、管理员最终快照及decisionBatchId，排序和分页沿用原接口。
// @Summary 获取我的提案
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
