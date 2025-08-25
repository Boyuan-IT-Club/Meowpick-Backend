package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
)

type ICourseService interface {
	ListCourses(ctx context.Context, query cmd.CourseQueryCmd) (*cmd.PaginatedCoursesResp, error)
}
type CourseService struct {
	courseMapper course.IMongoMapper
}

func NewCourseService(courseMapper course.IMongoMapper) *CourseService {
	return &CourseService{courseMapper: courseMapper}
}

func (s *CourseService) ListCourses(ctx context.Context, query cmd.CourseQueryCmd) (*cmd.PaginatedCoursesResp, error) {

	courseListFromDB, total, err := s.courseMapper.Find(ctx, query)

	if err != nil {
		return nil, err
	}

	// 将从数据库拿到的 course.Course 模型，转换为前端需要的 DTO 模型。
	var courseDTOList []cmd.CourseInList
	for _, dbCourse := range courseListFromDB {
		apiCourse := cmd.CourseInList{
			ID:         dbCourse.ID.Hex(), // 在这里把 ObjectID 转换为了字符串
			Name:       dbCourse.Name,
			Code:       dbCourse.Code,
			Department: dbCourse.Department,
			Categories: dbCourse.Categories,
			Campuses:   dbCourse.Campuses,
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
