package teacher

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	prefixKeyCacheKey = "cache:teacher"
	CollectionName    = "teacher"
)

type IMongoMapper interface {
	FindByKeyword(ctx context.Context, keyword string) ([]string, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) FindByKeyword(ctx context.Context, keyword string) ([]string, error) {
	var results []any
	// TODO: 分页和 Find
	// 模糊查找，不区分大小写
	results, err := m.conn.Distinct(ctx, "name", bson.M{"name": bson.M{"$regex": keyword, "$options": "i"}})

	names := make([]string, 0, len(results))
	for _, result := range results {
		if name, ok := result.(string); ok {
			names = append(names, name)
		}
	}

	if err != nil {
		return nil, err
	}
	return names, nil
}
