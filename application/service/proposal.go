package service

import (
	"context"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IProposalService = (*ProposalService)(nil)

type IProposalService interface {
	CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error)
}

type ProposalService struct {
	ProposalRepo *repo.ProposalRepo
	CourseRepo   *repo.CourseRepo
}

var ProposalServiceSet = wire.NewSet(
	wire.Struct(new(ProposalService), "*"),
	wire.Bind(new(IProposalService), new(*ProposalService)),
)

// CreateProposal 添加一个新的课程提案
func (s *ProposalService) CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 检查是否已经存在相同的提案（前端传回的和我已有的ProposalCourse）
	existingProposal, err1 := s.ProposalRepo.IsCourseInExistingProposals(ctx, req.Course)
	if err1 != nil {
		return nil, errorx.WrapByCode(err1, errno.ErrProposalCourseFindInProposalFailed,
			errorx.KV("operation", "IsCourseInExistingProposals"),
			errorx.KV("title", req.Title),            // 提案标题
			errorx.KV("userID", userId),              // 用户ID
			errorx.KV("courseName", req.Course.Name), // 课程名
			errorx.KV("courseCode", req.Course.Code))
	}
	//检查是否已经存在相同的课程（前端传回的的和我已有的course）
	existingCourse, err2 := s.CourseRepo.IsCourseInExistingCourses(ctx, req.Course)
	if err2 != nil {
		return nil, errorx.WrapByCode(err2, errno.ErrProposalCourseFindInCoursesFailed,
			errorx.KV("operation", "IsCourseInExistingCourses"),
			errorx.KV("title", req.Title))
	}

	if existingProposal == true {
		return nil, errorx.New(errno.ErrProposalCourseFoundInProposals,
			errorx.KV("title", req.Title))
	}
	if existingCourse == true {
		return nil, errorx.New(errno.ErrProposalCourseFoundInCourses,
			errorx.KV("title", req.Title))
	}
	campuses := []int32{}
	for _, campus := range req.Course.Campuses {
		campuses = append(campuses, mapping.Data.GetCampusIDByName(campus))
	}

	// 4. 构造课程信息
	course := model.Course{
		Name:       req.Course.Name,
		Code:       req.Course.Code,
		TeacherIDs: make([]string, 0),
		Department: mapping.Data.GetDepartmentIDByName(req.Course.Department),
		Category:   mapping.Data.GetCategoryIDByName(req.Course.Category),
		Campuses:   campuses,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 5. 创建提案对象
	status := mapping.Data.GetProposalStatusIDByName(req.Status)
	proposal := &model.Proposal{
		ID:        primitive.NewObjectID().Hex(),
		UserID:    userId,
		Title:     req.Title,
		Content:   req.Content,
		Deleted:   false,
		Course:    course,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 6. 保存提案到数据库
	if err := s.ProposalRepo.Insert(ctx, proposal); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrProposalCreateFailed,
			errorx.KV("title", req.Title),
			errorx.KV("userID", userId))
	}
	return &dto.CreateProposalResp{
		Resp:       dto.Success(),
		ProposalID: proposal.ID,
	}, nil
}
