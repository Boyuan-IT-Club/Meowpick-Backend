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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ ICommentService = (*CommentService)(nil)

type ICommentService interface {
	CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CreateCommentResp, error)

	GetTotalCommentsCount(ctx context.Context) (*dto.GetTotalCourseCommentsCountResp, error)
	GetMyComments(ctx context.Context, req *dto.GetMyCommentsReq) (*dto.GetMyCommentsResp, error)
	GetCourseComments(ctx context.Context, req *dto.ListCourseCommentsReq) (*dto.ListCourseCommentsResp, error)
}

type CommentService struct {
	CommentRepo      *repo.CommentRepo
	CommentCache     *cache.CommentCache
	CommentAssembler *assembler.CommentAssembler
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

// CreateComment 创建评论
func (s *CommentService) CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CreateCommentResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 构建Comment模型
	now := time.Now()
	comment := &model.Comment{
		UserID:    userId,
		CourseID:  req.CourseID,
		Content:   req.Content,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
		Deleted:   false,
	}

	// 插入数据库
	if err := s.CommentRepo.Insert(ctx, comment); err != nil {
		logs.CtxErrorf(ctx, "[CommentRepo] [Insert] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentInsertFailed, errorx.KV("content", req.Content))
	}

	// 转换为VO
	vo, err := s.CommentAssembler.ToCommentVO(ctx, comment, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentAssembler] [ToCommentVO] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentCvtFailed,
			errorx.KV("src", "database comment"), errorx.KV("dst", "comment vo"))
	}

	return &dto.CreateCommentResp{
		Resp:      dto.Success(),
		CommentVO: vo,
	}, nil
}

// GetTotalCommentsCount 获得课程总评论数
func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (*dto.GetTotalCourseCommentsCountResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询缓存
	count, ok, err := s.CommentCache.GetCount(ctx)
	if err == nil && ok {
		return &dto.GetTotalCourseCommentsCountResp{
			Count: count,
			Resp:  dto.Success(),
		}, nil
	}
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentCache] [GetCount] error: %v", err)
	}

	// 查询总评论数
	count, err = s.CommentRepo.Count(ctx)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentRepo] [Count] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentCountFailed)
	}

	// 设置缓存
	if err = s.CommentCache.SetCount(ctx, count, consts.CacheCommentCountTTL); err != nil {
		logs.CtxErrorf(ctx, "[CommentCache] [SetCount] error: %v", err)
	}

	return &dto.GetTotalCourseCommentsCountResp{
		Count: count,
		Resp:  dto.Success(),
	}, nil
}

// GetMyComments 分页获取用户所有评论
func (s *CommentService) GetMyComments(ctx context.Context, req *dto.GetMyCommentsReq) (*dto.GetMyCommentsResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询评论列表
	comments, total, err := s.CommentRepo.FindManyByUserID(ctx, req.PageParam, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentRepo] [FindManyByUserID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId))
	}

	// 转换为VO
	vos, err := s.CommentAssembler.ToMyCommentVOList(ctx, comments, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentAssembler] [ToMyCommentVOList] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentCvtFailed,
			errorx.KV("src", "database comments"), errorx.KV("dst", "comment vos"))
	}

	return &dto.GetMyCommentsResp{
		Resp:     dto.Success(),
		Total:    total,
		Comments: vos,
	}, nil
}

// GetCourseComments 分页获取课程所有评论
func (s *CommentService) GetCourseComments(ctx context.Context, req *dto.ListCourseCommentsReq) (*dto.ListCourseCommentsResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询评论列表
	comments, total, err := s.CommentRepo.FindManyByCourseID(ctx, req.PageParam, req.ID)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentRepo] [FindManyByCourseID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentFindFailed,
			errorx.KV("key", consts.ReqCourseID), errorx.KV("value", req.ID))
	}

	// 转换为VO
	vos, err := s.CommentAssembler.ToCommentVOList(ctx, comments, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[CommentAssembler] [ToCommentVOList] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCommentCvtFailed,
			errorx.KV("src", "database comments"), errorx.KV("dst", "comment vos"))
	}

	return &dto.ListCourseCommentsResp{
		Resp:     dto.Success(),
		Total:    total,
		Comments: vos,
	}, nil
}
