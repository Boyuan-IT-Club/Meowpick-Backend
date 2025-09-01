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
	resp = &cmd.LikeResp{
		Resp: cmd.Success(),
		LikeVO: &cmd.LikeVO{
			Like:    newActive,
			LikeCnt: likeCount, // <-- 现在 likeCount 是最新的准确数据了
		},
	}

	resp = &cmd.LikeResp{
		Resp: cmd.Success(),
		LikeVO: &cmd.LikeVO{
			Like:    newActive,
			LikeCnt: likeCount, // <-- 现在 likeCount 是最新的准确数据了
		},
	}

	return resp, nil
}
