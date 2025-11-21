package service

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/user"
	"github.com/google/wire"
)

var _ ITeacherService = (*TeacherService)(nil)

type ITeacherService interface {
	AddNewTeacher(ctx context.Context, req *dto.AddNewTeacherReq) (*dto.AddNewTeacherResp, error)
	ListCoursesByTeacher(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error)
}

type TeacherService struct {
	CourseMapper  *course.MongoRepo
	StaticData    *consts.StaticData
	CommentMapper *comment.MongoRepo
	UserMapper    *user.MongoRepo
	TeacherMapper *teacher.MongoRepo
	CourseDTO     *assembler.CourseDTO
	TeacherDTO    *assembler.TeacherDTO
}

var TeacherServiceSet = wire.NewSet(
	wire.Struct(new(TeacherService), "*"),
	wire.Bind(new(ITeacherService), new(*TeacherService)),
)

func (s *TeacherService) AddNewTeacher(ctx context.Context, req *dto.AddNewTeacherReq) (*dto.AddNewTeacherResp, error) {
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
	teacherVO := &dto.TeacherVO{
		Name:       req.Name,
		Title:      req.Title,
		Department: req.Department,
	}
	dbTeacher, err := s.TeacherDTO.ToTeacher(ctx, teacherVO)
	if err != nil {
		log.Error("TeacherVO To dbTeacher err:", teacherVO, err)
	}
	// 防重
	existingTeacher, err := s.TeacherMapper.FindOneTeacherByID(ctx, dbTeacher.ID)
	if err != nil && existingTeacher != nil {
		return nil, errorx.ErrTeacherDuplicate
	}

	// 增加教师
	teacherId, err := s.TeacherMapper.AddNewTeacher(ctx, dbTeacher)
	if err != nil {
		log.Error("Add New Teacher failed", err)
		return nil, err
	}
	teacherVO.ID = teacherId

	return &dto.AddNewTeacherResp{Resp: dto.Success(), TeacherVO: teacherVO}, nil
}

func (s *TeacherService) ListCoursesByTeacher(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error) {
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
		return &dto.ListCoursesResp{}, errorx.ErrFindSuccessButNoResult
	}

	paginatedCourses, err := s.CourseDTO.ToPaginatedCourses(ctx, courseListFromDB, total, req.PageParam)
	if err != nil {
		log.Error("ToPaginatedCourses failed", err)
		return nil, errorx.ErrCourseDB2VO
	}

	return &dto.ListCoursesResp{
		Resp:             dto.Success(),
		PaginatedCourses: paginatedCourses,
	}, nil
}
