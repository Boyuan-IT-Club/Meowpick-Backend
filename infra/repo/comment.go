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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ ICommentRepo = (*CommentRepo)(nil)

const (
	CommentCollectionName = "comment"
)

type ICommentRepo interface {
	Insert(ctx context.Context, c *model.Comment) error
	Count(ctx context.Context) (int64, error)
	GetTagsByCourseID(ctx context.Context, courseId string) (map[string]int64, error)

	FindManyByUserID(ctx context.Context, param *dto.PageParam, userId string) ([]*model.Comment, int64, error)
	FindManyByCourseID(ctx context.Context, param *dto.PageParam, courseId string) ([]*model.Comment, int64, error)
}

type CommentRepo struct {
	conn *monc.Model
}

func NewCommentRepo(cfg *config.Config) *CommentRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CommentCollectionName, cfg.Cache)
	return &CommentRepo{conn: conn}
}

// Insert 插入评论
func (r *CommentRepo) Insert(ctx context.Context, c *model.Comment) error {
	_, err := r.conn.InsertOneNoCache(ctx, c)
	return err
}

// Count 统计评论总数
func (r *CommentRepo) Count(ctx context.Context) (int64, error) {
	return r.conn.CountDocuments(ctx, bson.M{consts.Deleted: bson.M{"$ne": true}})
}

// GetTagsByCourseID 根据课程ID统计课程数量前3的标签
func (r *CommentRepo) GetTagsByCourseID(ctx context.Context, courseId string) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{consts.CourseID, courseId},
			{consts.Deleted, bson.M{"$ne": true}},
			{consts.Tags, bson.M{"$ne": nil}},
		}}},
		{{"$unwind", bson.D{
			{"path", "$tags"},
			{"preserveNullAndEmptyArrays", false},
		}}},
		{{"$match", bson.D{
			{consts.Tags, bson.M{"$ne": ""}},
		}}},
		{{"$group", bson.D{
			{consts.ID, "$tags"},
			{consts.Count, bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{
			{consts.Count, -1},
		}}},
		{{"$limit", 3}},
	}
	var tags []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := r.conn.Aggregate(ctx, &tags, pipeline); err != nil {
		return nil, err
	}
	results := make(map[string]int64)
	for _, tag := range tags {
		results[tag.ID] = tag.Count
	}
	return results, nil
}

// FindManyByUserID 根据用户ID分页查询用户所有评论
func (r *CommentRepo) FindManyByUserID(ctx context.Context, param *dto.PageParam, userId string) ([]*model.Comment, int64, error) {
	comments := []*model.Comment{}
	filter := bson.M{consts.UserID: userId, consts.Deleted: bson.M{"$ne": true}}
	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if err = r.conn.Find(ctx, &comments, filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

// FindManyByCourseID 根据课程ID分页查询课程所有评论
func (r *CommentRepo) FindManyByCourseID(ctx context.Context, param *dto.PageParam, courseId string) ([]*model.Comment, int64, error) {
	comments := []*model.Comment{}
	filter := bson.M{consts.CourseID: courseId, consts.Deleted: bson.M{"$ne": true}}
	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if err = r.conn.Find(ctx, &comments, filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}
