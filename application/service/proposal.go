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
	"encoding/json"
	"fmt"
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
	"go.mongodb.org/mongo-driver/mongo"
)

var _ IProposalService = (*ProposalService)(nil)

type IProposalService interface {
	PreviewApproval(context.Context, *dto.ToggleProposalReq) (*dto.ApprovalPreviewResp, error)
	EntityHistory(context.Context, *dto.EntityHistoryReq) (*dto.EntityHistoryResp, error)
	GetTeacher(context.Context, string) (*dto.GetTeacherResp, error)
	ListPendingProposals(ctx context.Context, req *dto.PageParam) (*dto.ListProposalResp, error)
	CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error)
	ResubmitProposal(ctx context.Context, req *dto.ResubmitProposalReq) (*dto.ResubmitProposalResp, error)
	ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error)
	SuggestProposals(ctx context.Context, req *dto.SuggestProposalReq) (*dto.ListProposalResp, error)
	GetProposal(ctx context.Context, req *dto.GetProposalReq) (*dto.GetProposalResp, error)
	DeleteProposal(ctx context.Context, req *dto.DeleteProposalReq) (*dto.DeleteProposalResp, error)
	UpdateProposal(ctx context.Context, req *dto.UpdateProposalReq) (*dto.UpdateProposalResp, error)
	GetProposalFieldSuggestions(ctx context.Context, req *dto.GetProposalFieldSuggestionsReq) (*dto.GetProposalFieldSuggestionsResp, error)
	ApproveProposal(ctx context.Context, req *dto.ToggleProposalReq) (*dto.ToggleProposalResp, error)
	RevokeProposal(ctx context.Context, req *dto.RevokeProposalReq) (*dto.RevokeProposalResp, error)
	RejectProposal(ctx context.Context, req *dto.RejectProposalReq) (*dto.RejectProposalResp, error)
}

type ProposalService struct {
	CourseRepo        *repo.CourseRepo
	CommentRepo       *repo.CommentRepo
	CommentCache      *cache.CommentCache
	CourseAssembler   *assembler.CourseAssembler
	ProposalRepo      *repo.ProposalRepo
	ProposalAssembler *assembler.ProposalAssembler
	LikeRepo          *repo.LikeRepo
	LikeCache         *cache.LikeCache
	UserRepo          *repo.UserRepo
	TeacherRepo       *repo.TeacherRepo
	MappingRepo       *repo.MappingRepo
	ChangeLogRepo     *repo.ChangeLogRepo
	ChangeLogService  IChangeLogService
}

var ProposalServiceSet = wire.NewSet(
	wire.Struct(new(ProposalService), "*"),
	wire.Bind(new(IProposalService), new(*ProposalService)),
)

// getDailyProposalLimit 根据用户贡献值计算每日提案发布上限
func getDailyProposalLimit(contribution int64) int64 {
	switch {
	case contribution >= consts.ContributionThresholdHigh:
		return consts.ProposalDailyQuotaHigh
	case contribution >= consts.ContributionThresholdMedium:
		return consts.ProposalDailyQuotaMedium
	default:
		return consts.ProposalDailyQuotaLow
	}
}

func validateProposalInput(title string, course *dto.ProposalCourseVO) error {
	if strings.TrimSpace(title) == "" {
		return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "title"))
	}
	if course == nil {
		return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "course"))
	}
	if strings.TrimSpace(course.Code) == "" {
		return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "course.code"))
	}
	if strings.TrimSpace(course.Name) == "" || strings.TrimSpace(course.Department) == "" ||
		strings.TrimSpace(course.Category) == "" || len(course.Campuses) == 0 {
		return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "course required fields"))
	}
	for _, campusName := range course.Campuses {
		campusName = strings.TrimSpace(campusName)
		if campusName == "" || mapping.Data.GetCampusIDByName(campusName) == 0 {
			return errorx.New(errno.ErrProposalInvalidCampus,
				errorx.KV("key", consts.Campuses), errorx.KV("value", campusName))
		}
	}
	for _, teacher := range course.Teachers {
		if teacher == nil || strings.TrimSpace(teacher.Name) == "" {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "course.teachers"))
		}
	}
	return nil
}

