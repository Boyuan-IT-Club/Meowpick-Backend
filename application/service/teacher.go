package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/google/wire"
)

type ITeacherService interface {
	AddNewTeacher(ctx context.Context, req *cmd.AddNewTeacherReq) (*cmd.AddNewTeacherResp, error)
	ListCoursesByTeacher(ctx context.Context, req *cmd.ListCoursesReq) (*cmd.ListCoursesResp, error)
}

type TeacherService struct {
	CourseMapper  *course.MongoMapper
	StaticData    *consts.StaticData
	CommentMapper *comment.MongoMapper
	UserMapper    *user.MongoMapper
	TeacherMapper *teacher.MongoMapper
	CourseDTO     *dto.CourseDTO
}

func (s *TeacherService) AddNewTeacher(ctx context.Context, req *cmd.AddNewTeacherReq) (*cmd.AddNewTeacherResp, error) {
	// 鉴权
	userID, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userID == "" {
		log.Error("Get user Id failed")
		return nil, errorx.ErrTokenInvalid
	}
	if admin, _ := s.UserMapper.IsAdmin(ctx, userID); !admin {
		return nil, errorx.ErrUserNotAdmin
	}

	// 如果拥有管理员权限，继续向下执行添加教师的逻辑
	teacherVO := &cmd.TeacherVO{
		Name:       req.Name,
		Title:      req.Title,
		Department: req.Department,
	}
	// 防重
	existingTeacher, err := s.TeacherMapper.FindOneTeacherByVO(ctx, teacherVO)
	if err != nil && existingTeacher != nil {
		return nil, errorx.ErrTeacherDuplicate
	}
	// 增加教师
	teacherId, err := s.TeacherMapper.AddNewTeacher(ctx, teacherVO)
	if err != nil {
		log.Error("Add New Teacher failed", err)
		return nil, err
	}
	teacherVO.ID = teacherId

	return &cmd.AddNewTeacherResp{Resp: cmd.Success(), TeacherVO: teacherVO}, nil
}

var TeacherServiceSet = wire.NewSet(
	wire.Struct(new(TeacherService), "*"),
	wire.Bind(new(ITeacherService), new(*TeacherService)),
)

func (s *TeacherService) ListCoursesByTeacher(ctx context.Context, req *cmd.ListCoursesReq) (*cmd.ListCoursesResp, error) {
	teacherID, err := s.TeacherMapper.GetTeacherIDByName(ctx, req.Keyword)
	if err != nil {
		log.Error("GetTeacherIDByName failed for keyword: ", req.Keyword, err)
		return nil, err
	}

	courseListFromDB, total, err := s.CourseMapper.FindCoursesByTeacherID(ctx, teacherID, req.PageParam)
	if err != nil {
		log.Error("FindCoursesByTeacher failed for teacherID: ", teacherID, err)
		return nil, err
	}
	if total == 0 {
		return &cmd.ListCoursesResp{}, errorx.ErrFindSuccessButNoResult
	}

	paginatedCourses, err := s.CourseDTO.ToPaginatedCourses(ctx, courseListFromDB, total, req.PageParam)
	if err != nil {
		log.Error("ToPaginatedCourses failed", err)
		return nil, errorx.ErrCourseDB2VO
	}

	return &cmd.ListCoursesResp{
		Resp:             cmd.Success(),
		PaginatedCourses: paginatedCourses,
	}, nil
}
