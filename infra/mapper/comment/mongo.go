package comment

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

const (
	prefixKeyCacheKey = "cache:comment"
	CollectionName    = "comment"
)

type IMongoMapper interface {
	Insert(ctx context.Context, c *Comment) error
	CountAll(ctx context.Context) (int64, error)
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) Insert(ctx context.Context, c *Comment) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

	_, err := m.conn.InsertOneNoCache(ctx, c)
	return err
}

func (m *MongoMapper) CountAll(ctx context.Context) (int64, error) {
	filter := bson.M{consts.Deleted: bson.M{"$ne": true}}
	count, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}
