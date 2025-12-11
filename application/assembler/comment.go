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

var _ ICommentAssembler = (*CommentAssembler)(nil)

type ICommentAssembler interface {
	ToCommentVO(ctx context.Context, c *model.Comment, userID string) (*dto.CommentVO, error)
	ToComment(ctx context.Context, vo *dto.CommentVO) (*model.Comment, error)
	TOMyCommentVO(ctx context.Context, c *model.Comment, userID string) (*dto.CommentVO, error)
	ToMyCommentVOList(ctx context.Context, comments []*model.Comment, userID string) ([]*dto.CommentVO, error)
	ToCommentVOList(ctx context.Context, comments []*model.Comment, userID string) ([]*dto.CommentVO, error)
	ToCommentList(ctx context.Context, vos []*dto.CommentVO) ([]*model.Comment, error)
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

// ToCommentVO 单个Comment转CommentVO (DB to VO) 包含点赞信息查询
func (a *CommentAssembler) ToCommentVO(ctx context.Context, c *model.Comment, userID string) (*dto.CommentVO, error) {
	// 获取点赞信息
	likeCnt, err := a.LikeRepo.GetLikeCount(ctx, c.ID, consts.CommentType)
	if err != nil {
		logs.CtxErrorf(ctx, "GetLikeCount failed for commentID=%s: %v", c.ID, err)
		return nil, err
	}

	// 这里的userID是查看评论的用户，而非评论作者
	active, err := a.LikeRepo.GetLikeStatus(ctx, userID, c.ID, consts.CommentType)
	if err != nil {
		logs.CtxErrorf(ctx, "GetLikeStatus failed for userID=%s, commentID=%s: %v", userID, c.ID, err)
		return nil, err
	}

	return &dto.CommentVO{
		ID:       c.ID,
		Content:  c.Content,
		Tags:     c.Tags,
		UserID:   c.UserID,
		CourseID: c.CourseID,
		LikeVO: &dto.LikeVO{
			Like:    active,
			LikeCnt: likeCnt,
		},
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

// ToComment 单个CommentVO转Comment (VO to DB)
func (a *CommentAssembler) ToComment(ctx context.Context, vo *dto.CommentVO) (*model.Comment, error) {
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
func (a *CommentAssembler) TOMyCommentVO(ctx context.Context, c *model.Comment, userID string) (*dto.CommentVO, error) {
	// 先获取除了Extra以外的字段
	myCommentVO, err := a.ToCommentVO(ctx, c, userID)
	if err != nil {
		return nil, err
	}

	// 获取Extra course相关
	var cou *model.Course
	cou, err = a.CourseRepo.FindByID(ctx, c.CourseID)
	if err != nil {
		return nil, err
	}

	// 获取Extra teacher相关
	var teachersNameAndTitle []string
	for _, teacherID := range cou.TeacherIDs {
		t, err := a.TeacherRepo.FindByID(ctx, teacherID)
		if err != nil {
			continue
		}
		teachersNameAndTitle = append(teachersNameAndTitle, t.Name+t.Title)
	}
	// 组合rawVO和Extra得到MyCommentVO
	myCommentVO.Name = cou.Name
	myCommentVO.Category = mapping.Data.GetCategoryNameByID(cou.Category)
	myCommentVO.Department = mapping.Data.GetDepartmentNameByID(cou.Department)
	myCommentVO.Teachers = teachersNameAndTitle

	return myCommentVO, nil
}

// ToMyCommentVOList Comment数组转MyCommentVO数组(with 4 extra fields) (DB Array to VO Array)
func (a *CommentAssembler) ToMyCommentVOList(ctx context.Context, comments []*model.Comment, userID string) ([]*dto.CommentVO, error) {
	if len(comments) == 0 {
		log.CtxError(ctx, "ToMyCommentVOList: comments is empty")
		return []*dto.CommentVO{}, nil
	}
	commentVOs := make([]*dto.CommentVO, 0, len(comments))

	for _, c := range comments {
		commentVO, err := a.TOMyCommentVO(ctx, c, userID)
		if err != nil {
			return nil, err
		}
		if commentVO != nil {
			commentVOs = append(commentVOs, commentVO)
		}
	}

	return commentVOs, nil
}

// ToCommentList CommentVO数组转Comment数组 (VO Array to DB Array)

func (a *CommentAssembler) ToCommentList(ctx context.Context, vos []*dto.CommentVO) ([]*model.Comment, error) {
	if len(vos) == 0 {
		return []*model.Comment{}, nil
	}

	comments := make([]*model.Comment, 0, len(vos))

	for _, vo := range vos {
		dbComment, err := a.ToComment(ctx, vo)
		if err != nil {
			return nil, err
		}
		if dbComment != nil {
			comments = append(comments, dbComment)
		}
	}

	return comments, nil
}

// ToCommentVOList Comment数组转CommentVO数组 (DB Array to VO Array)
func (a *CommentAssembler) ToCommentVOList(ctx context.Context, comments []*model.Comment, userID string) ([]*dto.CommentVO, error) {
	if len(comments) == 0 {
		log.CtxInfo(ctx, "ToCommentVOList: comments is empty")
		return []*dto.CommentVO{}, nil
	}

	// 提取所有 commentID
	commentIDs := make([]string, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}

	// 批量获取点赞数
	likeCountMap, err := a.LikeRepo.GetBatchLikeCount(ctx, userID, commentIDs, consts.CommentType)
	if err != nil {
		log.CtxError(ctx, "GetBatchLikeCount failed: %v", err)
		return nil, err
	}

	// 批量获取点赞状态
	likeStatusMap, err := a.LikeRepo.GetBatchLikeStatus(ctx, userID, commentIDs, consts.CommentType)
	if err != nil {
		log.CtxError(ctx, "GetBatchLikeStatus failed: %v", err)
		return nil, err
	}

	// 构建结果
	commentVOs := make([]*dto.CommentVO, 0, len(comments))
	for _, c := range comments {
		// 从批量查询结果中获取点赞信息
		likeCnt := likeCountMap[c.ID] // 如果不存在则为0
		active := likeStatusMap[c.ID] // 如果不存在则为false

		commentVO := &dto.CommentVO{
			ID:       c.ID,
			Content:  c.Content,
			Tags:     c.Tags,
			UserID:   c.UserID,
			CourseID: c.CourseID,
			LikeVO: &dto.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
		commentVOs = append(commentVOs, commentVO)
	}

	return commentVOs, nil
}
