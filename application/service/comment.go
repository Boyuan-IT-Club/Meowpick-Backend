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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/google/wire"
)

var _ ICommentService = (*CommentService)(nil)

type ICommentService interface {
	CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CreateCommentResp, error)
	GetMyComments(ctx context.Context, req *dto.GetMyCommentsReq) (*dto.GetMyCommentsResp, error)
	GetCourseComments(ctx context.Context, req *dto.ListCourseCommentsReq) (*dto.ListCourseCommentsResp, error)
	GetTotalCommentsCount(ctx context.Context) (*dto.GetTotalCourseCommentsCountResp, error)
}

type CommentService struct {
	CommentRepo      *repo.CommentRepo
	LikeRepo         *repo.LikeRepo
	CourseRepo       *repo.CourseRepo
	CommentAssembler *assembler.CommentAssembler
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

func (s *CommentService) CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CreateCommentResp, error) {
	userID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userID == "" {
		log.Error("userID is empty or invalid")
		return nil, errorx.ErrGetUserIDFailed
	}

	now := time.Now()

	newComment := &model.Comment{
		UserID:    userID,
		CourseID:  req.CourseID,
		Content:   req.Content,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
		Deleted:   false,
	}

	if err := s.CommentRepo.Insert(ctx, newComment); err != nil {
		log.CtxError(ctx, "Failed to insert comment for userID=%s: %v", userID, err)
		return nil, err
	}
	vo, err := s.CommentAssembler.ToCommentVO(ctx, newComment, userID)
	if err != nil {
		log.CtxError(ctx, "ToCommentVO failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrCommentDB2VO
	}
	resp := &dto.CreateCommentResp{
		Resp:      dto.Success(),
		CommentVO: vo,
	}

	return resp, nil
}

func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (*dto.GetTotalCourseCommentsCountResp, error) {
	count, err := s.CommentRepo.Count(ctx)
	if err != nil {
		log.CtxError(ctx, "Service GetTotalCommentCount failed: %v", err)
		return nil, errorx.ErrGetCountFailed
	}
	resp := &dto.GetTotalCourseCommentsCountResp{
		Resp:  dto.Success(),
		Count: count,
	}

	return resp, nil
}

func (s *CommentService) GetMyComments(ctx context.Context, req *dto.GetMyCommentsReq) (*dto.GetMyCommentsResp, error) {
	// 获得用户id
	userID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}
	// 构建查询参数
	param := &dto.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	// 查找数据库获得comments
	comments, total, err := s.CommentRepo.FindManyByUserID(ctx, param, userID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrFindFailed
	}
	if total == 0 {
		return nil, errorx.ErrFindSuccessButNoResult
	}
	// 数据转化db2vo
	myCommentVOs, err := s.CommentAssembler.ToMyCommentVOList(ctx, comments, userID)
	if err != nil {
		log.CtxError(ctx, "ToMyCommentVOList failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrCommentDB2VO
	}
	// 构建GetMyComments响应，包含评论信息&部分课程信息
	resp := &dto.GetMyCommentsResp{
		Resp:     dto.Success(),
		Total:    total,
		Comments: myCommentVOs,
	}
	return resp, nil
}

func (s *CommentService) GetCourseComments(ctx context.Context, req *dto.ListCourseCommentsReq) (*dto.ListCourseCommentsResp, error) {
	userID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	courseID := req.ID
	param := &dto.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := s.CommentRepo.FindManyByCourseID(ctx, param, courseID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", courseID, err)
		return nil, errorx.ErrFindFailed
	}

	vos, err := s.CommentAssembler.ToCommentVOList(ctx, comments, userID)
	if err != nil {
		log.CtxError(ctx, "ToCommentVOList failed for course: ", courseID, err)
		return nil, errorx.ErrCommentDB2VO
	}

	resp := &dto.ListCourseCommentsResp{
		Resp:     dto.Success(),
		Total:    total,
		Comments: vos,
	}

	return resp, nil
}
