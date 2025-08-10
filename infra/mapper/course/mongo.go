package course

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollectionName = "courses"
)

type IMongoMapper interface {
	Find(ctx context.Context, query dto.CourseQueryCmd) ([]Course, int64, error)
}

type MongoMapper struct {
	conn *monc.Model
}

var _ IMongoMapper = (*MongoMapper)(nil)

func NewCourseMapper(cfg *config.Config) IMongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) Find(ctx context.Context, query dto.CourseQueryCmd) ([]Course, int64, error) {
	//构建查询过滤器 (Filter)
	filter := bson.M{}
	if query.Keyword != "" {
		regex := bson.M{"$regex": primitive.Regex{Pattern: query.Keyword, Options: "i"}}
		filter["$or"] = []bson.M{
			{"name": regex},
			{"code": regex},
		}
	}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []Course{}, 0, nil
	}

	//构建分页和排序选项
	findOptions := options.Find()
	findOptions.SetSkip(int64((query.Page - 1) * query.PageSize))
	findOptions.SetLimit(int64(query.PageSize))
	findOptions.SetSort(bson.D{{"createdAt", -1}})

	var courses []Course
	err = m.conn.Find(ctx, &courses, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}
