package repository

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
)

// CourseRepository 定义了课程数据仓库的接口
type CourseRepository interface {
	// Find 方法接收一个查询对象，返回课程列表、总记录数和错误
	Find(ctx context.Context, query dto.CourseQuery) ([]course.Course, int64, error)
}
