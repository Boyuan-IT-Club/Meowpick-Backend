package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
	"time"
)

type ICommentService interface {
	CreateComment(ctx context.Context, req *cmd.CreateCommentReq) (*cmd.CreateCommentResp, error)
	GetTotalCommentsCount(ctx context.Context) (int64, error)
}

type CommentService struct {
	CommentMapper *comment.MongoMapper
	// CommentMapper comment.IMongoMapper
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

func (s *CommentService) CreateComment(ctx context.Context, req *cmd.CreateCommentReq) (*cmd.CreateCommentResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		log.Error("userID is empty or invalid")
		return nil, errorx.ErrGetUserIDFailed
	}

	now := time.Now()

	newComment := &comment.Comment{
		UserID:    userID,
		CourseID:  req.CourseID,
		Content:   req.Content,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
		Deleted:   false,
	}

	if err := s.CommentMapper.Insert(ctx, newComment); err != nil {
		return nil, err
	}

	resp := &cmd.CreateCommentResp{
		Resp:     cmd.Success(),
		UserID:   newComment.UserID,
		CourseID: newComment.CourseID,
		Content:  newComment.Content,
		Tags:     newComment.Tags,
		CreateAt: newComment.CreatedAt,
		UpdateAt: newComment.UpdatedAt,
	}

	return resp, nil
}

func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (int64, error) {
	return s.CommentMapper.CountAll(ctx)
}
