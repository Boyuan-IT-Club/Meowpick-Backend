package searchhistory

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	prefixKeyCacheKey = "cache:searchhistory"
	CollectionName    = "searchhistory"
)

type ISearchHistory interface {
	FindByUserID(ctx context.Context, userID string) ([]*SearchHistory, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) FindByUserID(ctx context.Context, userID string) ([]*SearchHistory, error) {
	var histories []*SearchHistory

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{consts.CreatedTime, -1}})
	findOptions.SetLimit(consts.SearchHistoryLimit)

	filter := bson.M{consts.UserId: userID}

	if err := m.conn.Find(ctx, &histories, filter, findOptions); err != nil {
		return nil, err
	}
	return histories, nil
}