func resolveApprovalTitle(currentTitle, requestedTitle string) (string, bool) {
	title := strings.TrimSpace(requestedTitle)
	if title == "" || title == currentTitle {
		return currentTitle, false
	}
	return title, true
}

// ListProposals 分页查询不同状态的提案，用于投票列表或管理端审核
func (s *ProposalService) ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	isAdmin, adminErr := s.UserRepo.IsAdminByID(ctx, userId)
	if adminErr != nil {
		return nil, errorx.WrapByCode(adminErr, errno.ErrUserFindFailed)
	}
	statusName := strings.TrimSpace(req.Status)
	if !isAdmin {
		statusName = consts.ProposalStatusApproved
	}
	status := mapping.Data.GetProposalStatusIDByName(statusName)
	if statusName != "" && status == 0 {
		return nil, errorx.New(errno.ErrProposalGetStatusFailed,
			errorx.KV("key", consts.Status), errorx.KV("value", statusName))
	}

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

	// 贡献值仅创建者可见
	filterContributionVisibility(vos, userId)

	// 为已通过提案附加关联的正式课程信息
	s.attachFinalCourses(ctx, vos)

	return &dto.ListProposalResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil
}

// ListPendingProposals exposes only active pending proposals to authenticated users.
func (s *ProposalService) ListPendingProposals(ctx context.Context, req *dto.PageParam) (*dto.ListProposalResp, error) {
	userID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}
	status := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
	proposals, total, err := s.ProposalRepo.FindManyByStatus(ctx, req, status)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
	}
	vos, err := s.ProposalAssembler.ToProposalVOArray(ctx, proposals, userID)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed, errorx.KV("src", "database proposals"), errorx.KV("dst", "proposal vos"))
	}
	filterContributionVisibility(vos, userID)
	return &dto.ListProposalResp{Resp: dto.Success(), Total: total, Proposals: vos}, nil
}

// normalizeJSONArrayParam 兼容前端将数组序列化为 JSON 字符串的 query 传参方式
// 例如 ?status=["pending","approved"] 会被展开为 ["pending", "approved"]
func normalizeJSONArrayParam(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(trimmed, "[") {
			var arr []string
			if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
				result = append(result, arr...)
				continue
			}
		}
		result = append(result, value)
	}
	return result
}

// SuggestProposals 分页搜索并筛选提案列表
func (s *ProposalService) SuggestProposals(ctx context.Context, req *dto.SuggestProposalReq) (*dto.ListProposalResp, error) {
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 兼容 JSON 字符串风格的数组传参
	req.Statuses = normalizeJSONArrayParam(req.Statuses)
	req.Campuses = normalizeJSONArrayParam(req.Campuses)

	// 角色判断：查询失败则按普通用户处理
	isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
	if err != nil {
		isAdmin = false
	}

	// 角色数据范围控制
	if !isAdmin {
		// 普通用户：强制状态为已通过，忽略前端传入
		req.Statuses = []string{consts.ProposalStatusApproved}
	}

	statuses := make([]int32, 0, len(req.Statuses))
	for _, statusName := range req.Statuses {
		statusName = strings.TrimSpace(statusName)
		if statusName == "" {
			continue
		}

		statusID := mapping.Data.GetProposalStatusIDByName(statusName)
		if statusID == 0 {
			return nil, errorx.New(errno.ErrProposalGetStatusFailed,
				errorx.KV("key", consts.Status),
				errorx.KV("value", statusName),
			)
		}
		statuses = append(statuses, statusID)
	}

	campuses := make([]string, 0, len(req.Campuses))
	for _, campusName := range req.Campuses {
		campusName = strings.TrimSpace(campusName)
		if campusName == "" {
			continue
		}

		if mapping.Data.GetCampusIDByName(campusName) == 0 {
			return nil, errorx.New(errno.ErrProposalInvalidCampus,
				errorx.KV("key", consts.Campuses),
				errorx.KV("value", campusName),
			)
		}
		campuses = append(campuses, campusName)
	}

	req.Campuses = campuses

	proposals, total, err := s.ProposalRepo.FindManyByFilter(ctx, req, statuses)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindManyByFilter] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
	}

	vos, err := s.ProposalAssembler.ToProposalVOArray(ctx, proposals, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVOArray] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposals"), errorx.KV("dst", "proposal vos"))
	}

	//贡献值权限过滤：仅创建者可见自己的 Contribution
	for _, vo := range vos {
		if vo.UserID != userId {
			// 若非创建者则将contribution置-1表示不显示
			vo.Contribution = -1
		}
		// 若创建者是自己，保留原值（由 Assembler 填充）
	}

	// 为已通过提案附加关联的正式课程信息
	s.attachFinalCourses(ctx, vos)

	return &dto.ListProposalResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil
}

