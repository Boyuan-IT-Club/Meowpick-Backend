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
	"crypto/sha256"
	"encoding/hex"
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
	typesMapping "github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"

	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ IProposalService = (*ProposalService)(nil)

type IProposalService interface {
	CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error)
	ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error)
	FilterProposals(ctx context.Context, req *dto.FilterProposalReq) (*dto.ListProposalResp, error)
	GetProposal(ctx context.Context, req *dto.GetProposalReq) (*dto.GetProposalResp, error)
	DeleteProposal(ctx context.Context, req *dto.DeleteProposalReq) (*dto.DeleteProposalResp, error)
	UpdateProposal(ctx context.Context, req *dto.UpdateProposalReq) (*dto.UpdateProposalResp, error)
	GetProposalSuggestions(ctx context.Context, req *dto.GetProposalSuggestionsReq) (*dto.GetProposalSuggestionsResp, error)
	GetProposalFieldSuggestions(ctx context.Context, req *dto.GetProposalFieldSuggestionsReq) (*dto.GetProposalFieldSuggestionsResp, error)
	ApproveProposal(ctx context.Context, req *dto.ToggleProposalReq) (*dto.ToggleProposalResp, error)
	RevokeProposal(ctx context.Context, req *dto.RevokeProposalReq) (*dto.RevokeProposalResp, error)
	RejectProposal(ctx context.Context, req *dto.RejectProposalReq) (*dto.RejectProposalResp, error)
}

type ProposalService struct {
	CourseRepo        *repo.CourseRepo
	CommentRepo       *repo.CommentRepo
	CourseAssembler   *assembler.CourseAssembler
	ProposalRepo      *repo.ProposalRepo
	ProposalAssembler *assembler.ProposalAssembler
	LikeRepo          *repo.LikeRepo
	LikeCache         *cache.LikeCache
	UserRepo          *repo.UserRepo
	TeacherRepo       *repo.TeacherRepo
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
		if teacher == nil || strings.TrimSpace(teacher.Name) == "" ||
			(teacher.ID == "" && strings.TrimSpace(teacher.Department) == "") {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "course.teachers"))
		}
	}
	return nil
}

