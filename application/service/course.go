package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/google/wire"
	"strconv"
)

type ICourseService interface {
	ListCourses(ctx context.Context, query cmd.CourseQueryCmd) (*cmd.PaginatedCoursesResp, error)
}
type CourseService struct {
	CourseMapper *course.MongoMapper
	StaticData   *consts.StaticData
}

var CourseServiceSet = wire.NewSet(
	wire.Struct(new(CourseService), "*"),
	wire.Bind(new(ICourseService), new(*CourseService)),
)

func (s *CourseService) ListCourses(ctx context.Context, query cmd.CourseQueryCmd) (*cmd.PaginatedCoursesResp, error) {

	courseListFromDB, total, err := s.CourseMapper.Find(ctx, query)

	if err != nil {
		return nil, err
	}

	// 将从数据库拿到的 course.Course 模型，转换为前端需要的 DTO 模型。
	var courseDTOList []cmd.CourseInList
	for _, dbCourse := range courseListFromDB {
		//先处理校区
		campusNames := make([]string, 0, len(dbCourse.Campuses))
		for _, dbCampus := range dbCourse.Campuses {
			campusName := s.StaticData.Campuses[strconv.FormatInt(int64(dbCampus), 10)]
			campusNames = append(campusNames, campusName)
		}

		apiCourse := cmd.CourseInList{
			ID:             dbCourse.ID,
			Name:           dbCourse.Name,
			Code:           dbCourse.Code,
			DepartmentName: s.StaticData.Departments[strconv.FormatInt(int64(dbCourse.Department), 10)],
			CategoriesName: s.StaticData.Courses[strconv.FormatInt(int64(dbCourse.Category), 10)],
			CampusesName:   campusNames,
			// ... 其他需要返回给前端的字段
		}
		courseDTOList = append(courseDTOList, apiCourse)
	}

	page := &cmd.PaginatedCourses{
		List:  courseDTOList,
		Total: total,
		Page:  query.Page,
		Size:  query.PageSize,
	}

	response := &cmd.PaginatedCoursesResp{
		Code: 0,
		Msg:  "",
		Page: page,
	}

	return response, nil
}