// attachFinalCourses 为列表中状态为已通过的提案附加关联的正式课程信息（通过课程的来源提案ID关联），
// 查询失败或课程已删除时 FinalCourse 保持为空，不影响主流程
func (s *ProposalService) attachFinalCourses(ctx context.Context, vos []*dto.ProposalVO) {
	for _, vo := range vos {
		if vo.Status != consts.ProposalStatusApproved || vo.FinalCourse != nil || vo.Type == model.ProposalUpdateTeacher {
			continue
		}

		// 根据提案 ID 查询关联的正式课程（仅返回未删除的课程）
		course, err := s.CourseRepo.FindByProposalID(ctx, vo.ID)
		if err != nil {
			// 查询失败不影响主流程，FinalCourse 保持为空
			logs.CtxWarnf(ctx, "[CourseRepo] [FindByProposalID] error: %v, proposalId: %s", err, vo.ID)
			continue
		}
		if course == nil {
			// 课程不存在或已被删除，FinalCourse 保持为空
			continue
		}

		finalCourse, err := s.CourseAssembler.ToProposalCourseVOFromCourse(ctx, course)
		if err != nil {
			logs.CtxWarnf(ctx, "[CourseAssembler] [ToProposalCourseVOFromCourse] error: %v, proposalId: %s", err, vo.ID)
			continue
		}
		vo.FinalCourse = finalCourse
	}
}

// GetProposal 获取提案详情
func (s *ProposalService) GetProposal(ctx context.Context, req *dto.GetProposalReq) (*dto.GetProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 1. 查询提案详情（默认查询未删除的提案）
	proposalId := req.ProposalID
	proposal, err := s.ProposalRepo.FindByID(ctx, proposalId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindByID] error: %v, proposalId: %s", err, proposalId)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed, errorx.KV("proposalId", proposalId))
	}
	if proposal == nil {
		// 未删除的提案不存在，尝试查询已删除的提案（仅提案创建者本人可见，避免泄露他人已删除提案的存在性）
		proposal, err = s.ProposalRepo.FindByIDIncludeDeleted(ctx, proposalId)
		if err != nil {
			logs.CtxErrorf(ctx, "[ProposalRepo] [FindByIDIncludeDeleted] error: %v, proposalId: %s", err, proposalId)
			return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed, errorx.KV("proposalId", proposalId))
		}
		if proposal == nil || proposal.UserID != userId {
			logs.CtxWarnf(ctx, "[ProposalRepo] [FindByID] proposal not found, proposalId: %s", proposalId)
			return nil, errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", proposalId))
		}
	}

	isCreator := proposal.UserID == userId
	isAdmin := false
	adminChecked := false
	approvedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	if !isCreator && proposal.Status != approvedStatusID {
		isAdmin, err = s.UserRepo.IsAdminByID(ctx, userId)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
				errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
		}
		adminChecked = true
		if !proposalDetailsVisible(proposal.Status, approvedStatusID, isCreator, isAdmin) {
			return nil, errorx.New(errno.ErrProposalNotFound,
				errorx.KV("key", consts.ReqProposalID), errorx.KV("value", proposalId))
		}
	}

	// 2. 转换为VO（附带当前用户的点赞状态）
	vo, err := s.ProposalAssembler.ToProposalVO(ctx, proposal, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVO] error: %v, proposalId: %s", err, proposalId)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposal"), errorx.KV("dst", "proposal vo"))
	}

	// 贡献值仅创建者可见（统一过滤）
	filterContributionVisibility([]*dto.ProposalVO{vo}, userId)

	// 填充最终课程信息：仅提案状态为已通过，且当前用户为提案创建者或管理员时可见
	if vo.Status == consts.ProposalStatusApproved {
		if !isCreator && !adminChecked {
			isAdmin, err = s.UserRepo.IsAdminByID(ctx, userId)
			if err != nil {
				// 管理员查询失败不影响主流程，按非管理员处理
				logs.CtxWarnf(ctx, "[UserRepo] [IsAdminByID] error: %v, userId: %s", err, userId)
				isAdmin = false
			}
		}
		if (isCreator || isAdmin) && vo.FinalCourse == nil && proposal.EffectiveType() != model.ProposalUpdateTeacher {
			course, err := s.CourseRepo.FindByProposalID(ctx, proposal.ID)
			if err != nil {
				// 查询失败不影响主流程，FinalCourse 保持为空
				logs.CtxWarnf(ctx, "[CourseRepo] [FindByProposalID] error: %v, proposalId: %s", err, proposal.ID)
			} else if course != nil {
				finalCourse, err := s.CourseAssembler.ToProposalCourseVOFromCourse(ctx, course)
				if err != nil {
					logs.CtxWarnf(ctx, "[CourseAssembler] [ToProposalCourseVOFromCourse] error: %v, proposalId: %s", err, proposal.ID)
				} else {
					vo.FinalCourse = finalCourse
				}
			}
		}
	}

	return &dto.GetProposalResp{
		Resp:     dto.Success(),
		Proposal: vo,
	}, nil
}