// CreateProposal 添加一个新的课程提案
func (s *ProposalService) CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	if validationErr := validateProposalInput(req.Title, req.Course); validationErr != nil {
		return nil, validationErr
	}

	// 转换为 proposalCourseModel，不执行自动注册
	req.Course.ID = primitive.NewObjectID().Hex()
	course, err := s.CourseAssembler.ToProposalCourseDB(ctx, req.Course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "database proposal course"), errorx.KV("dst", "course vo"),
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

	// 检查是否已经存在相同的课程 (DryRun转换，不执行自动注册)
	courseDBDryRun, err := s.CourseAssembler.ToCourseDBDryRunFromProposalCourse(ctx, req.Course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "proposal course vo"), errorx.KV("dst", "course model dryrun"),
		)
	}

	existingCourse, err := s.CourseRepo.IsCourseInExistingCourses(ctx, courseDBDryRun)
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

	// 每日上限检查：根据用户贡献值计算每日发布上限，并统计今日已提交数量
	user, err := s.UserRepo.FindByID(ctx, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [FindByID] error: %v, userId: %s", err, userId)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
	}
	if user == nil {
		return nil, errorx.New(errno.ErrUserNotFound,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
	}
	dailyLimit := getDailyProposalLimit(user.Contribution)
	todayCount, err := s.ProposalRepo.CountByUserToday(ctx, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [CountByUserToday] error: %v, userId: %s", err, userId)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCountFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
	}
	if todayCount >= dailyLimit {
		return nil, errorx.New(errno.ErrDailyProposalLimitReached, errorx.KV("limit", strconv.FormatInt(dailyLimit, 10)))
	}

	// 1. 构建数据库模型
	now := time.Now()
	proposalVO := &dto.ProposalVO{
		ID:           primitive.NewObjectID().Hex(),
		UserID:       userId,
		Title:        req.Title,
		Content:      req.Content,
		Status:       consts.ProposalStatusPending,
		Deleted:      false,
		Course:       req.Course,
		ShowUsername: req.ShowUsername,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	proposal, err := s.ProposalAssembler.ToProposalDB(ctx, proposalVO)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalDB] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "proposal vo"), errorx.KV("dst", "database proposal"),
		)
	}

	// 2. 在事务内串行化同用户当天额度检查和同课程防重检查。
	chinaLocation := time.FixedZone("Asia/Shanghai", 8*60*60)
	userDayKey := userId + ":" + now.In(chinaLocation).Format("2006-01-02")
	fingerprintBytes, marshalErr := json.Marshal(course)
	if marshalErr != nil {
		return nil, errorx.WrapByCode(marshalErr, errno.ErrProposalCvtFailed)
	}
	fingerprintSum := sha256.Sum256(fingerprintBytes)
	courseFingerprint := hex.EncodeToString(fingerprintSum[:])
	if guardErr := s.ProposalRepo.AcquireCreateGuards(ctx, userDayKey, courseFingerprint); guardErr != nil {
		return nil, errorx.WrapByCode(guardErr, errno.ErrProposalCreateFailed, errorx.KV("name", req.Course.Name))
	}
	err = s.ProposalRepo.WithTransaction(ctx, func(txCtx mongo.SessionContext) error {
		if guardErr := s.ProposalRepo.AcquireCreateGuards(txCtx, userDayKey, courseFingerprint); guardErr != nil {
			return guardErr
		}
		duplicateProposal, checkErr := s.ProposalRepo.IsCourseInExistingProposals(txCtx, course)
		if checkErr != nil {
			return errorx.WrapByCode(checkErr, errno.ErrProposalCourseFindInProposalsFailed)
		}
		if duplicateProposal {
			return errorx.New(errno.ErrProposalCourseFoundInProposals,
				errorx.KV("key", consts.ReqCourse), errorx.KV("value", req.Course.Name))
		}
		duplicateCourse, checkErr := s.CourseRepo.IsCourseInExistingCourses(txCtx, courseDBDryRun)
		if checkErr != nil {
			return errorx.WrapByCode(checkErr, errno.ErrProposalCourseFindInCoursesFailed)
		}
		if duplicateCourse {
			return errorx.New(errno.ErrProposalCourseFoundInCourses,
				errorx.KV("key", consts.ReqCourse), errorx.KV("value", req.Course.Name))
		}
		txUser, findErr := s.UserRepo.FindByID(txCtx, userId)
		if findErr != nil {
			return errorx.WrapByCode(findErr, errno.ErrUserFindFailed)
		}
		if txUser == nil {
			return errorx.New(errno.ErrUserNotFound, errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
		}
		limit := getDailyProposalLimit(txUser.Contribution)
		count, countErr := s.ProposalRepo.CountByUserToday(txCtx, userId)
		if countErr != nil {
			return errorx.WrapByCode(countErr, errno.ErrProposalCountFailed)
		}
		if count >= limit {
			return errorx.New(errno.ErrDailyProposalLimitReached, errorx.KV("limit", strconv.FormatInt(limit, 10)))
		}
		if insertErr := s.ProposalRepo.Insert(txCtx, proposal); insertErr != nil {
			return errorx.WrapByCode(insertErr, errno.ErrProposalCreateFailed, errorx.KV("name", req.Course.Name))
		}
		_, logErr := s.ChangeLogService.CreateChangeLog(txCtx, &dto.CreateChangeLogReq{
			TargetID: proposal.ID, TargetType: consts.TargetTypeProposal,
			Action: consts.ActionTypeCreateProposal, Content: "创建提案",
			UpdateSource: consts.UpdateSourceUser, ProposalID: proposal.ID,
		})
		return logErr
	})
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalService] transactional create failed: %v", err)
		return nil, err
	}

	// 3. 转换为 VO (包含点赞信息)
	vo, err := s.ProposalAssembler.ToProposalVO(ctx, proposal, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVO] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposal"), errorx.KV("dst", "proposal vo"))
	}

	return &dto.CreateProposalResp{
		Resp:       dto.Success(),
		ProposalID: proposal.ID,
		Proposal:   vo,
	}, nil
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

	return &dto.ListProposalResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil
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

