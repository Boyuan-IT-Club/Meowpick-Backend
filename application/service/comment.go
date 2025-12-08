package service

import (
	"context"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

var _ ICommentService = (*CommentService)(nil)

type ICommentService interface {
	CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CreateCommentResp, error)
	GetMyComments(ctx context.Context, req *dto.GetMyCommentsReq) (*dto.GetMyCommentsResp, error)
	GetCourseComments(ctx context.Context, req *dto.GetCourseCommentsReq) (*dto.GetCourseCommentsResp, error)
	GetTotalCommentsCount(ctx context.Context) (*dto.GetTotalCommentsCountResp, error)
}

type CommentService struct {
	CommentRepo *comment.MongoRepo
	LikeRepo    *like.MongoRepo
	CourseRepo  *course.MongoRepo
	StaticData  *mapping.StaticData
	CommentDto  *assembler.CommentDTO
}

var CommentServiceSet = wire.NewSet(
	wire.Struct(new(CommentService), "*"),
	wire.Bind(new(ICommentService), new(*CommentService)),
)

func (s *CommentService) CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CreateCommentResp, error) {
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

	if err := s.CommentRepo.Insert(ctx, newComment); err != nil {
		log.CtxError(ctx, "Failed to insert comment for userID=%s: %v", userID, err)
		return nil, err
	}
	vo, err := s.CommentDto.ToCommentVO(ctx, newComment, userID)
	if err != nil {
		log.CtxError(ctx, "ToCommentVO failed for userID=%s: %v", userID, err)
		return nil, errorx.ErrCommentDB2VO
	}
	resp := &dto.CreateCommentResp{
		Resp:      dto.Success(),
		CommentVO: vo,
	}

	return resp, nil
}

func (s *CommentService) GetTotalCommentsCount(ctx context.Context) (*dto.GetTotalCommentsCountResp, error) {
	count, err := s.CommentRepo.CountAll(ctx)
	if err != nil {
		log.CtxError(ctx, "Service GetTotalCommentCount failed: %v", err)
		return nil, errorx.ErrGetCountFailed
	}
	resp := &dto.GetTotalCommentsCountResp{
		Resp:  dto.Success(),
		Count: count,
	}

	return resp, nil
}

func (s *CommentService) GetMyComments(ctx context.Context, req *dto.GetMyCommentsReq) (*dto.GetMyCommentsResp, error) {
	// 获得用户id
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}
	// 构建查询参数
	param := &dto.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	// 查找数据库获得comments
	comments, total, err := s.CommentRepo.FindManyByUserID(ctx, param, userID)
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
	resp := &dto.GetMyCommentsResp{
		Resp:     dto.Success(),
		Total:    total,
		Comments: myCommentVOs,
	}
	return resp, nil
}

func (s *CommentService) GetCourseComments(ctx context.Context, req *dto.GetCourseCommentsReq) (*dto.GetCourseCommentsResp, error) {
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.ErrGetUserIDFailed
	}

	courseID := req.ID
	param := &dto.PageParam{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := s.CommentRepo.FindManyByCourseID(ctx, param, courseID)
	if err != nil {
		log.CtxError(ctx, "FindManyByUserID failed for userID=%s: %v", courseID, err)
		return nil, errorx.ErrFindFailed
	}

	vos, err := s.CommentDto.ToCommentVOList(ctx, comments, userID)
	if err != nil {
		log.CtxError(ctx, "ToCommentVOList failed for course: ", courseID, err)
		return nil, errorx.ErrCommentDB2VO
	}

	resp := &dto.GetCourseCommentsResp{
		Resp:     dto.Success(),
		Total:    total,
		Comments: vos,
	}

	return resp, nil
}
