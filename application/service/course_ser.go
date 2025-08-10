package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/repository"
)

type CourseService interface {
	ListCourses(ctx context.Context, query dto.CourseQuery) (*dto.PaginatedCoursesResponse, error)
}
type courseServiceImpl struct {
	courseRepo repository.CourseRepository
}

// NewCourseService 是 courseServiceImpl 的构造函数
func NewCourseService(courseRepo repository.CourseRepository) CourseService {
	return &courseServiceImpl{courseRepo: courseRepo}
}

// ListCourses 实现了核心的业务逻辑
func (s *courseServiceImpl) ListCourses(ctx context.Context, query dto.CourseQuery) (*dto.PaginatedCoursesResponse, error) {

	courseListFromDB, total, err := s.courseRepo.Find(ctx, query)
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

	// 3. 【组装】将处理好的数据，组装成最终的、完整的响应结构体
	// Controller 不应该关心分页的 total, page 这些细节，它只管从 Service 拿到最终结果然后返回JSON
	response := &dto.PaginatedCoursesResponse{
		List:  courseDTOList,
		Total: total,
		Page:  query.Page,
		Size:  query.PageSize,
	}

	return response, nil
}