func proposalDetailsVisible(status, approvedStatus int32, isCreator, isAdmin bool) bool {
	return status == approvedStatus || isCreator || isAdmin
}

// DeleteProposal 删除提案
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
		for id, name := range mapping.Data.AllCampuses() {
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
		// 按姓名、职称或无分隔的姓名+职称查询，并按搜索值合并重复教师记录。
		teachers, err := s.TeacherRepo.FindSuggestionCandidates(ctx, req.Keyword, nil)
		if err != nil {
			logs.CtxErrorf(ctx, "[TeacherRepo] [FindSuggestionCandidates] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrTeacherGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}
		allGroups := groupTeacherSuggestionCandidates(teachers, req.Keyword)
		total = int64(len(allGroups))
		groups := pageTeacherSuggestionGroups(allGroups, req.PageParam)
		teacherIDs := make([]string, 0, len(teachers))
		for _, group := range groups {
			teacherIDs = append(teacherIDs, group.TeacherIDs...)
		}
		coursesByTeacher, err := s.CourseRepo.FindRecentByTeacherIDs(ctx, teacherIDs, 2)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [FindRecentByTeacherIDs] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}

		for _, group := range groups {
			teacher := group.Teacher
			title := teacher.Title
			briefs := recentCourseBriefsForTeacherIDs(group.TeacherIDs, coursesByTeacher, 2)
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:      teacher.ID,
				Value:   teacher.Name,
				Label:   teacherSuggestionLabel(teacher.Name, teacher.Title),
				Title:   &title,
				Courses: &briefs,
			})
		}

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

func teacherSuggestionLabel(name, title string) string {
	if title == "" {
		return name
	}
	return name + " - " + title
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

	// 贡献值仅创建者可见（本接口返回均为自己的提案，过滤为恒等操作，保持一致）
	filterContributionVisibility(vos, userId)

	// 为已通过提案附加关联的正式课程信息
	s.attachFinalCourses(ctx, vos)

	return &dto.GetMyProposalsResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil

}

