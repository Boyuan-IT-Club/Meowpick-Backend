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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ ITeacherRepo = (*TeacherRepo)(nil)

const (
	TeacherCacheKeyPrefix  = "meowpick:teacher:"
	TeacherCollectionName  = "teacher"
	TeacherName2IDCacheKey = "meowpick:teacher_name_to_id:" // 再建一套name->ID的缓存
)

type ITeacherRepo interface {
	Insert(ctx context.Context, teacher *model.Teacher) (ID string, err error)
	FindByID(ctx context.Context, ID string) (*model.Teacher, error)
	GetSuggestions(ctx context.Context, keyword string, param *dto.PageParam) ([]*model.Teacher, error)
	Count(ctx context.Context, keyword string) (int64, error)
	FindIDByName(ctx context.Context, name string) (string, error)
}

type TeacherRepo struct {
	conn *monc.Model
}

func NewTeacherRepo(cfg *config.Config) *TeacherRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, TeacherCollectionName, cfg.Cache)
	return &TeacherRepo{conn: conn}
}

func (r *TeacherRepo) Insert(ctx context.Context, teacher *model.Teacher) (string, error) {
	cacheKey := TeacherCacheKeyPrefix + teacher.ID
	if _, err := r.conn.InsertOne(ctx, cacheKey, teacher); err != nil {
		return "", err
	}
	// 设置name->id映射缓存
	nameCacheKey := TeacherName2IDCacheKey + teacher.Name
	if err := r.conn.SetCache(nameCacheKey, cacheKey); err != nil {
		logs.Warnf("Failed to set name-to-id mapping cache: %v", err)
	}
	return teacher.ID, nil
}

func (r *TeacherRepo) FindByID(ctx context.Context, ID string) (*model.Teacher, error) {
	var teacher model.Teacher
	//cacheKey := CourseCacheKeyPrefix + ID
	if err := r.conn.FindOneNoCache(ctx, &teacher, bson.M{consts.ID: ID}) err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &teacher, nil
}

func (r *TeacherRepo) GetSuggestions(ctx context.Context, keyword string, param *dto.PageParam) ([]*model.Teacher, error) {
	var teachers []*model.Teacher
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	ops := page.FindPageOption(param)

	err := r.conn.Find(ctx, &teachers, filter, ops)
	if err != nil {
		return nil, err
	}
	return teachers, nil
}

func (r *TeacherRepo) Count(ctx context.Context, keyword string) (int64, error) {
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: keyword, Options: "i"}}}
	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TeacherRepo) FindIDByName(ctx context.Context, name string) (string, error) {
	nameCacheKey := TeacherName2IDCacheKey + name
	var teacherID string
	// 先查name-id缓存
	if err := r.conn.GetCache(nameCacheKey, &teacherID); err != nil && teacherID != "" {
		return teacherID, nil
	}
	filter := bson.M{consts.Name: name}
	var teacher model.Teacher

	// 使用NoCache版本，避免使用错误的缓存键
	if err := r.conn.FindOneNoCache(ctx, &teacher, filter); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	// 设置缓存键
	cacheKey := TeacherCacheKeyPrefix + teacher.ID
	if err := r.conn.SetCache(nameCacheKey, cacheKey); err != nil {
		logs.Errorf("Failed to set name-to-id mapping cache %s for teacher: %v", nameCacheKey, err)
	}

	return teacher.ID, nil
}
