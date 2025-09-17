package comment

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	prefixKeyCacheKey = "cache:comment"
	CollectionName    = "comment"
)

type IMongoMapper interface {
	Insert(ctx context.Context, c *Comment) error
	CountAll(ctx context.Context) (int64, error)
	FindManyByUserID(ctx context.Context, page, pageSize int64, userID string) ([]*Comment, int64, error)
	FindManyByCourseID(ctx context.Context, page, pageSize int64, courseID string) ([]*Comment, int64, error)
	CountCourseTag(ctx context.Context, courseID string) (map[string]int, error)
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

func (m *MongoMapper) FindManyByUserID(ctx context.Context, page, pageSize int64, userID string) ([]*Comment, int64, error) {
	var comments []*Comment
	filter := bson.M{consts.UserId: userID, consts.Deleted: bson.M{"$ne": true}}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	pageParam := &cmd.PageParam{Page: page, PageSize: pageSize}
	ops := util.FindPageOption(pageParam).SetSort(util.DSort(consts.CreatedAt, -1))

	if err = m.conn.Find(ctx, &comments, filter, ops); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (m *MongoMapper) FindManyByCourseID(ctx context.Context, page, pageSize int64, courseID string) ([]*Comment, int64, error) {
	var comments []*Comment
	filter := bson.M{consts.CourseId: courseID, consts.Deleted: bson.M{"$ne": true}}

	total, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	pageParam := &cmd.PageParam{Page: page, PageSize: pageSize}
	ops := util.FindPageOption(pageParam).SetSort(util.DSort(consts.CreatedAt, -1))

	if err := m.conn.Find(ctx, &comments, filter, ops); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (m *MongoMapper) CountCourseTag(ctx context.Context, courseID string) (map[string]int, error) {
	// 数据库聚合实现标签count 建议在/api/search接口封装CourseVO时起go routine获得CourseVO的tagCount字段
	// 构建管道
	pipeline := bson.A{
		// 阶段1：筛选符合条件的文档
		bson.M{"$match": bson.M{
			consts.CourseId: courseID,
			consts.Deleted:  bson.M{"$ne": true},
			"tags":          bson.M{"$exists": true, "$ne": nil},
		}},

		// 阶段2：展开tags数组
		bson.M{"$unwind": bson.M{
			"path":                       "$tags",
			"preserveNullAndEmptyArrays": false,
		}},

		// 阶段3：过滤掉空字符串的tag
		bson.M{"$match": bson.M{
			"tags": bson.M{"$ne": "", "$exists": true},
		}},

		// 阶段4：按tag分组并计数
		bson.M{"$group": bson.M{
			"_id":   "$tags",
			"count": bson.M{"$sum": 1}, // 这里sum使用的是int64还是int result map[string]xxx需要和sum的类型保持一致
		}},

		// 阶段5：按计数降序排序
		bson.M{"$sort": bson.M{"count": -1}},

		// 阶段6：限制返回结果数量
		bson.M{"$limit": 3},
	}

	// 使用monc的Aggregate方法执行聚合查询
	var results []struct {
		Tag   string `bson:"_id"`
		Count int    `bson:"count"` // 暂显式指定int类型，以Aggregate求sum时使用的类型为准(也可能为int32/int64)
	}

	if err := m.conn.Aggregate(ctx, &results, pipeline); err != nil {
		log.Error("Aggregate failed for courseID=%s: %v", courseID, err)
		return nil, err
	}

	// 转换为 map
	result := make(map[string]int)
	for _, item := range results {
		result[item.Tag] = item.Count
	}
	return result, nil
}
