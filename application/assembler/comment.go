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
	"github.com/zeromicro/go-zero/core/stores/monc"
)

var _ ICommentAssembler = (*CommentAssembler)(nil)

type ICommentAssembler interface {
	ToCommentVO(ctx context.Context, db *model.Comment, userId string) (*dto.CommentVO, error)
	ToCommentDB(ctx context.Context, vo *dto.CommentVO) (*model.Comment, error)
	TOMyCommentVO(ctx context.Context, db *model.Comment, userId string) (*dto.CommentVO, error)
	ToMyCommentVOArray(ctx context.Context, dbs []*model.Comment, userId string) ([]*dto.CommentVO, error)
	ToCommentVOArray(ctx context.Context, dbs []*model.Comment, userId string) ([]*dto.CommentVO, error)
	ToCommentDBArray(ctx context.Context, vos []*dto.CommentVO) ([]*model.Comment, error)
}

type CommentAssembler struct {
	LikeRepo    *repo.LikeRepo
	CourseRepo  *repo.CourseRepo
	TeacherRepo *repo.TeacherRepo
}

var CommentAssemblerSet = wire.NewSet(
	wire.Struct(new(CommentAssembler), "*"),
	wire.Bind(new(ICommentAssembler), new(*CommentAssembler)),
)

// ToCommentVO 单个CommentDB转CommentVO (DB to VO) 包含点赞信息查询
func (a *CommentAssembler) ToCommentVO(ctx context.Context, db *model.Comment, userId string) (*dto.CommentVO, error) {
	// 获取点赞信息
	likeCnt, err := a.LikeRepo.CountByTarget(ctx, db.ID, consts.CommentType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [CountByTarget] error: %v", err)
		return nil, err
	}

	// 这里的userId是查看评论的用户
	active, err := a.LikeRepo.IsLike(ctx, userId, db.ID, consts.CommentType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [IsLike] error: %v", err)
		return nil, err
	}

	return &dto.CommentVO{
		ID:       db.ID,
		Content:  db.Content,
		Tags:     db.Tags,
		UserID:   db.UserID,
		CourseID: db.CourseID,
		LikeVO: &dto.LikeVO{
			Like:    active,
			LikeCnt: likeCnt,
		},
		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}, nil
}

// ToCommentDB 单个CommentVO转Comment (VO to DB)
func (a *CommentAssembler) ToCommentDB(ctx context.Context, vo *dto.CommentVO) (*model.Comment, error) {
	if vo == nil {
		return nil, nil
	}

	return &model.Comment{
		ID:        vo.ID,
		Content:   vo.Content,
		Tags:      vo.Tags,
		UserID:    vo.UserID,
		CourseID:  vo.CourseID,
		CreatedAt: vo.CreatedAt,
		UpdatedAt: vo.UpdatedAt,
		Deleted:   false, // 默认为未删除
	}, nil
}

// TOMyCommentVO 单个Comment转MyCommentVO(with 4 Extra fields) (DB to VO)
func (a *CommentAssembler) TOMyCommentVO(ctx context.Context, db *model.Comment, userId string) (*dto.CommentVO, error) {
	// 先获取除了Extra以外的字段
	vo, err := a.ToCommentVO(ctx, db, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentAssembler] [ToCommentVO] error: %v", err)
		return nil, err
	}

	// 获取Extra course相关
	var course *model.Course
	course, err = a.CourseRepo.FindByID(ctx, db.CourseID)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [FindByID] error: %v", err)
		return nil, err
	}
	if course == nil {
		return nil, monc.ErrNotFound
	}

	// 获取Extra teacher相关
	var teachersNameAndTitle []string
	for _, teacherID := range course.TeacherIDs {
		teacher, err := a.TeacherRepo.FindByID(ctx, teacherID)
		if err != nil {
			logs.CtxErrorf(ctx, "[TeacherRepo] [FindByID] error: %v", err)
			continue
		}
		if teacher != nil {
			teachersNameAndTitle = append(teachersNameAndTitle, teacher.Name+teacher.Title)
		}
	}

	// 组合rawVO和Extra得到MyCommentVO
	vo.Name = course.Name
	vo.Category = mapping.Data.GetCategoryNameByID(course.Category)
	vo.Department = mapping.Data.GetDepartmentNameByID(course.Department)
	vo.Teachers = teachersNameAndTitle

	return vo, nil
}

