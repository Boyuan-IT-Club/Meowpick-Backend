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
	ToggleProposal(ctx context.Context, req *dto.ToggleProposalReq) (resp *dto.ToggleProposalResp, err error)
	ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error)
}

type ProposalService struct {
	CourseRepo        *repo.CourseRepo
	CourseAssembler   *assembler.CourseAssembler
	ProposalRepo      *repo.ProposalRepo
	ProposalCache     *cache.ProposalCache
	ProposalAssembler *assembler.ProposalAssembler
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
	courseDB, err := s.CourseAssembler.ToCourseDB(ctx, req.Course)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "database course"), errorx.KV("dst", "course vo"),
		)
	}

	// 检查是否已经存在相同的提案（前端传回的和我已有的ProposalCourse）
	existingProposal, err := s.ProposalRepo.IsCourseInExistingProposals(ctx, courseDB)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCourseFindInProposalsFailed,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}

	// 检查是否已经存在相同的课程（前端传回的的和我已有的course）
	existingCourse, err := s.CourseRepo.IsCourseInExistingCourses(ctx, courseDB)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCourseFindInCoursesFailed,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}

	if existingProposal == true {
		return nil, errorx.New(errno.ErrProposalCourseFoundInProposals,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}
	if existingCourse == true {
		return nil, errorx.New(errno.ErrProposalCourseFoundInCourses,
			errorx.KV("key", consts.ReqCourse),
			errorx.KV("value", req.Course.Name),
		)
	}
	campuses := []int32{}
	for _, campus := range req.Course.Campuses {
		campuses = append(campuses, mapping.Data.GetCampusIDByName(campus))
	}

	// 构造课程信息
	course := model.Course{
		Name:       req.Course.Name,
		Code:       req.Course.Code,
		TeacherIDs: make([]string, 0),
		Department: mapping.Data.GetDepartmentIDByName(req.Course.Department),
		Category:   mapping.Data.GetCategoryIDByName(req.Course.Category),
		Campuses:   campuses,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 创建提案对象
	status := mapping.Data.GetProposalStatusIDByName(req.Status)
	proposal := &model.Proposal{
		ID:        primitive.NewObjectID().Hex(),
		UserID:    userId,
		Title:     req.Title,
		Content:   req.Content,
		Deleted:   false,
		Course:    &course,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存提案到数据库
	if err = s.ProposalRepo.Insert(ctx, proposal); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCreateFailed,
			errorx.KV(consts.ReqTitle, req.Title),
			errorx.KV(consts.CtxUserID, userId))
	}

	return &dto.CreateProposalResp{
		Resp:       dto.Success(),
		ProposalID: proposal.ID,
	}, nil
}

// ToggleProposal 切换投票状态
func (s *ProposalService) ToggleProposal(ctx context.Context, req *dto.ToggleProposalReq) (resp *dto.ToggleProposalResp, err error) {

	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 投票或取消投票目标
	active, err := s.ProposalRepo.Toggle(ctx, userId, req.TargetID, consts.ProposalType)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalToggleFailed)
	}

	// 设置缓存的投票状态
	if err = s.ProposalCache.SetStatusByUserIdAndTarget(ctx, userId, req.TargetID, active,
		consts.CacheProposalStatusTTL,
	); err != nil {
		logs.CtxWarnf(ctx, "[ProposalCache] [SetStatusByUserIdAndTarget] error: %v", err)
	}

	// 获取新的总投票数
	proposalCount, err := s.ProposalRepo.CountProposalByTarget(ctx, req.TargetID, consts.ProposalType)
	if err != nil {
		logs.CtxWarnf(ctx, "[ProposalRepo] [CountByTarget] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCountFailed,
			errorx.KV("key", consts.ReqTargetID), errorx.KV("value", req.TargetID))
	}

	// 构造响应并返回
	return &dto.ToggleProposalResp{
		Resp:        dto.Success(),
		Proposal:    active,
		ProposalCnt: proposalCount,
	}, nil
}

// ListProposals 分页查询所有提案，用于投票列表或管理端审核
func (s *ProposalService) ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询提案列表
	proposals, total, err := s.ProposalRepo.FindMany(ctx, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindMany] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
	}

	// 转换为VO
	vos, err := s.ProposalAssembler.ToProposalVOArray(ctx, proposals)
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
