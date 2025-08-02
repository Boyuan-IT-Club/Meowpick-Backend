package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ICommentService interface {
	CreateComment(ctx context.Context, req *cmd.CreateCommentCmd, userId string) (*comment.Comment, error)
}

type CommentService struct {
	CommentMapper *comment.MongoMapper
	// CommentMapper comment.IMongoMapper
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

func (s *CommentService) CreateComment(ctx context.Context, req *cmd.CreateCommentCmd, userId string) (*comment.Comment, error) {
	userIDObj, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, err
	}
	courseIDObj, err := primitive.ObjectIDFromHex(req.CourseID)
	if err != nil {
		return nil, err
	}

	newComment := &comment.Comment{
		UserID:   userIDObj,
		CourseID: courseIDObj,
		Content:  req.Content,
		Tags:     req.Tags,
	}

	err = s.CommentMapper.Insert(ctx, newComment)
	if err != nil {
		return nil, err
	}

	return newComment, nil
}
