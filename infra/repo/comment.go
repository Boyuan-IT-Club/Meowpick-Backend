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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

var _ ICommentRepo = (*CommentRepo)(nil)

const (
	CommentCacheKeyPrefix = "meowpick:comment:courseID="
	CommentCollectionName = "comment"
)

type ICommentRepo interface {
	Insert(ctx context.Context, c *model.Comment) error
	Count(ctx context.Context) (int64, error)
	FindManyByUserID(ctx context.Context, param *dto.PageParam, userID string) ([]*model.Comment, int64, error)
	FindManyByCourseID(ctx context.Context, param *dto.PageParam, courseID string) ([]*model.Comment, int64, error)
	CountTagsByCourseID(ctx context.Context, courseID string) (map[string]int, error)
}

type CommentRepo struct {
	conn *monc.Model
}

func NewCommentRepo(cfg *config.Config) *CommentRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CommentCollectionName, cfg.Cache)
	return &CommentRepo{conn: conn}
}

func (r *CommentRepo) Insert(ctx context.Context, c *model.Comment) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	_, err := r.conn.InsertOneNoCache(ctx, c)
	return err
}

func (r *CommentRepo) Count(ctx context.Context) (int64, error) {
	// 考虑到性能，暂使用EstimatedDocumentCount
	//filter := bson.M{consts.Deleted: bson.M{"$ne": true}}
	//count, err := m.conn.CountDocuments(ctx, filter)

	count, err := r.conn.EstimatedDocumentCount(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *CommentRepo) FindManyByUserID(ctx context.Context, param *dto.PageParam, userID string) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	filter := bson.M{consts.UserID: userID, consts.Deleted: bson.M{"$ne": true}}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	ops := page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1))

	if err = r.conn.Find(ctx, &comments, filter, ops); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (r *CommentRepo) FindManyByCourseID(ctx context.Context, param *dto.PageParam, courseID string) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	filter := bson.M{consts.CourseID: courseID, consts.Deleted: bson.M{"$ne": true}}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	ops := page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1))

	if err = r.conn.Find(ctx, &comments, filter, ops); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (r *CommentRepo) CountTagsByCourseID(ctx context.Context, courseID string) (map[string]int, error) {
	// 数据库聚合实现标签count 建议在/api/search接口封装CourseVO时起go routine获得CourseVO的tagCount字段
	// 构建管道
	pipeline := bson.A{
		// 阶段1：筛选符合条件的文档
		bson.M{"$match": bson.M{
			consts.CourseID: courseID,
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
			consts.ID: "$tags",
			"count":   bson.M{"$sum": 1}, // 这里sum使用的是int64还是int result map[string]xxx需要和sum的类型保持一致
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

	if err := r.conn.Aggregate(ctx, &results, pipeline); err != nil {
		logs.Errorf("Aggregate failed for courseID=%s: %v", courseID, err)
		return nil, err
	}

	// 转换为 map
	result := make(map[string]int)
	for _, item := range results {
		result[item.Tag] = item.Count
	}
	return result, nil
}
