package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ICommentService interface {
	CreateComment(ctx context.Context, req *cmd.CreateCommentCmd, userId string) (*cmd.CreateCommentResp, error)
}

type CommentService struct {
	CommentMapper *comment.MongoMapper
	// CommentMapper comment.IMongoMapper
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

func (s *CommentService) CreateComment(ctx context.Context, req *cmd.CreateCommentCmd, userId string) (*cmd.CreateCommentResp, error) {
	uid, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, err
	}
	cid, err := primitive.ObjectIDFromHex(req.CourseID)
	if err != nil {
		return nil, err
	}

	newComment := &comment.Comment{
		UserID:   uid,
		CourseID: cid,
		Content:  req.Content,
		Tags:     req.Tags,
	}

	if err = s.CommentMapper.Insert(ctx, newComment); err != nil {
		return nil, err
	}

	resp := &cmd.CreateCommentResp{
		Code:    200,
		Msg:     "success",
		Comment: newComment,
	}

	return resp, nil
}
