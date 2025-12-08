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

package searchhistory

import (
	"context"
	"errors"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ IMongoRepo = (*MongoRepo)(nil)

const (
	CacheKeyPrefix = "meowpick:searchhistory:"
	CollectionName = "searchhistory"
)

type IMongoRepo interface {
	FindByUserID(ctx context.Context, userID string) ([]*SearchHistory, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	DeleteOldestByUserID(ctx context.Context, userID string) error
	UpsertByUserIDAndQuery(ctx context.Context, userID string, query string) error
}

type MongoRepo struct {
	conn *monc.Model
}

func NewMongoRepo(cfg *config.Config) *MongoRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CollectionName, cfg.Cache)
	return &MongoRepo{conn: conn}
}

func (m *MongoRepo) FindByUserID(ctx context.Context, userID string) ([]*SearchHistory, error) {
	var histories []*SearchHistory

	ops := options.Find()
	ops.SetSort(page.DSort(consts.CreatedAt, -1)) // 降序, 最新的在最前面
	ops.SetLimit(consts.SearchHistoryLimit)

	filter := bson.M{consts.UserId: userID}

	if err := m.conn.Find(ctx, &histories, filter, ops); err != nil {
		return nil, err
	}
	return histories, nil
}

func (m *MongoRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return m.conn.CountDocuments(ctx, bson.M{consts.UserId: userID})
}

func (m *MongoRepo) DeleteOldestByUserID(ctx context.Context, userID string) error {
	var oldest SearchHistory
	ops := options.FindOneAndDelete()
	ops.SetSort(page.DSort(consts.CreatedAt, 1))

	cacheKey := CacheKeyPrefix + userID
	if err := m.conn.FindOneAndDelete(ctx, cacheKey, &oldest, bson.M{consts.UserId: userID}, ops); err != nil && !errors.Is(err, monc.ErrNotFound) {
		return err
	}
	return nil
}

func (m *MongoRepo) UpsertByUserIDAndQuery(ctx context.Context, userID string, query string) error {
	filter := bson.M{
		consts.UserId: userID,
		consts.Query:  query,
	}

	// 2. 定义“更新操作”
	update := bson.M{
		// "$set"：无论找到还是没找到，都把 updatedAt 字段设置为现在的时间
		"$set": bson.M{
			consts.CreatedAt: time.Now(),
		},
		// 只有在没找到，需要插入新纪录的情况下，才设置这些字段
		"$setOnInsert": bson.M{
			consts.ID:     primitive.NewObjectID().Hex(),
			consts.UserId: userID,
			consts.Query:  query,
		},
	}

	// 如果没找到匹配的，就执行插入操作
	updateOptions := options.Update().SetUpsert(true)

	cacheKey := CacheKeyPrefix + userID
	_, err := m.conn.UpdateOne(ctx, cacheKey, filter, update, updateOptions)
	return err
}
