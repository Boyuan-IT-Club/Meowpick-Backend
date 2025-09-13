package teacher

import (
	"context"
	"errors"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollectionName  = "courses"
	CollectionName2 = "teachers"
)

type IMongoMapper interface {
	FindCoursesByTeacherID(ctx context.Context, req *cmd.GetTeachersReq) ([]*course.Course, int64, error)
	GetTeacherSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) ([]*Teacher, error)
}

type MongoMapper struct {
	conn  *monc.Model
	conn2 *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	conn2 := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName2, cfg.Cache)
	return &MongoMapper{conn: conn, conn2: conn2}
}

// FindCoursesByTeacherID 根据教师ID查询其教授的所有课程
func (m *MongoMapper) FindCoursesByTeacherID(ctx context.Context, req *cmd.GetTeachersReq) ([]*course.Course, int64, error) {
	if req.TeacherID == "" {
		return nil, 0, errors.New("TeacherID is required")
	}

	var courses []*course.Course

	// 在 MongoDB 中，对数组字段进行简单的相等查询，会自动查找数组中包含该元素的文档
	filter := bson.M{consts.TeacherIds: req.TeacherID} //TODO

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*course.Course{}, 0, nil
	}

	findOptions := util.FindPageOption(req).SetSort(util.DSort(consts.CreatedAt, -1))

	err = m.conn.Find(ctx, &courses, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

func (m *MongoMapper) GetTeacherSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) ([]*Teacher, error) {
	var teachers []*Teacher
	filter := bson.M{"name": bson.M{"$regex": primitive.Regex{Pattern: req.Keyword, Options: "i"}}}
	findOptions := options.Find().SetLimit(10).SetProjection(bson.M{"name": 1})

	err := m.conn2.Find(ctx, &teachers, filter, findOptions)
	if err != nil {
		return nil, err
	}
	return teachers, nil
}
