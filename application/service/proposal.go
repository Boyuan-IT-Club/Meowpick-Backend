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
	ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error)
	GetProposal(ctx context.Context, req *dto.GetProposalReq) (*dto.GetProposalResp, error)
	DeleteProposal(ctx context.Context, req *dto.DeleteProposalReq) (*dto.DeleteProposalResp, error)
	UpdateProposal(ctx context.Context, req *dto.UpdateProposalReq) (*dto.UpdateProposalResp, error)
}

type ProposalService struct {
	CourseRepo        *repo.CourseRepo
	CourseAssembler   *assembler.CourseAssembler
	ProposalRepo      *repo.ProposalRepo
	ProposalAssembler *assembler.ProposalAssembler
	LikeRepo          *repo.LikeRepo
	LikeCache         *cache.LikeCache
	UserRepo          *repo.UserRepo
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