// rollbackContributionStrict 在撤回事务内扣回贡献值并清空提案记录；任一步失败均回滚。
func (s *ProposalService) rollbackContributionStrict(ctx context.Context, proposal *model.Proposal) error {
	amount := proposal.Contribution
	if amount <= 0 {
		// 兜底：贡献值字段为空时重新计算一次得分
		amount = s.recalcContribution(ctx, proposal)
	}
	if amount <= 0 {
		logs.CtxInfof(ctx, "[ProposalService] [RevokeProposal] no contribution to rollback, proposalId: %s", proposal.ID)
		return nil
	}

	// 扣减用户贡献值
	if err := s.UserRepo.IncrementContribution(ctx, proposal.UserID, -amount); err != nil {
		return fmt.Errorf("decrement user contribution: %w", err)
	}

	// 提案贡献值置空，表示已撤回
	if err := s.ProposalRepo.UpdateContributionByID(ctx, proposal.ID, 0); err != nil {
		return fmt.Errorf("clear proposal contribution: %w", err)
	}
	return nil
}

// recalcContribution 重新计算提案的贡献值得分（兜底，对比提案原始课程与关联的最终课程）
func (s *ProposalService) recalcContribution(ctx context.Context, proposal *model.Proposal) int64 {
	originalVO, err := s.CourseAssembler.ToProposalCourseVO(ctx, proposal.Course)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToProposalCourseVO] error: %v, proposalId: %s", err, proposal.ID)
		return 0
	}

	course, err := s.CourseRepo.FindByProposalIDIncludeDeleted(ctx, proposal.ID)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [FindByProposalID] error: %v, proposalId: %s", err, proposal.ID)
		return 0
	}
	if course == nil {
		return 0
	}

	finalVO := s.courseToProposalCourseVO(ctx, course)
	if finalVO == nil {
		return 0
	}
	score := calcContributionScore(originalVO, finalVO)
	if strings.TrimSpace(originalVO.Code) == strings.TrimSpace(finalVO.Code) {
		score--
	}
	return score
}

// courseToProposalCourseVO 将正式课程模型转换为提案课程VO（用于贡献值重新计算）
func (s *ProposalService) courseToProposalCourseVO(ctx context.Context, course *model.Course) *dto.ProposalCourseVO {
	if course == nil {
		return nil
	}

	campuses := make([]string, 0, len(course.Campuses))
	for _, id := range course.Campuses {
		name := mapping.Data.GetCampusNameByID(id)
		if name != "" {
			campuses = append(campuses, name)
		}
	}

	teachers := make([]*dto.TeacherVO, 0, len(course.TeacherIDs))
	for _, teacherID := range course.TeacherIDs {
		teacher, err := s.TeacherRepo.FindByID(ctx, teacherID)
		if err != nil {
			logs.CtxErrorf(ctx, "[TeacherRepo] [FindByID] error: %v, teacherId: %s", err, teacherID)
			continue
		}
		if teacher == nil {
			continue
		}
		teachers = append(teachers, &dto.TeacherVO{
			ID:         teacher.ID,
			Name:       teacher.Name,
			Title:      teacher.Title,
			Department: mapping.Data.GetDepartmentNameByID(teacher.Department),
		})
	}

	return &dto.ProposalCourseVO{
		Name:       course.Name,
		Code:       course.Code,
		Department: mapping.Data.GetDepartmentNameByID(course.Department),
		Category:   mapping.Data.GetCategoryNameByID(course.Category),
		Campuses:   campuses,
		Teachers:   teachers,
	}
}

// calcContributionScore 对比提案原始课程与管理员最终确认课程，逐字段一致加1分，满分5分
func calcContributionScore(original, final *dto.ProposalCourseVO) int64 {
	if original == nil || final == nil {
		return 0
	}

	var score int64
	if strings.TrimSpace(original.Code) == strings.TrimSpace(final.Code) {
		score++
	}
	if strings.TrimSpace(original.Name) == strings.TrimSpace(final.Name) {
		score++
	}
	if strings.TrimSpace(original.Department) == strings.TrimSpace(final.Department) {
		score++
	}
	if strings.TrimSpace(original.Category) == strings.TrimSpace(final.Category) {
		score++
	}
	if stringSetEqual(original.Campuses, final.Campuses) {
		score++
	}
	if teacherNameSetEqual(original.Teachers, final.Teachers) {
		score++
	}
	return score
}