// FilterProposals 分页筛选提案列表
func (s *ProposalService) FilterProposals(ctx context.Context, req *dto.FilterProposalReq) (*dto.ListProposalResp, error) {
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
	isCreator := proposal.UserID == userId
	if vo.Status == consts.ProposalStatusApproved {
		isAdmin := false
		if !isCreator {
			isAdmin, err = s.UserRepo.IsAdminByID(ctx, userId)
			if err != nil {
				// 管理员查询失败不影响主流程，按非管理员处理
				logs.CtxWarnf(ctx, "[UserRepo] [IsAdminByID] error: %v, userId: %s", err, userId)
				isAdmin = false
			}
		}
		if isCreator || isAdmin {
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

	// 权限检查：仅允许删除自己创建的提案（管理员也不例外）
	if proposal.UserID != userId {
		return nil, errorx.New(errno.ErrUserNotOwner,
			errorx.KV("id", userId))
	}

	// 状态检查：只有待审核和已拒绝状态的提案才允许删除
	approvedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	if proposal.Status == approvedStatusID {
		logs.CtxInfof(ctx, "[DeleteProposal] cannot delete approved proposal, proposalId: %s, status: %d", proposalId, proposal.Status)
		return nil, errorx.New(errno.ErrProposalCannotDeleteApproved,
			errorx.KV("status", typesMapping.ProposalStatusMap[proposal.Status]))
	}

	//执行删除提案
	pendingStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
	rejectedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected)
	deleted, deleteErr := s.ProposalRepo.DeleteProposal(ctx, proposalId, userId, []int32{pendingStatusID, rejectedStatusID})
	err = deleteErr
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [Delete] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalDeleteFailed,
			errorx.KV("proposal_id", proposalId))
	}
	if !deleted {
		return nil, errorx.New(errno.ErrProposalDeleteFailed, errorx.KV("proposal_id", proposalId))
	}

	// 记录变更日志（仅创建者可删，来源固定为用户）
	if _, err = s.ChangeLogService.CreateChangeLog(ctx, &dto.CreateChangeLogReq{
		TargetID:     proposalId,
		TargetType:   consts.TargetTypeProposal,
		Action:       consts.ActionTypeDeleteProposal,
		Content:      "删除提案",
		UpdateSource: consts.UpdateSourceUser,
		ProposalID:   proposalId,
	}); err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogService] [CreateChangeLog] error: %v, proposalId: %s", err, proposalId)
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
	isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed)
	}
	if !isAdmin {
		return nil, errorx.New(errno.ErrUserNotAdmin, errorx.KV("id", userId))
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
	pendingStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
	if proposal.Status != pendingStatusID {
		return nil, errorx.New(errno.ErrProposalAlreadyProcessed,
			errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
	}
	if validationErr := validateProposalInput(req.Title, req.Course); validationErr != nil {
		return nil, validationErr
	}

	// 更新提案字段
	proposal.Title = req.Title
	proposal.Content = req.Content
	courseModel, err := s.CourseAssembler.ToProposalCourseDB(ctx, req.Course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "course vo"), errorx.KV("dst", "proposal course model"),
		)
	}
	proposal.Course = courseModel
	proposal.UpdatedAt = time.Now()

	// 执行更新
	updated, updateErr := s.ProposalRepo.UpdateProposal(ctx, proposal, pendingStatusID)
	if updateErr != nil {
		err = updateErr
		logs.CtxErrorf(ctx, "[ProposalRepo] [UpdateProposal] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}
	if !updated {
		return nil, errorx.New(errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}

	if _, err = s.ChangeLogService.CreateChangeLog(ctx, &dto.CreateChangeLogReq{
		TargetID:     proposal.ID,
		TargetType:   consts.TargetTypeProposal,
		Action:       consts.ActionTypeUpdateProposal,
		Content:      "更新提案",
		UpdateSource: consts.UpdateSourceAdmin,
		ProposalID:   proposal.ID,
	}); err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogService] [CreateChangeLog] error: %v, proposalId: %s", err, proposal.ID)
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

	// 查询提案建议（仅搜索已通过且未删除的提案）
	approvedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	proposals, _, err := s.ProposalRepo.GetSuggestionsByTitle(ctx, req.Keyword, req.PageParam, approvedStatusID)
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
		// 从数据库查询教师姓名
		teachers, total, err := s.TeacherRepo.GetSuggestionsByName(ctx, req.Keyword, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[TeacherRepo] [GetSuggestionsByName] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrTeacherGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}
		teacherIDs := make([]string, 0, len(teachers))
		for _, teacher := range teachers {
			teacherIDs = append(teacherIDs, teacher.ID)
		}
		coursesByTeacher, err := s.CourseRepo.FindRecentByTeacherIDs(ctx, teacherIDs, 2)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [FindRecentByTeacherIDs] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
				errorx.KV("keyword", req.Keyword))
		}

		for _, teacher := range teachers {
			courses := coursesByTeacher[teacher.ID]
			briefs := make([]dto.CourseBrief, 0, len(courses))
			for _, course := range courses {
				briefs = append(briefs, dto.CourseBrief{ID: course.ID, Name: course.Name})
			}
			suggestions = append(suggestions, &dto.FieldSuggestionVO{
				ID:      teacher.ID,
				Value:   teacher.Name,
				Label:   teacherSuggestionLabel(teacher.Name, teacher.Title),
				Courses: &briefs,
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

	// 为已通过提案附加关联的正式课程信息（通过课程的来源提案ID关联）
	for _, vo := range vos {
		if vo.Status != consts.ProposalStatusApproved {
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

	approvedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
	rejectedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected)
	var approvalUserID string

	// Course, teachers, newly allocated mappings, proposal status, contribution,
	// and audit log commit or roll back as one MongoDB transaction.
	err = s.ProposalRepo.WithTransaction(ctx, func(txCtx mongo.SessionContext) error {
		proposal, findErr := s.ProposalRepo.FindByID(txCtx, req.ProposalID)
		if findErr != nil {
			return errorx.WrapByCode(findErr, errno.ErrProposalFindFailed, errorx.KV("proposalId", req.ProposalID))
		}
		if proposal == nil {
			return errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
		}
		if proposal.Status == approvedStatusID || proposal.Status == rejectedStatusID {
			return errorx.New(errno.ErrProposalAlreadyProcessed, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
		}
		approvalUserID = proposal.UserID

		courseVO, resolveErr := s.resolveFinalCourse(txCtx, req, proposal)
		if resolveErr != nil {
			return resolveErr
		}
		if courseVO == nil {
			return errorx.New(errno.ErrCourseCvtFailed, errorx.KV("proposalId", req.ProposalID))
		}
		if validationErr := validateProposalInput(proposal.Title, courseVO); validationErr != nil {
			return validationErr
		}
		if createErr := s.createOrRestoreCourse(txCtx, proposal, courseVO); createErr != nil {
			return createErr
		}

		pendingStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
		updated, updateErr := s.ProposalRepo.UpdateStatusByID(txCtx, req.ProposalID, pendingStatusID, approvedStatusID)
		if updateErr != nil {
			return errorx.WrapByCode(updateErr, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
		}
		if !updated {
			return errorx.New(errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
		}
		if contributionErr := s.settleContributionStrict(txCtx, proposal, courseVO); contributionErr != nil {
			return contributionErr
		}
		if _, logErr := s.ChangeLogService.CreateChangeLog(txCtx, &dto.CreateChangeLogReq{
			TargetID:     req.ProposalID,
			TargetType:   consts.TargetTypeProposal,
			Action:       consts.ActionTypeApproveProposal,
			Content:      "审批提案：通过",
			UpdateSource: consts.UpdateSourceAdmin,
			ProposalID:   req.ProposalID,
		}); logErr != nil {
			return logErr
		}
		return nil
	})
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalService] transactional approval failed: %v, proposalId: %s", err, req.ProposalID)
		return nil, err
	}
	if invalidateErr := s.UserRepo.InvalidateByID(ctx, approvalUserID); invalidateErr != nil {
		logs.CtxWarnf(ctx, "[UserRepo] post-approval cache invalidation failed: %v", invalidateErr)
	}

	// Redis is outside the database transaction. Rebuild it only after commit;
	// cache failure is logged because MongoDB has already committed successfully.
	if refreshErr := mapping.Data.Refresh(ctx); refreshErr != nil {
		logs.CtxErrorf(ctx, "[Mapping] post-approval refresh failed: %v", refreshErr)
	}

	// 获取剩余待处理提案数量
	pendingStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
	_, pendingCount, err := s.ProposalRepo.FindManyByStatus(ctx, &dto.PageParam{Page: 1, PageSize: 1}, pendingStatusID)
	if err != nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindManyByStatus] error: %v", err)
	}

	// 返回成功响应
	return &dto.ToggleProposalResp{
		Resp:        dto.Success(),
		Proposal:    true,
		ProposalCnt: pendingCount,
	}, nil
}

// resolveFinalCourse 确定审批使用的最终课程信息，管理员传入的 finalCourse 优先
func (s *ProposalService) resolveFinalCourse(ctx context.Context, req *dto.ToggleProposalReq, proposal *model.Proposal) (*dto.ProposalCourseVO, error) {
	if req.FinalCourse != nil {
		return req.FinalCourse, nil
	}
	if proposal.Course == nil {
		return nil, nil
	}
	courseVO, err := s.CourseAssembler.ToProposalCourseVO(ctx, proposal.Course)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToProposalCourseVO] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed)
	}
	return courseVO, nil
}

