package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

type ICourseService interface {
	GetOneCourse(ctx context.Context, courseID string) (*cmd.GetOneCourseResp, error)
	ListCourses(ctx context.Context, req *cmd.ListCoursesReq) (*cmd.ListCoursesResp, error)
	GetDepartments(ctx context.Context, req *cmd.GetCoursesDepartmentsReq) (*cmd.GetCoursesDepartmentsResp, error)
	GetCategories(ctx context.Context, req *cmd.GetCourseCategoriesReq) (*cmd.GetCourseCategoriesResp, error)
	GetCampuses(ctx context.Context, req *cmd.GetCourseCampusesReq) (*cmd.GetCourseCampusesResp, error)
}

type CourseService struct {
	CourseMapper  *course.MongoMapper
	CommentMapper *comment.MongoMapper
	StaticData    *consts.StaticData
}

var CourseServiceSet = wire.NewSet(
	wire.Struct(new(CourseService), "*"),
	wire.Bind(new(ICourseService), new(*CourseService)),
)

// GetOneCourse 精确搜索，返回课程的元信息CourseVO
func (s *CourseService) GetOneCourse(ctx context.Context, courseID string) (*cmd.GetOneCourseResp, error) {
	var dbCourse *course.Course
	var err error
	if dbCourse, err = s.CourseMapper.FindOneByID(ctx, courseID); err != nil {
		return nil, err
	}

	// Optimize 可以单独抽象出dto层处理数据转化，使用go routine改善性能
	// 获得相关课程link并转化为VO
	var linkVOs []*cmd.CourseInLinkVO
	if dbCourse.LinkedCourses != nil {
		linkVOs = make([]*cmd.CourseInLinkVO, len(dbCourse.LinkedCourses))
		for i, c := range dbCourse.LinkedCourses {
			linkVOs[i] = &cmd.CourseInLinkVO{
				ID:   c.ID,
				Name: c.Name,
			}
		}
	}

	// 获得课程前三多的tag
	// TODO Optimize 起go routine进行处理，改善性能
	tagCount, err := s.CommentMapper.CountCourseTag(ctx, dbCourse.ID)
	if err != nil {
		log.Error("CountCourseTag Failed, courseID: ", courseID, err)
		return nil, errorx.ErrCountCourseTagsFailed
	}

	// 获取校区列表
	var campus []string
	for _, c := range dbCourse.Campuses {
		campus = append(campus, s.StaticData.GetCampusNameByID(c))
	}
	// 返回响应
	courseVO := &cmd.CourseVO{
		ID:         dbCourse.ID,
		Name:       dbCourse.Name,
		Code:       dbCourse.Code,
		Category:   s.StaticData.GetCategoryNameByID(dbCourse.Category),
		Campuses:   campus,
		Department: s.StaticData.GetDepartmentNameByID(dbCourse.Department),
		Link:       linkVOs,
		Teachers:   dbCourse.TeacherIDs,
		TagCount:   tagCount,
	}

	return &cmd.GetOneCourseResp{
		Resp:   cmd.Success(),
		Course: courseVO,
	}, nil
}

func (s *CourseService) ListCourses(ctx context.Context, req *cmd.ListCoursesReq) (*cmd.ListCoursesResp, error) {

	courseListFromDB, total, err := s.CourseMapper.FindMany(ctx, req.Keyword, req.PageParam)

	if err != nil {
		return nil, err
	}

	// 将从数据库拿到的 course.Course 模型，转换为前端需要的 DTO 模型。
	var courseDTOList []*cmd.CourseVO
	for _, dbCourse := range courseListFromDB {
		//先处理校区
		campusNames := make([]string, 0, len(dbCourse.Campuses))
		for _, dbCampus := range dbCourse.Campuses {
			campusName := s.StaticData.GetCampusNameByID(dbCampus)
			campusNames = append(campusNames, campusName)
		}

		apiCourse := &cmd.CourseVO{
			ID:         dbCourse.ID,
			Name:       dbCourse.Name,
			Code:       dbCourse.Code,
			Department: s.StaticData.GetDepartmentNameByID(dbCourse.Department),
			Category:   s.StaticData.GetCategoryNameByID(dbCourse.Category),
			Campuses:   campusNames,
			Teachers:   dbCourse.TeacherIDs,
			// ... 其他需要返回给前端的字段
		}
		courseDTOList = append(courseDTOList, apiCourse)
	}

	response := &cmd.ListCoursesResp{
		Resp: cmd.Success(),
		PaginatedCourses: &cmd.PaginatedCourses{
			Courses: courseDTOList,
			Total:   total,
			PageParam: &cmd.PageParam{
				Page:     req.Page,
				PageSize: req.PageSize,
			},
		},
	}

	return response, nil
}

func (s *CourseService) GetDepartments(ctx context.Context, req *cmd.GetCoursesDepartmentsReq) (*cmd.GetCoursesDepartmentsResp, error) {

	departsIDs, err := s.CourseMapper.GetDepartments(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	departs := make([]string, 0, len(departsIDs))
	for _, dbDepart := range departsIDs {
		departs = append(departs, s.StaticData.GetDepartmentNameByID(dbDepart))
	}

	response := &cmd.GetCoursesDepartmentsResp{
		Resp:        cmd.Success(),
		Departments: departs,
	}

	return response, nil
}

func (s *CourseService) GetCategories(ctx context.Context, req *cmd.GetCourseCategoriesReq) (*cmd.GetCourseCategoriesResp, error) {

	categoriesIDs, err := s.CourseMapper.GetCategories(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	categories := make([]string, 0, len(categoriesIDs))
	for _, dbCategory := range categoriesIDs {
		categories = append(categories, s.StaticData.GetCategoryNameByID(dbCategory))
	}

	response := &cmd.GetCourseCategoriesResp{
		Resp:       cmd.Success(),
		Categories: categories,
	}
	return response, nil
}

func (s *CourseService) GetCampuses(ctx context.Context, req *cmd.GetCourseCampusesReq) (*cmd.GetCourseCampusesResp, error) {

	campusesIDs, err := s.CourseMapper.GetCampuses(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	campuses := make([]string, 0, len(campusesIDs))
	for _, dbCampus := range campusesIDs {
		campuses = append(campuses, s.StaticData.GetCategoryNameByID(dbCampus))
	}

	response := &cmd.GetCourseCampusesResp{
		Resp:     cmd.Success(),
		Campuses: campuses,
	}
	return response, nil
}