// ToMyCommentVOArray Comment数组转MyCommentVO数组(with 4 extra fields) (DB Array to VO Array)
func (a *CommentAssembler) ToMyCommentVOArray(ctx context.Context, dbs []*model.Comment, userId string) ([]*dto.CommentVO, error) {
	if len(dbs) == 0 {
		logs.CtxWarnf(ctx, "[CommentAssembler] [ToMyCommentVOArray] empty comment db array")
		return []*dto.CommentVO{}, nil
	}

	vos := make([]*dto.CommentVO, 0, len(dbs))

	for _, db := range dbs {
		vo, err := a.TOMyCommentVO(ctx, db, userId)
		if err != nil {
			logs.CtxErrorf(ctx, "[CommentAssembler] [TOMyCommentVO] error: %v", err)
			return nil, err
		}
		if vo != nil {
			vos = append(vos, vo)
		}
	}

	return vos, nil
}

// ToCommentDBArray CommentVO数组转Comment数组 (VO Array to DB Array)

func (a *CommentAssembler) ToCommentDBArray(ctx context.Context, vos []*dto.CommentVO) ([]*model.Comment, error) {
	if len(vos) == 0 {
		logs.CtxWarnf(ctx, "[CommentAssembler] [ToCommentDBArray] empty comment vo array")
		return []*model.Comment{}, nil
	}

	dbs := make([]*model.Comment, 0, len(vos))

	for _, vo := range vos {
		db, err := a.ToCommentDB(ctx, vo)
		if err != nil {
			logs.CtxErrorf(ctx, "[CommentAssembler] [ToCommentDB] error: %v", err)
			return nil, err
		}
		if db != nil {
			dbs = append(dbs, db)
		}
	}

	return dbs, nil
}

// ToCommentVOArray Comment数组转CommentVO数组 (DB Array to VO Array)
func (a *CommentAssembler) ToCommentVOArray(ctx context.Context, dbs []*model.Comment, userId string) ([]*dto.CommentVO, error) {
	if len(dbs) == 0 {
		logs.CtxWarnf(ctx, "[CommentAssembler] [ToCommentVOArray] empty comment db array")
		return []*dto.CommentVO{}, nil
	}

	// 提取所有 commentID
	ids := make([]string, len(dbs))
	for i, db := range dbs {
		ids[i] = db.ID
	}

	// 批量获取点赞数
	likeCntMap, err := a.LikeRepo.CountByTargets(ctx, ids, consts.CommentType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [CountByTargets] error: %v", err)
		return nil, err
	}

	// 批量获取点赞状态
	likeStatusMap, err := a.LikeRepo.GetLikesByUserIDAndTargets(ctx, userId, ids, consts.CommentType)
	if err != nil {
		logs.CtxErrorf(ctx, "[LikeRepo] [GetLikesByUserIDAndTargets] error: %v", err)
		return nil, err
	}

	// 构建结果
	vos := make([]*dto.CommentVO, 0, len(dbs))
	for _, db := range dbs {
		// 从批量查询结果中获取点赞信息
		likeCnt := likeCntMap[db.ID]   // 如果不存在则为0
		active := likeStatusMap[db.ID] // 如果不存在则为false
		commentVO := &dto.CommentVO{
			ID:       db.ID,
			Content:  db.Content,
			Tags:     db.Tags,
			UserID:   db.UserID,
			CourseID: db.CourseID,
			LikeVO: &dto.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			CreatedAt: db.CreatedAt,
			UpdatedAt: db.UpdatedAt,
		}
		vos = append(vos, commentVO)
	}

	return vos, nil
}
