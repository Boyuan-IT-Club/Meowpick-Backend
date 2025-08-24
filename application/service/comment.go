package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/google/wire"
	"time"
)

type ICommentService interface {
	CreateComment(ctx context.Context, req *cmd.CreateCommentReq, userId string) (*cmd.CreateCommentResp, error)
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

func (s *CommentService) CreateComment(ctx context.Context, req *cmd.CreateCommentReq, userId string) (*cmd.CreateCommentResp, error) {
	now := time.Now()

	newComment := &comment.Comment{
		UserID:    userId,
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

	commentResp := &cmd.ResponseComment{
		UserID:   newComment.UserID,
		CourseID: newComment.CourseID,
		Content:  newComment.Content,
		Tags:     newComment.Tags,
		CreateAt: newComment.CreatedAt,
		UpdateAt: newComment.UpdatedAt,
	}

	resp := &cmd.CreateCommentResp{
		Code:    200,
		Msg:     "success",
		Comment: commentResp,
	}

	return resp, nil
}

func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (int64, error) {
	return s.CommentMapper.CountAll(ctx)
}
