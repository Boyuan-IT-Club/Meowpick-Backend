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

package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"

	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IProposalService = (*ProposalService)(nil)

type IProposalService interface {
	CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error)
	ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error)
	GetProposal(ctx context.Context, req *dto.GetProposalReq) (*dto.GetProposalResp, error)
	DeleteProposal(ctx context.Context, req *dto.DeleteProposalReq) (*dto.DeleteProposalResp, error)
	UpdateProposal(ctx context.Context, req *dto.UpdateProposalReq) (*dto.UpdateProposalResp, error)
	GetProposalSuggestions(ctx context.Context, req *dto.GetProposalSuggestionsReq) (*dto.GetProposalSuggestionsResp, error)
	GetProposalFieldSuggestions(ctx context.Context, req *dto.GetProposalFieldSuggestionsReq) (*dto.GetProposalFieldSuggestionsResp, error)
	ApproveProposal(ctx context.Context, req *dto.ToggleProposalReq) (*dto.ToggleProposalResp, error)
}

type ProposalService struct {
	CourseRepo        *repo.CourseRepo
	CourseAssembler   *assembler.CourseAssembler
	ProposalRepo      *repo.ProposalRepo
	ProposalAssembler *assembler.ProposalAssembler
	LikeRepo          *repo.LikeRepo
	LikeCache         *cache.LikeCache
	UserRepo          *repo.UserRepo
	TeacherRepo       *repo.TeacherRepo
}

var ProposalServiceSet = wire.NewSet(
	wire.Struct(new(ProposalService), "*"),
	wire.Bind(new(IProposalService), new(*ProposalService)),
)

// CreateProposal 添加一个新的课程提案
func (s *ProposalService) CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 转换为 courseModel
	req.Course.ID = primitive.NewObjectID().Hex()
	course, err := s.CourseAssembler.ToCourseDB(ctx, req.Course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "database course"), errorx.KV("dst", "course vo"),
		)
	}

	// 检查是否已经存在相同的提案
	existingProposal, err := s.ProposalRepo.IsCourseInExistingProposals(ctx, course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCourseFindInProposalsFailed,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}

	// 检查是否已经存在相同的课程
	existingCourse, err := s.CourseRepo.IsCourseInExistingCourses(ctx, course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCourseFindInCoursesFailed,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}

	if existingProposal {
		return nil, errorx.New(errno.ErrProposalCourseFoundInProposals,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}
	if existingCourse {
		return nil, errorx.New(errno.ErrProposalCourseFoundInCourses,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}

	// 使用Assembler转换提案
	now := time.Now()
	proposalVO := &dto.ProposalVO{
		ID:        primitive.NewObjectID().Hex(),
		UserID:    userId,
		Title:     req.Title,
		Content:   req.Content,
		Deleted:   false,
		Status:    consts.ProposalStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		// 这里不设置Course，因为上面已经获得过CourseDB了

	}

	proposal, err := s.ProposalAssembler.ToProposalDB(ctx, proposalVO)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "proposal vo"), errorx.KV("dst", "database proposal"),
		)
	}

	// 设置课程，防止重复转换
	proposal.Course = course

	// 保存提案到数据库
	if err = s.ProposalRepo.Insert(ctx, proposal); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCreateFailed, errorx.KV("name", req.Course.Name))
	}

	return &dto.CreateProposalResp{
		Resp:       dto.Success(),
		ProposalID: proposal.ID,
	}, nil
}

// ListProposals 分页查询不同状态的提案，用于投票列表或管理端审核
func (s *ProposalService) ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 获得状态
	status := mapping.Data.GetProposalStatusIDByName(req.Status)

	// 获得提案
	var err error
	var total int64
	var proposals []*model.Proposal
	if status == 0 { // 获取所有
		proposals, total, err = s.ProposalRepo.FindMany(ctx, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[ProposalRepo] [FindMany] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
		}
	} else { // 获取指定状态
		proposals, total, err = s.ProposalRepo.FindManyByStatus(ctx, req.PageParam, status)
		if err != nil {
			logs.CtxErrorf(ctx, "[ProposalRepo] [FindManyByStatus] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
		}
	}

	// 转换为VO
	vos, err := s.ProposalAssembler.ToProposalVOArray(ctx, proposals, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVOArray] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposals"), errorx.KV("dst", "proposal vos"))
	}

	return &dto.ListProposalResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil
}

// GetProposal 获取提案详情
func (s *ProposalService) GetProposal(ctx context.Context, req *dto.GetProposalReq) (*dto.GetProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询提案详情
	proposalId := req.ProposalID
	proposal, err := s.ProposalRepo.FindByID(ctx, proposalId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindByID] error: %v, proposalId: %s", err, proposalId)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed, errorx.KV("proposalId", proposalId))
	}
	if proposal == nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindByID] proposal not found, proposalId: %s", proposalId)
		return nil, errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", proposalId))
	}

	// 转换为VO
	vo, err := s.ProposalAssembler.ToProposalVO(ctx, proposal, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVO] error: %v, proposalId: %s", err, proposalId)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposal"), errorx.KV("dst", "proposal vo"))
	}

	return &dto.GetProposalResp{
		Resp:     dto.Success(),
		Proposal: vo,
	}, nil
}

