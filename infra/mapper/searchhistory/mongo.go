package searchhistory

import (
	"context"
	"errors"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const (
	prefixKeyCacheKey = "cache:searchhistory"
	CollectionName    = "searchhistory"
)

type ISearchHistory interface {
	FindByUserID(ctx context.Context, userID string) ([]*SearchHistory, error)
	Insert(ctx context.Context, h *SearchHistory) error
	DeleteByUserIDAndQuery(ctx context.Context, userID string, query string) error
	CountByUserID(ctx context.Context, userID string) (int64, error)
	DeleteOldestByUserID(ctx context.Context, userID string) error
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

	ops := options.Find()
	ops.SetSort(bson.D{{consts.CreatedAt, -1}}) // 降序, 最新的在最前面
	ops.SetLimit(consts.SearchHistoryLimit)

	filter := bson.M{consts.UserId: userID}

	if err := m.conn.Find(ctx, &histories, filter, ops); err != nil {
		return nil, err
	}
	return histories, nil
}

func (m *MongoMapper) Insert(ctx context.Context, h *SearchHistory) error {
	now := time.Now()
	if h.CreatedAt.IsZero() {
		h.CreatedAt = now
	}

	_, err := m.conn.InsertOneNoCache(ctx, h)
	return err
}

func (m *MongoMapper) DeleteByUserIDAndQuery(ctx context.Context, userID string, query string) error {
	_, err := m.conn.DeleteMany(ctx, bson.M{consts.UserId: userID, consts.Query: query})
	return err
}

func (m *MongoMapper) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return m.conn.CountDocuments(ctx, bson.M{consts.UserId: userID})
}

func (m *MongoMapper) DeleteOldestByUserID(ctx context.Context, userID string) error {
	var oldest SearchHistory
	ops := options.FindOneAndDelete()
	ops.SetSort(bson.D{{consts.CreatedAt, 1}})

	if err := m.conn.FindOneAndDelete(ctx, "", &oldest, bson.M{consts.UserId: userID}, ops); err != nil && !errors.Is(err, monc.ErrNotFound) {
		return err
	}
	return nil
}
