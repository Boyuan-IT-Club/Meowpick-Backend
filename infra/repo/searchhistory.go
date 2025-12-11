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
	FindManyByUserID(ctx context.Context, userId string) ([]*model.SearchHistory, error)
	CountByUserID(ctx context.Context, userId string) (int64, error)
	DeleteOldestByUserID(ctx context.Context, userId string) error
	UpsertByUserIDAndQuery(ctx context.Context, userId string, query string) error
}

type SearchHistoryRepo struct {
	conn *monc.Model
}

func NewSearchHistoryRepo(cfg *config.Config) *SearchHistoryRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, SearchHistoryCollectionName, cfg.Cache)
	return &SearchHistoryRepo{conn: conn}
}

// FindManyByUserID 查询用户最近15条搜索历史
func (r *SearchHistoryRepo) FindManyByUserID(ctx context.Context, userId string) ([]*model.SearchHistory, error) {
	histories := []*model.SearchHistory{}
	if err := r.conn.Find(
		ctx,
		&histories,
		bson.M{consts.UserID: userId},
		options.Find().SetSort(page.DSort(consts.CreatedAt, -1)).SetLimit(consts.SearchHistoryLimit), // 倒序，最新的在前面
	); err != nil {
		return nil, err
	}
	return histories, nil
}

// CountByUserID 查询用户的搜索历史数量
func (r *SearchHistoryRepo) CountByUserID(ctx context.Context, userId string) (int64, error) {
	return r.conn.CountDocuments(ctx, bson.M{consts.UserID: userId})
}

// DeleteOldestByUserID 删除用户最旧的一条搜索历史
func (r *SearchHistoryRepo) DeleteOldestByUserID(ctx context.Context, userId string) error {
	oldest := &model.SearchHistory{}
	if err := r.conn.FindOneAndDelete(
		ctx,
		SearchHistoryCacheKeyPrefix+userId,
		oldest,
		bson.M{consts.UserID: userId},
		options.FindOneAndDelete().SetSort(page.DSort(consts.CreatedAt, 1)), // 升序，最旧的在前面
	); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// UpsertByUserIDAndQuery 插入或更新用户搜索历史
func (r *SearchHistoryRepo) UpsertByUserIDAndQuery(ctx context.Context, userId string, query string) error {
	if _, err := r.conn.UpdateOne(ctx,
		SearchHistoryCacheKeyPrefix+userId,
		bson.M{consts.UserID: userId, consts.Query: query},
		bson.M{
			"$set": bson.M{consts.CreatedAt: time.Now()}, // 无论找到还是没找到，都把 createdAt 字段设置为现在的时间
			"$setOnInsert": bson.M{ // 只有在没找到，需要插入新纪录的情况下，才设置这些字段
				consts.ID:     primitive.NewObjectID().Hex(),
				consts.UserID: userId,
				consts.Query:  query,
			},
		},
		options.Update().SetUpsert(true), // 如果没找到匹配的，就执行插入操作
	); err != nil {
		return err
	}
	return nil
}
