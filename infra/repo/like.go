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
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ ILikeRepo = (*LikeRepo)(nil)

const (
	LikeCollectionName = "like"
)

type ILikeRepo interface {
	Toggle(ctx context.Context, userId, targetId string, targetType int32) (bool, error)
	IsLike(ctx context.Context, userId, targetId string, targetType int32) (bool, error)
	CountByTarget(ctx context.Context, targetId string, targetType int32) (int64, error)

	GetLikesByUserIDAndTargets(ctx context.Context, userId string, targetIds []string, targetType int32) (map[string]bool, error)
	CountByTargets(ctx context.Context, targetIds []string, targetType int32) (map[string]int64, error)
}

type LikeRepo struct {
	conn  *monc.Model
	cache *cache.LikeCache
}

func NewLikeRepo(config *config.Config) *LikeRepo {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, LikeCollectionName, config.Cache)
	return &LikeRepo{conn: conn}
}

// Toggle 翻转点赞状态
func (r *LikeRepo) Toggle(ctx context.Context, userId, targetId string, targetType int32) (bool, error) {
	now := time.Now()
	pipeline := mongo.Pipeline{
		{{"$set", bson.M{
			consts.ID: bson.M{"$ifNull": bson.A{"$" + consts.ID, primitive.NewObjectID().Hex()}},

			consts.UserID:   bson.M{"$ifNull": bson.A{"$" + consts.UserID, userId}},
			consts.TargetID: bson.M{"$ifNull": bson.A{"$" + consts.TargetID, targetId}},
			// consts.TargetType: bson.M{"$ifNull": bson.A{"$" + consts.TargetType, targetType}},

			consts.CreatedAt: bson.M{"$ifNull": bson.A{"$" + consts.CreatedAt, now}},
			consts.UpdatedAt: now,

			consts.Active: bson.M{"$cond": bson.A{
				bson.M{"$not": bson.M{"$ifNull": bson.A{"$" + consts.ID, nil}}},
				true,
				bson.M{"$not": "$active"},
			}},
		}}},
	}
	var like struct {
		Active bool `bson:"active"`
	}
	err := r.conn.FindOneAndUpdateNoCache(ctx,
		&like,
		bson.M{consts.UserID: userId, consts.TargetID: targetId},
		pipeline,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)
	return like.Active, err
}

// IsLike 获取一个用户对一个目标的当前点赞状态（是/否点赞）
func (r *LikeRepo) IsLike(ctx context.Context, userId, targetId string, targetType int32) (bool, error) {
	cnt, err := r.conn.CountDocuments(ctx, bson.M{
		consts.UserID:   userId,
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false},
		//consts.TargetType: targetType,
	})
	return cnt > 0, err
}

// CountByTarget 获得目标的总点赞数
func (r *LikeRepo) CountByTarget(ctx context.Context, targetId string, targetType int32) (int64, error) {
	return r.conn.CountDocuments(ctx, bson.M{
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false},
		//consts.TargetType: targetType,
	})
}

// GetLikesByUserIDAndTargets 批量获取一个用户对多个目标的点赞状态，返回目标id->bool映射
func (r *LikeRepo) GetLikesByUserIDAndTargets(ctx context.Context, userId string, targetIds []string, targetType int32) (map[string]bool, error) {
	var likes []struct {
		TargetID string `bson:"targetId"`
	}
	if err := r.conn.Find(ctx, &likes, bson.M{
		consts.UserID:   userId,
		consts.TargetID: bson.M{"$in": targetIds},
		consts.Active:   bson.M{"$ne": false},
		//consts.TargetType: targetType,
	}); err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(targetIds))
	for _, like := range likes {
		result[like.TargetID] = true
	}
	return result, nil
}

// CountByTargets 批量获取多个目标的点赞数，返回目标id->count映射
func (r *LikeRepo) CountByTargets(ctx context.Context, targetIds []string, targetType int32) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{consts.TargetID, bson.D{{"$in", targetIds}}},
			{consts.Active, bson.D{{"$ne", false}}},
			//{consts.TargetType, targetType},
		}}},
		{{"$group", bson.D{
			{"_id", "$targetId"},
			{"count", bson.D{{"$sum", 1}}},
		}}},
	}
	var likes []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := r.conn.Aggregate(ctx, &likes, pipeline); err != nil {
		return nil, err
	}
	results := make(map[string]int64, len(targetIds))
	for _, like := range likes {
		results[like.ID] = like.Count
	}
	return results, nil
}
