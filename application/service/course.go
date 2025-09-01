package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/google/wire"
)

type ICourseService interface {
	ListCourses(ctx context.Context, req *cmd.GetCoursesReq) (*cmd.GetCoursesResp, error)
	GetDeparts(ctx context.Context, req *cmd.GetCoursesDepartsReq) (*cmd.GetCoursesDepartsResp, error)
	GetCategories(ctx context.Context, req *cmd.GetCourseCategoriesReq) (*cmd.GetCourseCategoriesResp, error)
	GetCampuses(ctx context.Context, req *cmd.GetCourseCampusesReq) (*cmd.GetCourseCampusesResp, error)
}
type CourseService struct {
	CourseMapper *course.MongoMapper
	StaticData   *consts.StaticData
}

var CourseServiceSet = wire.NewSet(
	wire.Struct(new(CourseService), "*"),
	wire.Bind(new(ICourseService), new(*CourseService)),
)

func (s *CourseService) ListCourses(ctx context.Context, req *cmd.GetCoursesReq) (*cmd.GetCoursesResp, error) {

	courseListFromDB, total, err := s.CourseMapper.Find(ctx, req)

	if err != nil {
		return nil, err
	}

	// 将从数据库拿到的 course.Course 模型，转换为前端需要的 DTO 模型。
	var courseDTOList []cmd.CourseInList
	for _, dbCourse := range courseListFromDB {
		//先处理校区
		campusNames := make([]string, 0, len(dbCourse.Campuses))
		for _, dbCampus := range dbCourse.Campuses {
			campusName := s.StaticData.GetCampusNameByID(dbCampus)
			campusNames = append(campusNames, campusName)
		}

		apiCourse := cmd.CourseInList{
			ID:             dbCourse.ID,
			Name:           dbCourse.Name,
			Code:           dbCourse.Code,
			DepartmentName: s.StaticData.GetDepartmentNameByID(dbCourse.Department),
			CategoriesName: s.StaticData.GetCourseNameByID(dbCourse.Category),
			CampusesName:   campusNames,
			TeachersName:   dbCourse.TeacherIDs,
			// ... 其他需要返回给前端的字段
		}
		courseDTOList = append(courseDTOList, apiCourse)
	}

	response := &cmd.GetCoursesResp{
		Resp: cmd.Success(),
		PaginatedCourses: &cmd.PaginatedCourses{
			List:  courseDTOList,
			Total: total,
			PageParam: &cmd.PageParam{
				Page:     req.Page,
				PageSize: req.PageSize,
			},
		},
	}

	return response, nil
}

func (s *CourseService) GetDeparts(ctx context.Context, req *cmd.GetCoursesDepartsReq) (*cmd.GetCoursesDepartsResp, error) {

	departsIDs, err := s.CourseMapper.GetDeparts(ctx, req)
	if err != nil {
		return nil, err
	}

	departs := make([]string, 0, len(departsIDs))
	for _, dbDepart := range departsIDs {
		departs = append(departs, s.StaticData.GetDepartmentNameByID(dbDepart))
	}

	response := &cmd.GetCoursesDepartsResp{
		Resp:    cmd.Success(),
		Departs: departs,
	}

	return response, nil
}

func (s *CourseService) GetCategories(ctx context.Context, req *cmd.GetCourseCategoriesReq) (*cmd.GetCourseCategoriesResp, error) {

	categoriesIDs, err := s.CourseMapper.GetCategories(ctx, req)
	if err != nil {
		return nil, err
	}

	categories := make([]string, 0, len(categoriesIDs))
	for _, dbCategory := range categoriesIDs {
		categories = append(categories, s.StaticData.GetCourseNameByID(dbCategory))
	}

	response := &cmd.GetCourseCategoriesResp{
		Resp:       cmd.Success(),
		Categories: categories,
	}
	return response, nil
}

func (s *CourseService) GetCampuses(ctx context.Context, req *cmd.GetCourseCampusesReq) (*cmd.GetCourseCampusesResp, error) {

	campusesIDs, err := s.CourseMapper.GetCampuses(ctx, req)
	if err != nil {
		return nil, err
	}

	campuses := make([]string, 0, len(campusesIDs))
	for _, dbCampus := range campusesIDs {
		campuses = append(campuses, s.StaticData.GetCourseNameByID(dbCampus))
	}

	response := &cmd.GetCourseCampusesResp{
		Resp:     cmd.Success(),
		Campuses: campuses,
	}
	return response, nil
}