// createOrRestoreCourse 创建或恢复提案关联的正式课程（一对一原则）
func (s *ProposalService) createOrRestoreCourse(ctx context.Context, proposal *model.Proposal, courseVO *dto.ProposalCourseVO) error {
	// 1. 通过提案ID查找是否已存在关联课程（包含已软删除的旧课程）
	existingCourse, err := s.CourseRepo.FindByProposalIDIncludeDeleted(ctx, proposal.ID)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [FindByProposalID] error: %v, proposalId: %s", err, proposal.ID)
		return errorx.WrapByCode(err, errno.ErrCourseCreateFailed, errorx.KV("proposalId", proposal.ID))
	}
	if existingCourse != nil {
		if existingCourse.Deleted {
			// 已软删除的旧课程：用最终课程信息更新并恢复
			restoredCourse, cvtErr := s.CourseAssembler.ToCourseDBFromProposalCourse(ctx, courseVO)
			if cvtErr != nil {
				logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDBFromProposalCourse] error: %v", cvtErr)
				return errorx.WrapByCode(cvtErr, errno.ErrCourseCvtFailed)
			}
			if restoredCourse == nil {
				return errorx.New(errno.ErrCourseCvtFailed)
			}
			restoredCourse.ID = existingCourse.ID
			restoredCourse.ProposalID = proposal.ID
			if err := s.CourseRepo.UpdateCourse(ctx, restoredCourse); err != nil {
				logs.CtxErrorf(ctx, "[CourseRepo] [UpdateCourse] error: %v, courseId: %s", err, existingCourse.ID)
				return errorx.WrapByCode(err, errno.ErrCourseCreateFailed, errorx.KV("courseId", existingCourse.ID))
			}
			return nil
		}
		// 已存在未删除的关联课程，跳过创建
		logs.CtxInfof(ctx, "[ProposalService] [ApproveProposal] associated course already exists, skip create, proposalId: %s", proposal.ID)
		return nil
	}

	// 2. 未找到关联课程：DryRun 防重检查
	dryRunCourse, err := s.CourseAssembler.ToCourseDBDryRunFromProposalCourse(ctx, courseVO)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDBDryRunFromProposalCourse] error: %v", err)
		return errorx.WrapByCode(err, errno.ErrCourseCvtFailed)
	}
	if dryRunCourse == nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDBDryRunFromProposalCourse] course is nil")
		return errorx.New(errno.ErrCourseCvtFailed)
	}

	courseExists, err := s.CourseRepo.IsCourseInExistingCourses(ctx, dryRunCourse)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [IsCourseInExistingCourses] error: %v", err)
		return errorx.WrapByCode(err, errno.ErrCourseCreateFailed)
	}
	if courseExists {
		logs.CtxInfof(ctx, "[ProposalService] [ApproveProposal] course already exists, skip create, proposalId: %s", proposal.ID)
		return nil
	}

	// 3. 创建新课程并写入来源提案ID
	course, err := s.CourseAssembler.ToCourseDBFromProposalCourse(ctx, courseVO)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDBFromProposalCourse] error: %v", err)
		return errorx.WrapByCode(err, errno.ErrCourseCvtFailed)
	}
	if course == nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDBFromProposalCourse] course is nil")
		return errorx.New(errno.ErrCourseCvtFailed)
	}

	course.ID = primitive.NewObjectID().Hex()
	course.CreatedAt = time.Now()
	course.UpdatedAt = time.Now()
	course.Deleted = false
	course.ProposalID = proposal.ID

	if err := s.CourseRepo.Insert(ctx, course); err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [Insert] error: %v", err)
		return errorx.WrapByCode(err, errno.ErrCourseCreateFailed, errorx.KV("name", course.Name))
	}
	return nil
}

