package course_mapper

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/repository" // 引入 Repository 接口
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo" // 引入 mongo driver
	"go.mongodb.org/mongo-driver/mongo/options"
)

// courseMapper 结构体，负责与数据库交互。它依赖一个 mongo.Database 连接实例。
type courseMapper struct {
	db *mongo.Database
}

var _ repository.CourseRepository = (*courseMapper)(nil)

// courseMapper 的构造函数，供依赖注入使用
func NewCourseMapper(db *mongo.Database) repository.CourseRepository {
	return &courseMapper{db: db}
}

// Find 方法的具体实现
func (m *courseMapper) Find(ctx context.Context, query dto.CourseQuery) ([]course.Course, int64, error) {
	//构建查询过滤器 (Filter)
	filter := bson.M{}
	if query.Keyword != "" {
		regex := bson.M{"$regex": primitive.Regex{Pattern: query.Keyword, Options: "i"}}
		filter["$or"] = []bson.M{
			{"name": regex},
			{"code": regex},
		}
	}

	//计算总数
	collection := m.db.Collection("courses")
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []course.Course{}, 0, nil
	}

	//构建分页和排序选项
	findOptions := options.Find()
	findOptions.SetSkip(int64((query.Page - 1) * query.PageSize))
	findOptions.SetLimit(int64(query.PageSize))
	findOptions.SetSort(bson.D{{"createdAt", -1}})

	//执行查询
	var courses []course.Course
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	//解码结果
	if err = cursor.All(ctx, &courses); err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}
