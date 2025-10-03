package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

type ICommentService interface {
	CreateComment(ctx context.Context, req *cmd.CreateCommentReq) (*cmd.CreateCommentResp, error)
	GetMyComments(ctx context.Context, req *cmd.GetMyCommentsReq) (*cmd.GetMyCommentsResp, error)
	GetCourseComments(ctx context.Context, req *cmd.GetCourseCommentsReq) (*cmd.GetCourseCommentsResp, error)
	GetTotalCommentsCount(ctx context.Context) (*cmd.GetTotalCommentsCountResp, error)
}

type CommentService struct {
	CommentMapper *comment.MongoMapper
	LikeMapper    *like.MongoMapper
	CourseMapper  *course.MongoMapper
	StaticData    *consts.StaticData
	CommentDto    *dto.CommentDTO
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
		log.CtxError(ctx, "Failed to insert comment for userID=%s: %v", userID, err)
		return nil, err
	}
	vo, err := s.CommentDto.ToCommentVO(ctx, newComment, userID)
	if err != nil {
		log.CtxError(ctx, "ToCommentVO failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrCommentDB2VO
	}
	resp := &cmd.CreateCommentResp{
		Resp:      cmd.Success(),
		CommentVO: vo,
	}

	return resp, nil
}

func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (*cmd.GetTotalCommentsCountResp, error) {
	count, err := s.CommentMapper.CountAll(ctx)
	if err != nil {
		log.CtxError(ctx, "Service GetTotalCommentCount failed: %v", err)
		return nil, errorx.ErrGetCountFailed
	}
	resp := &cmd.GetTotalCommentsCountResp{
		Resp:  cmd.Success(),
		Count: count,
	}

	return resp, nil
}

func (s *CommentService) GetMyComments(ctx context.Context, req *cmd.GetMyCommentsReq) (*cmd.GetMyCommentsResp, error) {
	// 获得用户id
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}
	// 构建查询参数
	param := &cmd.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	// 查找数据库获得comments
	comments, total, err := s.CommentMapper.FindManyByUserID(ctx, param, userID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrFindFailed
	}
	if total == 0 {
		return nil, errorx.ErrFindSuccessButNoResult
	}
	// 数据转化db2vo
	myCommentVOs, err := s.CommentDto.ToMyCommentVOList(ctx, comments, userID)
	if err != nil {
		log.CtxError(ctx, "ToMyCommentVOList failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrCommentDB2VO
	}
	// 构建GetMyComments响应，包含评论信息&部分课程信息
	resp := &cmd.GetMyCommentsResp{
		Resp:     cmd.Success(),
		Total:    total,
		Comments: myCommentVOs,
	}
	return resp, nil
}

func (s *CommentService) GetCourseComments(ctx context.Context, req *cmd.GetCourseCommentsReq) (*cmd.GetCourseCommentsResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	courseID := req.ID
	param := &cmd.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := s.CommentMapper.FindManyByCourseID(ctx, param, courseID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", courseID, err)
		return nil, errorx.ErrFindFailed
	}

	vos, err := s.CommentDto.ToCommentVOList(ctx, comments)
	if err != nil {
		log.CtxError(ctx, "ToCommentVOList failed for course: ", courseID, err)
		return nil, errorx.ErrCommentDB2VO
	}

	resp := &cmd.GetCourseCommentsResp{
		Resp:     cmd.Success(),
		Total:    total,
		Comments: vos,
	}

	return resp, nil
}