// DeleteProposal 删除提案
func (s *ProposalService) DeleteProposal(ctx context.Context, req *dto.DeleteProposalReq) (*dto.DeleteProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	proposalId := req.ProposalID

	// 检查提案是否存在
	proposal, err := s.ProposalRepo.FindByID(ctx, proposalId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindByID] error: %v, proposalId: %s", err, proposalId)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
	}
	if proposal == nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindByID] proposal not found, proposalId: %s", proposalId)
		return nil, errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", proposalId))
	}

	//权限检查：非管理员只能删除自己的提案
	if proposal.UserID != userId {
		// 查询用户是否是管理员
		isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
		if err != nil {
			logs.CtxErrorf(ctx, "[UserRepo] [GetByID] error: %v, userId: %s", err, userId)
			return nil, errorx.New(errno.ErrUserNotAdmin,
				errorx.KV("id", userId))
		}

		if !isAdmin {
			return nil, errorx.New(errno.ErrUserNotOwner,
				errorx.KV("id", userId))
		}
	}

	//执行删除提案
	err = s.ProposalRepo.DeleteProposal(ctx, proposalId, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [Delete] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalDeleteFailed,
			errorx.KV("proposal_id", proposalId))
	}

	return &dto.DeleteProposalResp{
		Resp:       dto.Success(),
		ProposalID: req.ProposalID,
		DeletedAt:  time.Now(),
		OperatorID: userId,
		Deleted:    true,
	}, nil
}

// UpdateProposal 更新提案
func (s *ProposalService) UpdateProposal(ctx context.Context, req *dto.UpdateProposalReq) (*dto.UpdateProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	//查询提案
	proposal, err := s.ProposalRepo.FindByID(ctx, req.ProposalID)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindByID] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed, errorx.KV("proposalId", req.ProposalID))
	}
	if proposal == nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindByID] proposal not found, proposalId: %s", req.ProposalID)
		return nil, errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
	}

	//更新提案字段
	proposal.Title = req.Title
	proposal.Content = req.Content
	courseModel, err := s.CourseAssembler.ToCourseDB(ctx, req.Course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "course vo"), errorx.KV("dst", "course model"),
		)
	}
	proposal.Course = courseModel
	proposal.UpdatedAt = time.Now()

	// 执行更新
	if err = s.ProposalRepo.UpdateProposal(ctx, proposal); err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [UpdateProposal] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}

	return &dto.UpdateProposalResp{
		Resp:       dto.Success(),
		ProposalID: proposal.ID,
	}, nil
}

// GetProposalSuggestions 获取提案搜索建议
func (s *ProposalService) GetProposalSuggestions(ctx context.Context, req *dto.GetProposalSuggestionsReq) (*dto.GetProposalSuggestionsResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询提案建议
	proposals, _, err := s.ProposalRepo.GetSuggestionsByTitle(ctx, req.Keyword, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [GetSuggestionsByTitle] error: %v, keyword: %s", err, req.Keyword)
		return nil, errorx.WrapByCode(err, errno.ErrProposalGetSuggestionsFailed,
			errorx.KV("keyword", req.Keyword))
	}

	// 转换为VO
	var vos []*dto.ProposalSuggestionsVO
	for _, proposal := range proposals {
		vos = append(vos, &dto.ProposalSuggestionsVO{
			ID:    proposal.ID,
			Title: proposal.Title,
		})
	}

	return &dto.GetProposalSuggestionsResp{
		Resp:        dto.Success(),
		Suggestions: vos,
	}, nil
}

