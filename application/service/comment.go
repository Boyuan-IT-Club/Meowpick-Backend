package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
	"time"
)

type ICommentService interface {
	CreateComment(ctx context.Context, req *cmd.CreateCommentReq) (*cmd.CreateCommentResp, error)
	GetMyComments(ctx context.Context, req *cmd.GetMyCommentsReq) (*cmd.GetCommentsResp, error)
	GetCourseComments(ctx context.Context, req *cmd.GetCourseCommentsReq) (*cmd.GetCommentsResp, error)
	GetTotalCommentsCount(ctx context.Context) (*cmd.GetTotalCommentsCountResp, error)
}

type CommentService struct {
	CommentMapper *comment.MongoMapper
	LikeMapper    *like.MongoMapper
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

func (s *CommentService) CreateComment(ctx context.Context, req *cmd.CreateCommentReq) (*cmd.CreateCommentResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
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
		log.CtxError(ctx, "Failed to insert comment for userID=%s: %v", userID, err)
		return nil, err
	}

	resp := &cmd.CreateCommentResp{
		Resp: cmd.Success(),
		CommentVO: &cmd.CommentVO{
			UserID:    newComment.UserID,
			CourseID:  newComment.CourseID,
			Content:   newComment.Content,
			Tags:      newComment.Tags,
			CreatedAt: newComment.CreatedAt,
			UpdatedAt: newComment.UpdatedAt,
		},
	}

	return resp, nil
}

func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (int64, error) {
	return s.CommentMapper.CountAll(ctx)
}

func (s *CommentService) GetMyComments(ctx context.Context, req *cmd.GetMyCommentsReq) (*cmd.GetCommentsResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	comments, total, err := s.CommentMapper.FindManyByUserID(ctx, req, userID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrFindFailed
	}

	vos := make([]*cmd.CommentVO, 0, len(comments))
	for _, c := range comments {
		likeCnt, err := s.LikeMapper.GetLikeCount(ctx, c.ID, consts.CommentType)
		if err != nil {
			log.CtxError(ctx, "GetLikeCount failed for commentID=%s: %v", c.ID, err)
			return nil, errorx.ErrGetCountFailed
		}

		active, err := s.LikeMapper.GetLikeStatus(ctx, userID, c.ID, consts.CommentType)
		if err != nil {
			log.CtxError(ctx, "GetLikeStatus failed for userID=%s, commentID=%s: %v", userID, c.ID, err)
			return nil, errorx.ErrGetStatusFailed
		}

		vo := &cmd.CommentVO{
			ID:       c.ID,
			Content:  c.Content,
			Tags:     c.Tags,
			UserID:   c.UserID,
			CourseID: c.CourseID,
			LikeVO: &cmd.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
		vos = append(vos, vo)
	}

	resp := &cmd.GetCommentsResp{
		Resp:  cmd.Success(),
		Total: total,
		Rows:  vos,
	}

	return resp, nil
}

func (s *CommentService) GetCourseComments(ctx context.Context, req *cmd.GetCourseCommentsReq) (*cmd.GetCommentsResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	courseID := req.CourseID // TODO 调用search接口 校验courseID是否有效

	comments, total, err := s.CommentMapper.FindManyByCourseID(ctx, req, courseID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", courseID, err)
		return nil, errorx.ErrFindFailed
	}

	vos := make([]*cmd.CommentVO, 0, len(comments))
	for _, c := range comments {
		likeCnt, err := s.LikeMapper.GetLikeCount(ctx, c.ID, consts.CommentType)
		if err != nil {
			log.CtxError(ctx, "GetLikeCount failed for commentID=%s: %v", c.ID, err)
			return nil, errorx.ErrGetCountFailed
		}

		active, err := s.LikeMapper.GetLikeStatus(ctx, userID, c.ID, consts.CommentType)
		if err != nil {
			log.CtxError(ctx, "GetLikeStatus failed for userID=%s, commentID=%s: %v", courseID, c.ID, err)
			return nil, errorx.ErrGetStatusFailed
		}

		vo := &cmd.CommentVO{
			ID:       c.ID,
			Content:  c.Content,
			Tags:     c.Tags,
			UserID:   c.UserID,
			CourseID: c.CourseID,
			LikeVO: &cmd.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
		vos = append(vos, vo)
	}

	resp := &cmd.GetCommentsResp{
		Resp:  cmd.Success(),
		Total: total,
		Rows:  vos,
	}

	return resp, nil
}

func (s *CommentService) GetMyComments(ctx context.Context, req *cmd.GetMyCommentsReq) (*cmd.GetCommentsResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	comments, total, err := s.CommentMapper.FindManyByUserID(ctx, req, userID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrFindFailed
	}

	vos := make([]*cmd.CommentVO, 0, len(comments))
	for _, c := range comments {
		likeCnt, err := s.LikeMapper.GetLikeCount(ctx, c.ID, consts.CommentType)
		if err != nil {
			log.CtxError(ctx, "GetLikeCount failed for commentID=%s: %v", c.ID, err)
			return nil, errorx.ErrGetCountFailed
		}

		active, err := s.LikeMapper.GetLikeStatus(ctx, userID, c.ID, consts.CommentType)
		if err != nil {
			log.CtxError(ctx, "GetLikeStatus failed for userID=%s, commentID=%s: %v", userID, c.ID, err)
			return nil, errorx.ErrGetStatusFailed
		}

		vo := &cmd.CommentVO{
			ID:       c.ID,
			Content:  c.Content,
			Tags:     c.Tags,
			UserID:   c.UserID,
			CourseID: c.CourseID,
			LikeVO: &cmd.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
		vos = append(vos, vo)
	}

	resp := &cmd.GetCommentsResp{
		Resp:  cmd.Success(),
		Total: total,
		Rows:  vos,
	}

	return resp, nil
}