// stringSetEqual 比较两个字符串集合是否相等（忽略顺序）
func stringSetEqual(a, b []string) bool {
	setA := make(map[string]struct{}, len(a))
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		setA[s] = struct{}{}
	}
	setB := make(map[string]struct{}, len(b))
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		setB[s] = struct{}{}
	}
	if len(setA) != len(setB) {
		return false
	}
	for s := range setA {
		if _, ok := setB[s]; !ok {
			return false
		}
	}
	return true
}

// teacherNameSetEqual 比较两个教师列表的姓名集合是否相等（忽略顺序）
func teacherNameSetEqual(a, b []*dto.TeacherVO) bool {
	namesA := make([]string, 0, len(a))
	for _, t := range a {
		if t != nil {
			namesA = append(namesA, t.Name)
		}
	}
	namesB := make([]string, 0, len(b))
	for _, t := range b {
		if t != nil {
			namesB = append(namesB, t.Name)
		}
	}
	return stringSetEqual(namesA, namesB)
}

// filterContributionVisibility 贡献值仅创建者可见，非创建者置-1（前端识别-1隐藏展示）
func filterContributionVisibility(vos []*dto.ProposalVO, userId string) {
	for _, vo := range vos {
		if vo == nil {
			continue
		}
		if vo.UserID != userId {
			vo.Contribution = -1
		}
	}
}

const approvalRevokeWindow = 24 * time.Hour

func validateApprovalRevocation(proposal *model.Proposal, approvedStatusID int32, loggedApprovalAt, now time.Time) error {
	if proposal.Status != approvedStatusID {
		return errorx.New(errno.ErrProposalStatusNotApproved, errorx.KV("proposalId", proposal.ID))
	}
	// Legacy likes could overwrite UpdatedAt. The latest approval log caps the
	// window at the approval time; new approvals write UpdatedAt just before it.
	approvedAt := proposal.UpdatedAt
	if !loggedApprovalAt.IsZero() && (approvedAt.IsZero() || loggedApprovalAt.Before(approvedAt)) {
		approvedAt = loggedApprovalAt
	}
	if approvedAt.IsZero() || !now.Before(approvedAt.Add(approvalRevokeWindow)) {
		return errorx.New(errno.ErrProposalRevokeTimeLimitExceeded, errorx.KV("proposalId", proposal.ID))
	}
	return nil
}