// GetProposalFieldSuggestions 获取提案字段建议
func (s *ProposalService) GetProposalFieldSuggestions(ctx context.Context, req *dto.GetProposalFieldSuggestionsReq) (*dto.GetProposalFieldSuggestionsResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	suggestions := []*dto.FieldSuggestionVO{}
	var total int64

	// 根据字段类型路由到不同的查询逻辑
	switch req.Field {
	case consts.FieldDepartment:
		// 从映射表模糊匹配学院
		ids := mapping.Data.GetDepartmentIDsByKeyword(req.Keyword)
		for _, id := range ids {
			name := mapping.Data.GetDepartmentNameByID(id)
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:    strconv.Itoa(int(id)),
				Value: name,
				Label: name,
			})
		}
		total = int64(len(suggestions))

	case consts.FieldCategory:
		// 从映射表模糊匹配课程类别
		ids := mapping.Data.GetCategoryIDsByKeyword(req.Keyword)
		for _, id := range ids {
			name := mapping.Data.GetCategoryNameByID(id)
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:    strconv.Itoa(int(id)),
				Value: name,
				Label: name,
			})
		}
		total = int64(len(suggestions))

	case consts.FieldCampus:
		// 从映射表模糊匹配校区
		for id, name := range mapping.Data.CampusNameByID {
			if strings.Contains(strings.ToLower(name), strings.ToLower(req.Keyword)) {
				suggestions = append(suggestions, &dto.FieldSuggestionVO{
					ID:    strconv.Itoa(int(id)),
					Value: name,
					Label: name,
				})
			}
		}
		total = int64(len(suggestions))

	case consts.FieldCourseName:
		// 从数据库查询课程名称
		courses, total, err := s.CourseRepo.GetSuggestionsByName(ctx, req.Keyword, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [GetSuggestionsByName] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}
		for _, course := range courses {
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:    course.ID,
				Value: course.Name,
				Label: course.Name,
			})
		}
		_ = total

	case consts.FieldCourseCode:
		// 从数据库查询课程代码
		courses, total, err := s.CourseRepo.GetSuggestionsByCode(ctx, req.Keyword, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [GetSuggestionsByCode] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}
		for _, course := range courses {
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:    course.ID,
				Value: course.Code,
				Label: course.Code + " - " + course.Name,
			})
		}
		_ = total

	case consts.FieldTeacherName:
		// 从数据库查询教师姓名
		teachers, total, err := s.TeacherRepo.GetSuggestionsByName(ctx, req.Keyword, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[TeacherRepo] [GetSuggestionsByName] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrTeacherGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}
		for _, teacher := range teachers {
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:    teacher.ID,
				Value: teacher.Name,
				Label: teacher.Name + " - " + teacher.Title,
			})
		}
		_ = total

	default:
		logs.CtxErrorf(ctx, "[ProposalService] [GetProposalFieldSuggestions] invalid field: %s", req.Field)
		return nil, errorx.New(errno.ErrProposalInvalidField,
			errorx.KV("field", req.Field))
	}

	return &dto.GetProposalFieldSuggestionsResp{
		Resp:        dto.Success(),
		Field:       req.Field,
		Suggestions: suggestions,
		Total:       total,
	}, nil
}

// GetMyProposals 获取我的提案
func (s *ProposalService) GetMyProposals(ctx context.Context, req *dto.GetMyProposalsReq) (*dto.GetMyProposalsResp, error) {
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询提案列表
	proposals, total, err := s.ProposalRepo.FindManyByUserID(ctx, req.PageParam, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindManyByUserID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
	}

	// 转换为VO
	vos, err := s.ProposalAssembler.ToProposalVOArray(ctx, proposals, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVOArray] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposals"), errorx.KV("dst", "proposal vos"))
	}

	return &dto.GetMyProposalsResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil

}

// ApproveProposal 审批提案
func (s *ProposalService) ApproveProposal(ctx context.Context, req *dto.ToggleProposalReq) (*dto.ToggleProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}
	// 检查用户是否为管理员
	isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [IsAdminByID] error: %v, userId: %s", err, userId)
		return nil, errorx.WrapByCode(err, errno.ErrUserNotAdmin, errorx.KV("userId", userId))
	}
	if !isAdmin {
		return nil, errorx.New(errno.ErrUserNotAdmin, errorx.KV("userId", userId))
	}
	// 验证提案ID
	if req.ProposalID == "" {
		return nil, errorx.New(errno.ErrProposalIDRequired, errorx.KV("key", consts.ReqProposalID))
	}

	// 查询提案是否存在
	proposal, err := s.ProposalRepo.FindByID(ctx, req.ProposalID)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindByID] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed, errorx.KV("proposalId", req.ProposalID))
	}
	if proposal == nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindByID] proposal not found, proposalId: %s", req.ProposalID)
		return nil, errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
	}

	// 检查当前状态，不允许重复审批
	approvedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	rejectedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected)
	if proposal.Status == approvedStatusID || proposal.Status == rejectedStatusID {
		return nil, errorx.New(errno.ErrProposalAlreadyProcessed, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
	}

	// 更新提案状态为已通过
	newStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	updated, err := s.ProposalRepo.UpdateStatusByID(ctx, req.ProposalID, newStatusID)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [UpdateStatusByID] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}
	if !updated {
		return nil, errorx.New(errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}

	// 如果提案通过，同时创建对应的课程
	approvedStatusID = mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	if newStatusID == approvedStatusID {
		// 创建课程
		course := proposal.Course
		course.ID = primitive.NewObjectID().Hex()
		course.CreatedAt = time.Now()
		course.UpdatedAt = time.Now()
		course.Deleted = false

		err = s.CourseRepo.Insert(ctx, course)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [Insert] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseCreateFailed, errorx.KV("name", course.Name))
		}
	}

	// 返回成功响应
	return &dto.ToggleProposalResp{
		Resp: dto.Success(),
	}, nil
}
