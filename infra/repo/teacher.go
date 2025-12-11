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
	TeacherCollectionName  = "teacher"
	TeacherCacheKeyPrefix  = "meowpick:teacher:"
	TeacherName2IDCacheKey = "meowpick:teacher_name_to_id:" // name->ID缓存
)

type ITeacherRepo interface {
	Insert(ctx context.Context, teacher *model.Teacher) error
	GetSuggestions(ctx context.Context, name string, param *dto.PageParam) ([]*model.Teacher, error)
	IsExistByID(ctx context.Context, name string) (bool, error)

	FindByID(ctx context.Context, id string) (*model.Teacher, error)
	FindIDByName(ctx context.Context, name string) (string, error)
}

type TeacherRepo struct {
	conn *monc.Model
}

func NewTeacherRepo(cfg *config.Config) *TeacherRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, TeacherCollectionName, cfg.Cache)
	return &TeacherRepo{conn: conn}
}

// Insert 插入教师
func (r *TeacherRepo) Insert(ctx context.Context, teacher *model.Teacher) error {
	cacheKey := TeacherCacheKeyPrefix + teacher.ID
	if _, err := r.conn.InsertOne(ctx, cacheKey, teacher); err != nil {
		return err
	}
	// 设置name->ID映射缓存
	if err := r.conn.SetCache(TeacherName2IDCacheKey+teacher.Name, cacheKey); err != nil {
		logs.CtxWarnf(ctx, "TeacherRepo SetCache name to id mapping failed: %v", err)
	}
	return nil
}

// GetSuggestions 根据教师名称模糊分页查询教师
func (r *TeacherRepo) GetSuggestions(ctx context.Context, name string, param *dto.PageParam) ([]*model.Teacher, error) {
	teachers := []*model.Teacher{}
	if err := r.conn.Find(ctx, &teachers, bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: name, Options: "i"}}}, page.FindPageOption(param)); err != nil {
		return nil, err
	}
	return teachers, nil
}

// IsExistByID 根据教师ID判断教师是否存在
func (r *TeacherRepo) IsExistByID(ctx context.Context, id string) (bool, error) {
	count, err := r.conn.CountDocuments(ctx, bson.M{consts.ID: id})
	return count > 0, err
}

// FindByID 根据教师ID查询教师
func (r *TeacherRepo) FindByID(ctx context.Context, id string) (*model.Teacher, error) {
	teacher := &model.Teacher{}
	if err := r.conn.FindOneNoCache(ctx, teacher, bson.M{consts.ID: id}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return teacher, nil
}

// FindIDByName 根据教师名称查询教师ID
func (r *TeacherRepo) FindIDByName(ctx context.Context, name string) (string, error) {
	var teacherId string
	teacher := &model.Teacher{}
	nameCacheKey := TeacherName2IDCacheKey + name
	// 先查name-id缓存
	if err := r.conn.GetCache(nameCacheKey, &teacherId); err != nil && teacherId != "" {
		return teacherId, nil
	}
	// 使用NoCache版本，避免使用错误的缓存键
	if err := r.conn.FindOneNoCache(ctx, teacher, bson.M{consts.Name: name}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	// 设置缓存键
	if err := r.conn.SetCache(nameCacheKey, TeacherCacheKeyPrefix+teacher.ID); err != nil {
		logs.CtxWarnf(ctx, "TeacherRepo SetCache name to id mapping failed: %v", err)
	}
	return teacher.ID, nil
}