// RevokeProposal 撤回提案操作（通过/拒绝）
func (s *ProposalService) legacyRevokeProposal(ctx context.Context, req *dto.RevokeProposalReq) (*dto.RevokeProposalResp, error) {
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

	approvedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	rejectedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected)
	pendingStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
	if req.ActionType != consts.RevokeActionApprove && req.ActionType != consts.RevokeActionReject {
		return nil, errorx.New(errno.ErrRevokeActionTypeInvalid, errorx.KV("actionType", req.ActionType))
	}

	var proposalUserID string
	var deletedTeachers []*model.Teacher
	var deletedMappings []*model.Mapping
	err = s.workflowTransaction(ctx, func(txCtx mongo.SessionContext) error {
		deletedTeachers = nil
		deletedMappings = nil
		proposal, findErr := s.ProposalRepo.FindByID(txCtx, req.ProposalID)
		if findErr != nil {
			return errorx.WrapByCode(findErr, errno.ErrProposalFindFailed, errorx.KV("proposalId", req.ProposalID))
		}
		if proposal == nil {
			return errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
		}
		proposalUserID = proposal.UserID

		expectedStatusID := rejectedStatusID
		action := consts.ActionTypeRevokeRejectProposal
		content := "撤回提案审批：拒绝→待审核"
		if req.ActionType == consts.RevokeActionApprove {
			expectedStatusID = approvedStatusID
			action = consts.ActionTypeRevokeApproveProposal
			content = "撤回提案审批：通过→待审核"
			var loggedApprovalAt time.Time
			if proposal.Status == approvedStatusID {
				var logErr error
				loggedApprovalAt, logErr = s.ChangeLogRepo.FindLatestProposalApprovalTime(txCtx, req.ProposalID)
				if logErr != nil {
					return errorx.WrapByCode(logErr, errno.ErrChangeLogFindFailed)
				}
			}
			if validationErr := validateApprovalRevocation(proposal, approvedStatusID, loggedApprovalAt, time.Now()); validationErr != nil {
				return validationErr
			}
			associatedCourse, courseErr := s.CourseRepo.FindByProposalIDIncludeDeleted(txCtx, req.ProposalID)
			if courseErr != nil {
				return errorx.WrapByCode(courseErr, errno.ErrCourseNotFoundCannotRevoke)
			}
			if associatedCourse != nil {
				if associatedCourse.DecisionBatchID != "" {
					return errorx.New(errno.ErrCourseModifiedCannotRevoke)
				}
				if !associatedCourse.Deleted {
					if courseErr = s.CourseRepo.SoftDeleteByID(txCtx, associatedCourse.ID); courseErr != nil {
						return errorx.WrapByCode(courseErr, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
					}
				}
				commentIDs, commentErr := s.CommentRepo.SoftDeleteByCourseID(txCtx, associatedCourse.ID)
				if commentErr != nil {
					return fmt.Errorf("soft delete course comments: %w", commentErr)
				}
				commentTargetType := mapping.Data.GetLikeTargetTypeIDByName(consts.LikeTargetTypeComment)
				if likeErr := s.LikeRepo.DeleteByTargets(txCtx, commentIDs, commentTargetType); likeErr != nil {
					return fmt.Errorf("delete course comment likes: %w", likeErr)
				}
				seenTeacherIDs := make(map[string]struct{}, len(associatedCourse.TeacherIDs))
				for _, teacherID := range associatedCourse.TeacherIDs {
					if teacherID == "" {
						continue
					}
					if _, seen := seenTeacherIDs[teacherID]; seen {
						continue
					}
					seenTeacherIDs[teacherID] = struct{}{}
					courseReferenced, referenceErr := s.CourseRepo.IsTeacherReferenced(txCtx, teacherID)
					if referenceErr != nil {
						return fmt.Errorf("check course teacher reference %s: %w", teacherID, referenceErr)
					}
					proposalReferenced, referenceErr := s.ProposalRepo.IsTeacherReferenced(txCtx, teacherID)
					if referenceErr != nil {
						return fmt.Errorf("check proposal teacher reference %s: %w", teacherID, referenceErr)
					}
					if proposal.DecisionBatchID == "" || !shouldDeleteTeacher(courseReferenced, proposalReferenced) {
						continue
					}
					deletedTeacher, deleteErr := s.TeacherRepo.DeleteByID(txCtx, teacherID)
					if deleteErr != nil {
						return fmt.Errorf("delete unreferenced teacher %s: %w", teacherID, deleteErr)
					}
					if deletedTeacher != nil {
						deletedTeachers = append(deletedTeachers, deletedTeacher)
					}
				}
				for _, mappingRef := range revokedCourseMappingReferences(associatedCourse, deletedTeachers) {
					deletedMapping, cleanupErr := s.deleteMappingIfUnreferenced(txCtx, mappingRef.mappingType, mappingRef.code)
					if cleanupErr != nil {
						return cleanupErr
					}
					if deletedMapping != nil {
						deletedMappings = append(deletedMappings, deletedMapping)
					}
				}
			}
			if contributionErr := s.rollbackContributionStrict(txCtx, proposal); contributionErr != nil {
				return contributionErr
			}
		} else if proposal.Status != rejectedStatusID {
			return errorx.New(errno.ErrProposalStatusNotRejected, errorx.KV("proposalId", req.ProposalID))
		}

		var updated bool
		var updateErr error
		if req.ActionType == consts.RevokeActionReject {
			updated, updateErr = s.ProposalRepo.UpdateStatusAndReasonByID(
				txCtx, req.ProposalID, expectedStatusID, pendingStatusID, "",
			)
		} else {
			updated, updateErr = s.ProposalRepo.UpdateStatusByID(txCtx, req.ProposalID, expectedStatusID, pendingStatusID)
		}
		if updateErr != nil {
			return errorx.WrapByCode(updateErr, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
		}
		if !updated {
			return errorx.New(errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
		}
		_, logErr := s.ChangeLogService.CreateChangeLog(txCtx, &dto.CreateChangeLogReq{
			TargetID: req.ProposalID, TargetType: consts.TargetTypeProposal, Action: action,
			Content: content, UpdateSource: consts.UpdateSourceAdmin, ProposalID: req.ProposalID,
		})
		return logErr
	})
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalService] transactional revoke failed: %v, proposalId: %s", err, req.ProposalID)
		return nil, err
	}
	if req.ActionType == consts.RevokeActionApprove {
		if s.CommentCache != nil {
			if invalidateErr := s.CommentCache.DeleteCount(ctx); invalidateErr != nil {
				logs.CtxWarnf(ctx, "[CommentCache] post-revoke count invalidation failed: %v", invalidateErr)
			}
		}
		for _, teacher := range deletedTeachers {
			if invalidateErr := s.TeacherRepo.InvalidateDeleted(ctx, teacher); invalidateErr != nil {
				logs.CtxWarnf(ctx, "[TeacherRepo] post-revoke cache invalidation failed: %v", invalidateErr)
			}
		}
		if invalidateErr := s.UserRepo.InvalidateByID(ctx, proposalUserID); invalidateErr != nil {
			logs.CtxWarnf(ctx, "[UserRepo] post-revoke cache invalidation failed: %v", invalidateErr)
		}
		if refreshErr := mapping.Data.Refresh(ctx); refreshErr != nil {
			logs.CtxWarnf(ctx, "[Mapping] post-revoke refresh failed: %v", refreshErr)
		}
		if len(deletedMappings) > 0 {
			logs.CtxInfof(ctx, "[Mapping] removed %d unreferenced mappings after revoke", len(deletedMappings))
		}
	}

	return &dto.RevokeProposalResp{
		Resp:       dto.Success(),
		ProposalID: req.ProposalID,
	}, nil
}