// settleContributionStrict is part of approval's MongoDB transaction. Any write
// failure aborts the entire approval instead of leaving contribution half-settled.
func (s *ProposalService) settleContributionStrict(ctx context.Context, proposal *model.Proposal, courseVO *dto.ProposalCourseVO) error {
	originalVO, err := s.CourseAssembler.ToProposalCourseVO(ctx, proposal.Course)
	if err != nil {
		return fmt.Errorf("convert original proposal course: %w", err)
	}

	score := calcContributionScore(originalVO, courseVO)
	if score <= 0 {
		return nil
	}

	if err := s.UserRepo.IncrementContribution(ctx, proposal.UserID, score); err != nil {
		return fmt.Errorf("increment user contribution: %w", err)
	}
	if err := s.ProposalRepo.UpdateContributionByID(ctx, proposal.ID, score); err != nil {
		return fmt.Errorf("record proposal contribution: %w", err)
	}
	return nil
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
	return calcContributionScore(originalVO, finalVO)
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

// RevokeProposal 撤回提案操作（通过/拒绝）
func (s *ProposalService) RevokeProposal(ctx context.Context, req *dto.RevokeProposalReq) (*dto.RevokeProposalResp, error) {
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
	err = s.ProposalRepo.WithTransaction(ctx, func(txCtx mongo.SessionContext) error {
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
			if proposal.Status != approvedStatusID {
				return errorx.New(errno.ErrProposalStatusNotApproved, errorx.KV("proposalId", req.ProposalID))
			}
			associatedCourse, courseErr := s.CourseRepo.FindByProposalIDIncludeDeleted(txCtx, req.ProposalID)
			if courseErr != nil {
				return errorx.WrapByCode(courseErr, errno.ErrCourseNotFoundCannotRevoke)
			}
			if associatedCourse != nil {
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
		if invalidateErr := s.UserRepo.InvalidateByID(ctx, proposalUserID); invalidateErr != nil {
			logs.CtxWarnf(ctx, "[UserRepo] post-revoke cache invalidation failed: %v", invalidateErr)
		}
		if refreshErr := mapping.Data.Refresh(ctx); refreshErr != nil {
			logs.CtxWarnf(ctx, "[Mapping] post-revoke refresh failed: %v", refreshErr)
		}
	}

	return &dto.RevokeProposalResp{
		Resp:       dto.Success(),
		ProposalID: req.ProposalID,
	}, nil
}

// RejectProposal 拒绝提案，将状态从 pending 改为 rejected
func (s *ProposalService) RejectProposal(ctx context.Context, req *dto.RejectProposalReq) (*dto.RejectProposalResp, error) {
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [IsAdminByID] error: %v, userId: %s", err, userId)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed, errorx.KV("userId", userId))
	}
	if !isAdmin {
		return nil, errorx.New(errno.ErrUserNotAdmin, errorx.KV("userId", userId))
	}

	if req.ProposalID == "" {
		return nil, errorx.New(errno.ErrProposalIDRequired, errorx.KV("key", consts.ReqProposalID))
	}

	proposal, err := s.ProposalRepo.FindByID(ctx, req.ProposalID)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindByID] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed, errorx.KV("proposalId", req.ProposalID))
	}
	if proposal == nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindByID] proposal not found, proposalId: %s", req.ProposalID)
		return nil, errorx.New(errno.ErrProposalNotFound, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
	}

	pendingStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
	rejectedStatusID := mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected)
	if proposal.Status != pendingStatusID {
		return nil, errorx.New(errno.ErrProposalAlreadyProcessed, errorx.KV("key", consts.ReqProposalID), errorx.KV("value", req.ProposalID))
	}

	newStatusID := rejectedStatusID
	updated, err := s.ProposalRepo.UpdateStatusAndReasonByID(ctx, req.ProposalID, pendingStatusID, newStatusID, req.Reason)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [UpdateStatusAndReasonByID] error: %v, proposalId: %s", err, req.ProposalID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}
	if !updated {
		return nil, errorx.New(errno.ErrProposalUpdateFailed, errorx.KV("proposalId", req.ProposalID))
	}

	content := "审批提案：拒绝"
	if req.Reason != "" {
		content = req.Reason
	}

	if _, err = s.ChangeLogService.CreateChangeLog(ctx, &dto.CreateChangeLogReq{
		TargetID:     req.ProposalID,
		TargetType:   consts.TargetTypeProposal,
		Action:       consts.ActionTypeRejectProposal,
		Content:      content,
		UpdateSource: consts.UpdateSourceAdmin,
		ProposalID:   req.ProposalID,
	}); err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogService] [CreateChangeLog] error: %v, proposalId: %s", err, req.ProposalID)
	}

	_, pendingCount, err := s.ProposalRepo.FindManyByStatus(ctx, nil, pendingStatusID)
	if err != nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [FindManyByStatus] error: %v", err)
	}

	return &dto.RejectProposalResp{
		Resp:         dto.Success(),
		Rejected:     true,
		PendingCount: pendingCount,
	}, nil
}
