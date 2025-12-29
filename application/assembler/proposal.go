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

package assembler

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ IProposalAssembler = (*ProposalAssembler)(nil)

type IProposalAssembler interface {
	ToProposalVO(ctx context.Context, db *model.Proposal, userId string) (*dto.ProposalVO, error)
	ToProposalVOArray(ctx context.Context, dbs []*model.Proposal, userId string) ([]*dto.ProposalVO, error)
	ToProposalDB(ctx context.Context, vo *dto.ProposalVO) (*model.Proposal, error)
	ToProposalDBArray(ctx context.Context, vos []*dto.ProposalVO) ([]*model.Proposal, error)
}

type ProposalAssembler struct {
	CourseAssembler *CourseAssembler
	LikeRepo        *repo.LikeRepo
}

var ProposalAssemblerSet = wire.NewSet(
	wire.Struct(new(ProposalAssembler), "*"),
	wire.Bind(new(IProposalAssembler), new(*ProposalAssembler)),
)

// ToProposalVO 单个ProposalDB转ProposalVO (DB to VO)
func (a *ProposalAssembler) ToProposalVO(ctx context.Context, db *model.Proposal, userId string) (*dto.ProposalVO, error) {
	var courseVO *dto.CourseVO
	if db.Course != nil {
		var err error
		courseVO, err = a.CourseAssembler.ToCourseVO(ctx, db.Course)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseVO] error: %v", err)
			return nil, err
		}
	}

	// 获得点赞目标类型
	targetType := mapping.Data.GetLikeTargetTypeIDByName(consts.LikeTargetTypeProposal)

	// 获取点赞信息
	likeCnt, err := a.LikeRepo.CountByTarget(ctx, db.ID, targetType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [CountByID] error: %v", err)
		return nil, err
	}

	// 这里的userId是查看评论的用户
	active, err := a.LikeRepo.IsLike(ctx, userId, db.ID, targetType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [IsLike] error: %v", err)
		return nil, err
	}

	return &dto.ProposalVO{
		ID:      db.ID,
		UserID:  db.UserID,
		Title:   db.Title,
		Content: db.Content,
		Course:  courseVO,
		Status:  mapping.Data.GetProposalStatusNameByID(db.Status),
		Deleted: db.Deleted,
		LikeVO: &dto.LikeVO{
			Like:    active,
			LikeCnt: likeCnt,
		},
		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}, nil
}

// ToProposalVOArray ProposalDB数组转ProposalVO数组 (DB Array to VO Array)
func (a *ProposalAssembler) ToProposalVOArray(ctx context.Context, dbs []*model.Proposal, userId string) ([]*dto.ProposalVO, error) {
	if len(dbs) == 0 {
		logs.CtxWarnf(ctx, "[ProposalAssembler] [ToProposalVOArray] empty proposal db array")
		return []*dto.ProposalVO{}, nil
	}

	// 提取所有 proposalIds
	ids := make([]string, len(dbs))
	for i, db := range dbs {
		ids[i] = db.ID
	}

	// 获得点赞目标类型
	targetType := mapping.Data.GetLikeTargetTypeIDByName(consts.LikeTargetTypeProposal)

	// 批量获取点赞数
	likeCntMap, err := a.LikeRepo.CountByTargets(ctx, ids, targetType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [CountByTargets] error: %v", err)
		return nil, err
	}

	// 批量获取点赞状态
	likeStatusMap, err := a.LikeRepo.GetLikesByUserIDAndTargets(ctx, userId, ids, targetType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [GetLikesByUserIDAndTargets] error: %v", err)
		return nil, err
	}

	// 构建结果
	vos := make([]*dto.ProposalVO, 0, len(dbs))
	for _, db := range dbs {
		// 从批量查询结果中获取点赞信息
		likeCnt := likeCntMap[db.ID]   // 如果不存在则为0
		active := likeStatusMap[db.ID] // 如果不存在则为false
		var courseVO *dto.CourseVO
		if db.Course != nil {
			courseVO, err = a.CourseAssembler.ToCourseVO(ctx, db.Course)
			if err != nil {
				logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseVO] error: %v", err)
				return nil, err
			}
		}
		proposalVO := &dto.ProposalVO{
			ID:      db.ID,
			Content: db.Content,
			Title:   db.Title,
			UserID:  db.UserID,
			Status:  mapping.Data.GetProposalStatusNameByID(db.Status),
			Deleted: db.Deleted,
			LikeVO: &dto.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			Course:    courseVO,
			CreatedAt: db.CreatedAt,
			UpdatedAt: db.UpdatedAt,
		}
		vos = append(vos, proposalVO)
	}

	return vos, nil
}

// ToProposalDB 单个ProposalVO转ProposalDB (VO to DB)
func (a *ProposalAssembler) ToProposalDB(ctx context.Context, vo *dto.ProposalVO) (*model.Proposal, error) {
	var courseDB *model.Course
	if vo.Course != nil {
		var err error
		courseDB, err = a.CourseAssembler.ToCourseDB(ctx, vo.Course)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDB] error: %v", err)
			return nil, err
		}
	}

	return &model.Proposal{
		ID:        vo.ID,
		UserID:    vo.UserID,
		Title:     vo.Title,
		Content:   vo.Content,
		Course:    courseDB,
		Status:    mapping.Data.GetProposalStatusIDByName(vo.Status),
		Deleted:   vo.Deleted,
		CreatedAt: vo.CreatedAt,
		UpdatedAt: vo.UpdatedAt,
	}, nil
}

// ToProposalDBArray ProposalVO数组转ProposalDB数组 (VO Array to DB Array)
func (a *ProposalAssembler) ToProposalDBArray(ctx context.Context, vos []*dto.ProposalVO) ([]*model.Proposal, error) {
	if len(vos) == 0 {
		logs.CtxWarnf(ctx, "[ProposalAssembler] [ToProposalDBArray] empty proposal vo array")
		return []*model.Proposal{}, nil
	}

	dbs := make([]*model.Proposal, 0, len(vos))
	for _, vo := range vos {
		db, err := a.ToProposalDB(ctx, vo)
		if err != nil {
			logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalDB] error: %v", err)
			return nil, err
		}
		if db != nil {
			dbs = append(dbs, db)
		}
	}

	return dbs, nil
}