func shouldDeleteTeacher(courseReferenced, proposalReferenced bool) bool {
	return !courseReferenced && !proposalReferenced
}

func shouldDeleteMapping(courseReferenced, teacherReferenced bool) bool {
	return !courseReferenced && !teacherReferenced
}

type mappingReference struct {
	mappingType model.MappingType
	code        int32
}

func revokedCourseMappingReferences(course *model.Course, deletedTeachers []*model.Teacher) []mappingReference {
	if course == nil {
		return nil
	}
	candidates := []mappingReference{
		{mappingType: model.MappingTypeDepartment, code: course.Department},
		{mappingType: model.MappingTypeCategory, code: course.Category},
	}
	for _, teacher := range deletedTeachers {
		if teacher != nil {
			candidates = append(candidates, mappingReference{mappingType: model.MappingTypeDepartment, code: teacher.Department})
		}
	}

	seen := make(map[mappingReference]struct{}, len(candidates))
	result := make([]mappingReference, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.code <= 0 {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func (s *ProposalService) deleteMappingIfUnreferenced(ctx context.Context, mappingType model.MappingType, code int32) (*model.Mapping, error) {
	if code <= 0 {
		return nil, nil
	}
	courseReferenced, err := s.CourseRepo.IsMappingReferenced(ctx, mappingType, code)
	if err != nil {
		return nil, fmt.Errorf("check course mapping reference type=%d code=%d: %w", mappingType, code, err)
	}
	teacherReferenced := false
	if mappingType == model.MappingTypeDepartment {
		teacherReferenced, err = s.TeacherRepo.IsDepartmentReferenced(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("check teacher department reference code=%d: %w", code, err)
		}
	}
	if !shouldDeleteMapping(courseReferenced, teacherReferenced) {
		return nil, nil
	}
	deleted, err := s.MappingRepo.DeleteByCodeAndType(ctx, code, mappingType)
	if err != nil {
		return nil, fmt.Errorf("delete unreferenced mapping type=%d code=%d: %w", mappingType, code, err)
	}
	return deleted, nil
}

// RejectProposal 拒绝提案，将状态从 pending 改为 rejected
