package comment

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	prefixKeyCacheKey = "cache:comment"
	CollectionName    = "comment"
)

type IMongoMapper interface {
	Insert(ctx context.Context, c *Comment) error
}

type MongoMapper struct {
	conn *monc.Model
}

func NewMongoMapper(cfg *config.Config) *MongoMapper {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoMapper{conn: conn}
}

func (m *MongoMapper) Insert(ctx context.Context, c *Comment) error {
	if c.ID.IsZero() {
		c.ID = primitive.NewObjectID()
		c.CreatedAt = time.Now()
		c.UpdatedAt = time.Now()
	}
	_, err := m.conn.InsertOneNoCache(ctx, c)
	return err
}
