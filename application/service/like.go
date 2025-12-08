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
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/google/wire"
)

var _ ILikeService = (*LikeService)(nil)

type ILikeService interface {
	Like(ctx context.Context, req *dto.CreateLikeReq) (resp *dto.LikeResp, err error)
}

type LikeService struct {
	LikeMapper *like.MongoRepo
}

var LikeServiceSet = wire.NewSet(
	wire.Struct(new(LikeService), "*"),
	wire.Bind(new(ILikeService), new(*LikeService)),
)

func (s *LikeService) Like(ctx context.Context, req *dto.CreateLikeReq) (resp *dto.LikeResp, err error) {
	// 参数校验
	var targetID string
	var userID string
	var ok bool
	if targetID = req.TargetID; targetID == "" {
		log.Error("targetID is empty or invalid")
		return nil, errorx.ErrEmptyTargetID
	}

	userID, ok = ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		log.Error("userID is empty or invalid")
		return nil, errorx.ErrGetUserIDFailed
	}

	// 步骤一：先执行点赞或取消点赞的操作
	newActive, err := s.LikeMapper.ToggleLike(ctx, userID, targetID, consts.CommentType)
	if err != nil {
		return nil, errorx.ErrLikeFailed
	}

	// 步骤二：操作完成后，再去获取最新的总点赞数
	likeCount, err := s.LikeMapper.GetLikeCount(ctx, targetID, consts.CommentType)
	if err != nil {
		return nil, errorx.ErrGetCountFailed
	}

	// 步骤三：使用两个最新的数据创建响应
	resp = &dto.LikeResp{
		Resp: dto.Success(),
		LikeVO: &dto.LikeVO{
			Like:    newActive,
			LikeCnt: likeCount, // <-- 现在 likeCount 是最新的准确数据了
		},
	}

	resp = &dto.LikeResp{
		Resp: dto.Success(),
		LikeVO: &dto.LikeVO{
			Like:    newActive,
			LikeCnt: likeCount, // <-- 现在 likeCount 是最新的准确数据了
		},
	}

	return resp, nil
}
