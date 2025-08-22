package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

type ILikeService interface {
	Like(ctx context.Context, req *cmd.CreateLikeReq) (resp *cmd.LikeResp, err error)
}

type LikeService struct {
	LikeMapper *like.MongoMapper
}

var LikeServiceSet = wire.NewSet(
	wire.Struct(new(LikeService), "*"),
	wire.Bind(new(ILikeService), new(*LikeService)),
)

func (s *LikeService) Like(ctx context.Context, req *cmd.CreateLikeReq) (resp *cmd.LikeResp, err error) {
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

	// 填充响应参数
	var likeCount int64 // 根据前端乐观更新设计，这里需要先查询原点赞数，再进行这次的“点赞”操作
	var newActive bool  // 新的点赞状态

	if likeCount, err = s.LikeMapper.GetLikeCount(ctx, targetID, consts.CommentType); err != nil {
		return nil, errorx.ErrGetCountFailed
	}
	if newActive, err = s.LikeMapper.ToggleLike(ctx, userID, targetID, consts.CommentType); err != nil {
		return nil, errorx.ErrLikeFailed
	}

	// 创建响应并返回
	resp = &cmd.LikeResp{
		Like:    newActive,
		LikeCnt: likeCount,
	}

	return resp, nil
}
