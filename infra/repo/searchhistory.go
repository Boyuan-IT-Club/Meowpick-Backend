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
	"errors"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ ISearchHistoryRepo = (*SearchHistoryRepo)(nil)

const (
	SearchHistoryCacheKeyPrefix = "meowpick:searchhistory:"
	SearchHistoryCollectionName = "searchhistory"
)

type ISearchHistoryRepo interface {
	FindByUserID(ctx context.Context, userID string) ([]*model.SearchHistory, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	DeleteOldestByUserID(ctx context.Context, userID string) error
	UpsertByUserIDAndQuery(ctx context.Context, userID string, query string) error
}

type SearchHistoryRepo struct {
	conn *monc.Model
}

func NewSearchHistoryRepo(cfg *config.Config) *SearchHistoryRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, SearchHistoryCollectionName, cfg.Cache)
	return &SearchHistoryRepo{conn: conn}
}

func (r *SearchHistoryRepo) FindByUserID(ctx context.Context, userID string) ([]*model.SearchHistory, error) {
	var histories []*model.SearchHistory

	ops := options.Find()
	ops.SetSort(page.DSort(consts.CreatedAt, -1)) // 降序, 最新的在最前面
	ops.SetLimit(consts.SearchHistoryLimit)

	filter := bson.M{consts.UserID: userID}

	if err := r.conn.Find(ctx, &histories, filter, ops); err != nil {
		return nil, err
	}
	return histories, nil
}

func (r *SearchHistoryRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.conn.CountDocuments(ctx, bson.M{consts.UserID: userID})
}

func (r *SearchHistoryRepo) DeleteOldestByUserID(ctx context.Context, userID string) error {
	var oldest model.SearchHistory
	ops := options.FindOneAndDelete()
	ops.SetSort(page.DSort(consts.CreatedAt, 1))

	cacheKey := SearchHistoryCacheKeyPrefix + userID
	if err := r.conn.FindOneAndDelete(ctx, cacheKey, &oldest, bson.M{consts.UserID: userID}, ops); err != nil && !errors.Is(err, monc.ErrNotFound) {
		return err
	}
	return nil
}

func (r *SearchHistoryRepo) UpsertByUserIDAndQuery(ctx context.Context, userID string, query string) error {
	filter := bson.M{
		consts.UserID: userID,
		consts.Query:  query,
	}

	// 定义“更新操作”
	update := bson.M{
		// "$set"：无论找到还是没找到，都把 updatedAt 字段设置为现在的时间
		"$set": bson.M{
			consts.CreatedAt: time.Now(),
		},
		// 只有在没找到，需要插入新纪录的情况下，才设置这些字段
		"$setOnInsert": bson.M{
			consts.ID:     primitive.NewObjectID().Hex(),
			consts.UserID: userID,
			consts.Query:  query,
		},
	}

	// 如果没找到匹配的，就执行插入操作
	updateOptions := options.Update().SetUpsert(true)

	cacheKey := SearchHistoryCacheKeyPrefix + userID
	_, err := r.conn.UpdateOne(ctx, cacheKey, filter, update, updateOptions)
	return err
}
