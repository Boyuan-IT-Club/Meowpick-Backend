package comment

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
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
	FindManyByUserID(ctx context.Context, req *cmd.GetMyCommentsReq, userID string) ([]*Comment, int64, error)
	FindManyByCourseID(ctx context.Context, req *cmd.GetCourseCommentsReq, courseID string) ([]*Comment, int64, error)
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

func (m *MongoMapper) FindManyByUserID(ctx context.Context, req *cmd.GetMyCommentsReq, userID string) ([]*Comment, int64, error) {
	var comments []*Comment
	filter := bson.M{consts.UserId: userID, consts.Deleted: bson.M{"$ne": true}}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	ops := util.FindPageOption(req).SetSort(util.DSort(consts.CreatedAt, -1))

	if err = m.conn.Find(ctx, &comments, filter, ops); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (m *MongoMapper) FindManyByCourseID(ctx context.Context, req *cmd.GetCourseCommentsReq, courseID string) ([]*Comment, int64, error) {
	var comments []*Comment
	filter := bson.M{consts.CourseID: courseID, consts.Deleted: bson.M{"$ne": true}}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	ops := util.FindPageOption(req).SetSort(util.DSort(consts.CreatedAt, -1))

	if err := m.conn.Find(ctx, &comments, filter, ops); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}