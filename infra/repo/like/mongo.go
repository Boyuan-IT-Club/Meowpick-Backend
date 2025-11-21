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

var _ IMongoRepo = (*MongoRepo)(nil)

const (
	CacheKeyPrefix = "meowpick:like:"
	CollectionName = "like"
)

type IMongoRepo interface {
	// ToggleLike 翻转点赞状态，返回完成后目标的点赞状态
	ToggleLike(ctx context.Context, userID, targetID string, targetType int32) (bool, error)
	// GetLikeStatus 获取一个用户对一个目标的当前点赞状态（是/否点赞）
	GetLikeStatus(ctx context.Context, userID, targetID string, targetType int32) (bool, error)
	// GetLikeCount 获得目标的总点赞数
	GetLikeCount(ctx context.Context, targetID string, targetType int32) (int64, error)
	// GetBatchLikeStatus 批量获取一个用户对多个目标的点赞状态，返回目标id->bool映射
	GetBatchLikeStatus(ctx context.Context, userID string, targetID []string, targetType int32) (map[string]bool, error)
	// GetBatchLikeCount 批量获取多个目标的点赞数
	GetBatchLikeCount(ctx context.Context, userID string, targetID []string, targetType int32) (map[string]int64, error)
}

type MongoRepo struct {
	conn      *monc.Model
	likeCache *cache.LikeCache
}

func NewMongoRepo(config *config.Config) *MongoRepo {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, CollectionName, config.Cache)
	return &MongoRepo{
		conn:      conn,
		likeCache: cache.NewLikeCache(config),
	}
}

func (m *MongoRepo) ToggleLike(ctx context.Context, userID, targetID string, targetType int32) (bool, error) {
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

func (m *MongoRepo) GetLikeStatus(ctx context.Context, userID, targetID string, targetType int32) (bool, error) {
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

func (m *MongoRepo) GetLikeCount(ctx context.Context, targetID string, targetType int32) (int64, error) {
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

func (m *MongoRepo) GetBatchLikeStatus(ctx context.Context, userID string, targetIDs []string, targetType int32) (map[string]bool, error) {
	if len(targetIDs) == 0 {
		return make(map[string]bool), nil
	}

	// 先查缓存
	cachedResults, missingIDs, err := m.likeCache.GetBatchLikeStatus(ctx, userID, targetIDs, targetType)
	if err != nil {
		log.Error("Batch get like status from cache error", err)
		// 缓存失败，直接查数据库
	}
	// 提前创建待返回的结果
	result := cachedResults
	if result == nil {
		result = make(map[string]bool)
	}

	// 如果有未命中的ID，查询数据库
	if len(missingIDs) > 0 {
		// 使用聚合查询批量获取点赞状态
		pipeline := []bson.M{
			{
				"$match": bson.M{
					consts.UserId:   userID,
					consts.TargetId: bson.M{"$in": missingIDs},
					consts.Active:   bson.M{"$ne": false},
				},
			},
			{
				"$group": bson.M{
					"_id": "$" + consts.TargetId,
				},
			},
		}

		// 临时结果结构体
		type aggResult struct {
			ID string `bson:"_id"`
		}
		var results []aggResult

		// 执行聚合查询，结果填充到 results
		err = m.conn.Aggregate(ctx, &results, pipeline)
		if err != nil {
			log.CtxError(ctx, "Aggregate like status error", err)
			return nil, errorx.ErrFindFailed
		}

		// 收集已点赞的targetID
		likedSet := make(map[string]bool)
		for _, doc := range results {
			likedSet[doc.ID] = true
		}

		// 批量设置缓存
		for _, targetID := range missingIDs {
			liked := likedSet[targetID] // 如果不存在则为false
			result[targetID] = liked

			// 异步设置缓存，避免阻塞
			go func(tID string, status bool) {
				_ = m.likeCache.SetLikeStatus(ctx, userID, tID, status, 10*time.Minute)
			}(targetID, liked)
		}
	}

	return result, nil
}

func (m *MongoRepo) GetBatchLikeCount(ctx context.Context, userID string, targetIDs []string, targetType int32) (map[string]int64, error) {
	if len(targetIDs) == 0 {
		return make(map[string]int64), nil
	}

	// 先查缓存
	cachedResults, missingIDs, err := m.likeCache.GetBatchLikeCount(ctx, targetIDs, targetType)
	if err != nil {
		log.Error("Batch get like count from cache error", err)
		// 缓存失败，直接查数据库
		missingIDs = targetIDs
		cachedResults = make(map[string]int64)
	}

	result := cachedResults
	if result == nil {
		result = make(map[string]int64)
	}

	// 如果有未命中的ID，查询数据库
	if len(missingIDs) > 0 {
		// 使用聚合查询批量获取点赞数
		pipeline := []bson.M{
			{
				"$match": bson.M{
					consts.TargetId: bson.M{"$in": missingIDs},
					consts.Active:   bson.M{"$ne": false},
				},
			},
			{
				"$group": bson.M{
					"_id":   "$" + consts.TargetId,
					"count": bson.M{"$sum": 1},
				},
			},
		}

		// 临时结果结构体
		type aggResult struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		var results []aggResult

		// 执行聚合查询，结果填充到 results
		err = m.conn.Aggregate(ctx, &results, pipeline)
		if err != nil {
			log.CtxError(ctx, "Aggregate like count error", err)
			return nil, errorx.ErrCountFailed
		}

		// 收集点赞数结果
		for _, doc := range results {
			result[doc.ID] = doc.Count
		}

		// 为没有点赞记录的targetID设置count为0
		for _, targetID := range missingIDs {
			if _, exists := result[targetID]; !exists {
				result[targetID] = 0
			}
		}

		// 批量设置缓存
		for _, targetID := range missingIDs {
			count := result[targetID]
			// 异步设置缓存，避免阻塞
			go func(tID string, cnt int64) {
				_ = m.likeCache.SetLikeCount(ctx, tID, cnt, 10*time.Minute)
			}(targetID, count)
		}
	}

	return result, nil
}
