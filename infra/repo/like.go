// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repo

import (
	"context"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ ILikeRepo = (*LikeRepo)(nil)

const (
	LikeCacheKeyPrefix = "meowpick:like:"
	LikeCollectionName = "like"
)

type ILikeRepo interface {
	ToggleLike(ctx context.Context, userID, targetId string, targetType int32) (bool, error)
	GetLikeStatus(ctx context.Context, userID, targetId string, targetType int32) (bool, error)
	GetLikeCount(ctx context.Context, targetId string, targetType int32) (int64, error)
	GetBatchLikeStatus(ctx context.Context, userID string, targetIds []string, targetType int32) (map[string]bool, error)
	GetBatchLikeCount(ctx context.Context, userID string, targetIds []string, targetType int32) (map[string]int64, error)
}

type LikeRepo struct {
	conn  *monc.Model
	cache *cache.LikeCache
}

func NewLikeRepo(config *config.Config) *LikeRepo {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, LikeCollectionName, config.Cache)
	return &LikeRepo{
		conn:  conn,
		cache: cache.NewLikeCache(config),
	}
}

// ToggleLike 翻转点赞状态，返回完成后目标的点赞状态
func (r *LikeRepo) ToggleLike(ctx context.Context, userID, targetId string, targetType int32) (bool, error) {
	// 把存在且 active != false 的文档视为当前已点赞（包括 active 字段缺失的历史记录）
	filterActive := bson.M{
		consts.UserID:   userID,
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false}, // 非false
		//"targetType": targetType,
	}

	cnt, err := r.conn.CountDocuments(ctx, filterActive)
	if err != nil {
		return false, err
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
	ops := options.Update().SetUpsert(true)

	filter := bson.M{
		consts.UserID:   userID,
		consts.TargetID: targetId,
	}
	cacheKey := LikeCacheKeyPrefix + userID + "-" + targetId
	if _, err = r.conn.UpdateOne(ctx, cacheKey, filter, update, ops); err != nil {
		return false, err
	}

	// 更新点赞状态缓存
	_ = r.cache.SetLikeStatus(ctx, userID, targetId, newActive, 10*time.Minute)

	// 更新点赞数缓存（如果缓存中存在的话）
	if newActive {
		// 点赞：缓存+1，如果缓存不存在则不处理（等下次查询时回填）
		if _, err = r.cache.IncrLikeCount(ctx, targetId, 1); err != nil {
			logs.Errorf("Increase like count cache error: %v", err)
		}
	} else {
		// 取消点赞：缓存-1
		if _, err = r.cache.IncrLikeCount(ctx, targetId, -1); err != nil {
			logs.Errorf("Increase like count cache error: %v", err)
		}
	}

	return newActive, nil
}

// GetLikeStatus 获取一个用户对一个目标的当前点赞状态（是/否点赞）
func (r *LikeRepo) GetLikeStatus(ctx context.Context, userID, targetId string, targetType int32) (bool, error) {
	// 缓存查询
	if liked, found := r.cache.GetLikeStatus(ctx, userID, targetId); found {
		return liked, nil
	}

	// 缓存未命中，走数据库查询
	filter := bson.M{
		consts.UserID:   userID,
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false}, // 非 false 视为点赞（包含缺失字段）
		//"targetType": targetType,
	}

	cnt, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// GetLikeCount 获得目标的总点赞数
func (r *LikeRepo) GetLikeCount(ctx context.Context, targetId string, targetType int32) (int64, error) {
	// 缓存查询
	if count, found := r.cache.GetLikeCount(ctx, targetId); found {
		return count, nil
	}

	// 缓存未命中，走数据库查询
	filter := bson.M{
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false}, // 排除 active==false，包含缺失字段
		//"targetType": targetType,
	}

	count, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetBatchLikeStatus 批量获取一个用户对多个目标的点赞状态，返回目标id->bool映射
func (r *LikeRepo) GetBatchLikeStatus(ctx context.Context, userID string, targetIds []string, targetType int32) (map[string]bool, error) {
	if len(targetIds) == 0 {
		return make(map[string]bool), nil
	}

	// 先查缓存
	cachedResults, missingIDs, err := r.cache.GetBatchLikeStatus(ctx, userID, targetIds, targetType)
	if err != nil {
		logs.Errorf("Batch get like status from cache error: %v", err)
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
					consts.UserID:   userID,
					consts.TargetID: bson.M{"$in": missingIDs},
					consts.Active:   bson.M{"$ne": false},
				},
			},
			{
				"$group": bson.M{
					"_id": "$" + consts.TargetID,
				},
			},
		}

		// 临时结果结构体
		type aggResult struct {
			ID string `bson:"_id"`
		}
		var results []aggResult

		// 执行聚合查询，结果填充到 results
		err = r.conn.Aggregate(ctx, &results, pipeline)
		if err != nil {
			logs.Errorf("Aggregate like status error: %v", err)
			return nil, err
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
				_ = r.cache.SetLikeStatus(ctx, userID, tID, status, 10*time.Minute)
			}(targetID, liked)
		}
	}

	return result, nil
}

// GetBatchLikeCount 批量获取多个目标的点赞数
func (r *LikeRepo) GetBatchLikeCount(ctx context.Context, userID string, targetIds []string, targetType int32) (map[string]int64, error) {
	if len(targetIds) == 0 {
		return make(map[string]int64), nil
	}

	// 先查缓存
	cachedResults, missingIDs, err := r.cache.GetBatchLikeCount(ctx, targetIds, targetType)
	if err != nil {
		logs.Errorf("Batch get like count from cache error: %v", err)
		// 缓存失败，直接查数据库
		missingIDs = targetIds
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
					consts.TargetID: bson.M{"$in": missingIDs},
					consts.Active:   bson.M{"$ne": false},
				},
			},
			{
				"$group": bson.M{
					"_id":   "$" + consts.TargetID,
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
		err = r.conn.Aggregate(ctx, &results, pipeline)
		if err != nil {
			logs.Errorf("Aggregate like count error: %v", err)
			return nil, err
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
				_ = r.cache.SetLikeCount(ctx, tID, cnt, 10*time.Minute)
			}(targetID, count)
		}
	}

	return result, nil
}
