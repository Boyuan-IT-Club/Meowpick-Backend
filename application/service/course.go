package service

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

var _ ICourseService = (*CourseService)(nil)

type ICourseService interface {
	GetOneCourse(ctx context.Context, courseID string) (*dto.GetOneCourseResp, error)
	ListCourses(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error)
	GetDepartments(ctx context.Context, req *dto.GetCoursesDepartmentsReq) (*dto.GetCoursesDepartmentsResp, error)
	GetCategories(ctx context.Context, req *dto.GetCourseCategoriesReq) (*dto.GetCourseCategoriesResp, error)
	GetCampuses(ctx context.Context, req *dto.GetCourseCampusesReq) (*dto.GetCourseCampusesResp, error)
}

type CourseService struct {
	CourseRepo  *course.MongoRepo
	CommentRepo *comment.MongoRepo
	TeacherRepo *teacher.MongoRepo
	StaticData  *mapping.StaticData
	CourseDTO   *assembler.CourseDTO
}

var CourseServiceSet = wire.NewSet(
	wire.Struct(new(CourseService), "*"),
	wire.Bind(new(ICourseService), new(*CourseService)),
)

// GetOneCourse 精确搜索，返回课程的元信息CourseVO
func (s *CourseService) GetOneCourse(ctx context.Context, courseID string) (*dto.GetOneCourseResp, error) {

	dbCourse, err := s.CourseRepo.FindOneByID(ctx, courseID)
	if err != nil || dbCourse == nil { // 使用id搜索不应出现找不到的情况
		return nil, err
	}

	courseVO, err := s.CourseDTO.ToCourseVO(ctx, dbCourse)
	if err != nil {
		log.CtxError(ctx, "CourseDB To CourseVO error: %v", err)
		return nil, errorx.ErrCourseDB2VO
	}

	return &dto.GetOneCourseResp{
		Resp:   dto.Success(),
		Course: courseVO,
	}, nil
}

func (s *CourseService) ListCourses(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error) {
	// 获取符合条件的总课程数量
	total, err := s.CourseRepo.CountCourses(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}
	// 若搜不到任何课程，直接返回空响应
	if total == 0 {
		// TODO 正常搜也会搜索不到的场景是否需要log？
		return &dto.ListCoursesResp{
			Resp: dto.Success(), PaginatedCourses: &dto.PaginatedCourses{},
		}, errorx.ErrFindSuccessButNoResult
	}

	// 使用模糊匹配搜索课程
	dbCourses, err := s.CourseRepo.GetCourseSuggestions(ctx, req.Keyword, req.PageParam)
	if err != nil {
		return nil, err
	}

	// 将数据库课程列表转换为分页结果
	paginatedCourses, err := s.CourseDTO.ToPaginatedCourses(ctx, dbCourses, total, req.PageParam)
	if err != nil {
		log.CtxError(ctx, "CourseDB To CourseVO error: %v", err)
		return nil, errorx.ErrCourseDB2VO
	}

	// 返回响应
	response := &dto.ListCoursesResp{
		Resp:             dto.Success(),
		PaginatedCourses: paginatedCourses,
	}

	return response, nil
}

func (s *CourseService) GetDepartments(ctx context.Context, req *dto.GetCoursesDepartmentsReq) (*dto.GetCoursesDepartmentsResp, error) {

	departsIDs, err := s.CourseRepo.GetDepartments(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	departs := make([]string, 0, len(departsIDs))
	for _, dbDepart := range departsIDs {
		departs = append(departs, s.StaticData.GetDepartmentNameByID(dbDepart))
	}

	response := &dto.GetCoursesDepartmentsResp{
		Resp:        dto.Success(),
		Departments: departs,
	}

	return response, nil
}

func (s *CourseService) GetCategories(ctx context.Context, req *dto.GetCourseCategoriesReq) (*dto.GetCourseCategoriesResp, error) {

	categoriesIDs, err := s.CourseRepo.GetCategories(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	categories := make([]string, 0, len(categoriesIDs))
	for _, dbCategory := range categoriesIDs {
		categories = append(categories, s.StaticData.GetCategoryNameByID(dbCategory))
	}

	response := &dto.GetCourseCategoriesResp{
		Resp:       dto.Success(),
		Categories: categories,
	}
	return response, nil
}

func (s *CourseService) GetCampuses(ctx context.Context, req *dto.GetCourseCampusesReq) (*dto.GetCourseCampusesResp, error) {

	campusesIDs, err := s.CourseRepo.GetCampuses(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	campuses := make([]string, 0, len(campusesIDs))
	for _, dbCampus := range campusesIDs {
		campuses = append(campuses, s.StaticData.GetCategoryNameByID(dbCampus))
	}

	response := &dto.GetCourseCampusesResp{
		Resp:     dto.Success(),
		Campuses: campuses,
	}
	return response, nil
}
