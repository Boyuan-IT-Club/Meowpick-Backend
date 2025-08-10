package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
)

type ICourseService interface {
	ListCourses(ctx context.Context, query dto.CourseQueryCmd) (*dto.PaginatedCoursesResp, error)
}
type CourseService struct {
	courseMapper course.IMongoMapper
}

// NewCourseService 是 CourseService 的构造函数
func NewCourseService(courseMapper course.IMongoMapper) ICourseService {
	return &CourseService{courseMapper: courseMapper}
}

// ListCourses 实现了核心的业务逻辑
func (s *CourseService) ListCourses(ctx context.Context, query dto.CourseQueryCmd) (*dto.PaginatedCoursesResp, error) {

	courseListFromDB, total, err := s.courseMapper.Find(ctx, query)
	if err != nil {
		return nil, err
	}

	// 将从数据库拿到的 course.Course 模型，转换为前端需要的 types.CourseForList DTO 模型。
	var courseDTOList []dto.CourseInList
	for _, dbCourse := range courseListFromDB {
		apiCourse := dto.CourseInList{
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

	response := &dto.PaginatedCoursesResp{
		List:  courseDTOList,
		Total: total,
		Page:  query.Page,
		Size:  query.PageSize,
	}

	return response, nil
}
