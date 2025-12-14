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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ ILikeService = (*LikeService)(nil)

type ILikeService interface {
	ToggleLike(ctx context.Context, req *dto.ToggleLikeReq) (resp *dto.ToggleLikeResp, err error)
}

type LikeService struct {
	LikeRepo  *repo.LikeRepo
	LikeCache *cache.LikeCache
}

var LikeServiceSet = wire.NewSet(
	wire.Struct(new(LikeService), "*"),
	wire.Bind(new(ILikeService), new(*LikeService)),
)

// ToggleLike 点赞或取消点赞评论
func (s *LikeService) ToggleLike(ctx context.Context, req *dto.ToggleLikeReq) (resp *dto.ToggleLikeResp, err error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 点赞或取消点赞目标
	active, err := s.LikeRepo.Toggle(ctx, userId, req.TargetID, consts.CommentType)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrLikeToggleFailed)
	}

	// 设置缓存的点赞状态
	if err = s.LikeCache.SetStatusByUserIdAndTarget(ctx, userId, req.TargetID, active,
		consts.CacheLikeStatusTTL,
	); err != nil {
		logs.CtxWarnf(ctx, "[LikeCache] [SetStatusByUserIdAndTarget] error: %v", err)
	}

	// 获取最新的总点赞数
	likeCount, err := s.LikeRepo.CountByTarget(ctx, req.TargetID, consts.CommentType)
	if err != nil {
		logs.CtxWarnf(ctx, "[LikeRepo] [CountByTarget] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrLikeCountFailed,
			errorx.KV("key", consts.ReqTargetID), errorx.KV("value", req.TargetID))
	}

	return &dto.ToggleLikeResp{
		Resp: dto.Success(),
		LikeVO: &dto.LikeVO{
			Like:    active,
			LikeCnt: likeCount,
		},
	}, nil
}
