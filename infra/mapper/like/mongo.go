package like

import (
	"context"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CacheKeyPrefix = "like:"
	CollectionName = "like"
)

type IMongoMapper interface {
	// ToggleLike 翻转点赞状态，返回完成后目标的点赞状态
	ToggleLike(ctx context.Context, userID, targetID string, targetType int32) (bool, error)
	// GetLikeStatus 获取一个用户对一个目标的当前点赞状态（是/否点赞）
	GetLikeStatus(ctx context.Context, userID, targetID string, targetType int32) (bool, error)
	// GetLikeCount 获得目标的总点赞数
	GetLikeCount(ctx context.Context, targetID string, targetType int32) (int64, error)
}

type MongoMapper struct {
	conn      *monc.Model
	likeCache *cache.LikeCache
}

func NewMongoMapper(config *config.Config) *MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, CollectionName, config.Cache)
	return &MongoMapper{
		conn:      conn,
		likeCache: cache.NewLikeCache(config),
	}
}

func (m *MongoMapper) ToggleLike(ctx context.Context, userID, targetID string, targetType int32) (bool, error) {
	// 把存在且 active != false 的文档视为当前已点赞（包括 active 字段缺失的历史记录）
	filterActive := bson.M{
		consts.UserId:   userID,
		consts.TargetId: targetID,
		consts.Active:   bson.M{"$ne": false}, // 非false
		//"targetType": targetType,
	}

	cnt, err := m.conn.CountDocuments(ctx, filterActive)
	if err != nil {
		return false, errorx.ErrFindFailed
	}

	// 如果已有记录（包含缺失 active 字段的历史记录），则本次操作为取消（newActive=false）
	// 否则为点赞（newActive=true）
	newActive := cnt == 0

	update := bson.M{
		"$set": bson.M{
			consts.Active:    newActive,
			consts.UpdatedAt: time.Now(),
		},
		"$setOnInsert": bson.M{
			consts.CreatedAt: time.Now(),
			//"targetType": targetType,
		},
	}
	updateOptions := options.Update().SetUpsert(true)

	filter := bson.M{
		consts.UserId:   userID,
		consts.TargetId: targetID,
	}
	cacheKey := CacheKeyPrefix + userID + "-" + targetID
	if _, err := m.conn.UpdateOne(ctx, cacheKey, filter, update, updateOptions); err != nil {
		return false, errorx.ErrUpdateFailed
	}

	// 更新点赞状态缓存
	_ = m.likeCache.SetLikeStatus(ctx, userID, targetID, newActive, 10*time.Minute)

	// 更新点赞数缓存（如果缓存中存在的话）
	if newActive {
		// 点赞：缓存+1，如果缓存不存在则不处理（等下次查询时回填）
		if _, err = m.likeCache.IncrLikeCount(ctx, targetID, 1); err != nil {
			log.Error("Increase like count cache error", err)
		}
	} else {
		// 取消点赞：缓存-1
		if _, err = m.likeCache.IncrLikeCount(ctx, targetID, -1); err != nil {
			log.Error("Increase like count cache error", err)
		}
	}

	return newActive, nil
}

func (m *MongoMapper) GetLikeStatus(ctx context.Context, userID, targetID string, targetType int32) (bool, error) {
	// 缓存查询
	if liked, found := m.likeCache.GetLikeStatus(ctx, userID, targetID); found {
		return liked, nil
	}

	// 缓存未命中，走数据库查询
	filter := bson.M{
		consts.UserId:   userID,
		consts.TargetId: targetID,
		consts.Active:   bson.M{"$ne": false}, // 非 false 视为点赞（包含缺失字段）
		//"targetType": targetType,
	}

	cnt, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, errorx.ErrFindFailed
	}
	return cnt > 0, nil
}

func (m *MongoMapper) GetLikeCount(ctx context.Context, targetID string, targetType int32) (int64, error) {
	// 缓存查询
	if count, found := m.likeCache.GetLikeCount(ctx, targetID); found {
		return count, nil
	}

	// 缓存未命中，走数据库查询
	filter := bson.M{
		consts.TargetId: targetID,
		consts.Active:   bson.M{"$ne": false}, // 排除 active==false，包含缺失字段
		//"targetType": targetType,
	}

	count, err := m.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, errorx.ErrCountFailed
	}
	return count, nil
}
