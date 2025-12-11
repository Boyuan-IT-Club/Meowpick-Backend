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
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ ICourseRepo = (*CourseRepo)(nil)

const (
	CourseCacheKeyPrefix = "meowpick:course:"
	CourseCollectionName = "course"
)

type ICourseRepo interface {
	FindByID(ctx context.Context, id string) (*model.Course, error)
	FindManyByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByNameLike(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByTeacherID(ctx context.Context, teacherId string, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByCategoryID(ctx context.Context, categoryId int32, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByDepartmentID(ctx context.Context, departmentId int32, param *dto.PageParam) ([]*model.Course, int64, error)

	GetDepartmentsByName(ctx context.Context, name string) ([]int32, error)
	GetCategoriesByName(ctx context.Context, name string) ([]int32, error)
	GetCampusesByName(ctx context.Context, name string) ([]int32, error)
	GetSuggestions(ctx context.Context, name string, param *dto.PageParam)
}

type CourseRepo struct {
	conn *monc.Model
}

func NewCourseRepo(cfg *config.Config) *CourseRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CourseCollectionName, cfg.Cache)
	return &CourseRepo{conn: conn}
}

// FindByID 根据课程ID查询课程
func (r *CourseRepo) FindByID(ctx context.Context, id string) (*model.Course, error) {
	course := &model.Course{}
	if err := r.conn.FindOneNoCache(ctx, course, bson.M{consts.ID: id}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return course, nil
}

// FindManyByName 根据课程名称分页查询课程
func (r *CourseRepo) FindManyByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{consts.Name: name}
	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1))); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// FindManyByNameLike 根据课程名称分页模糊查询课程
func (r *CourseRepo) FindManyByNameLike(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: name, Options: "i"}}}
	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param)); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// FindManyByTeacherID 根据教师ID分页查询其教授的课程
func (r *CourseRepo) FindManyByTeacherID(ctx context.Context, teacherId string, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{consts.TeacherIDs: teacherId}
	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1))); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// FindManyByCategoryID 根据课程分类ID分页查询课程
func (r *CourseRepo) FindManyByCategoryID(ctx context.Context, categoryId int32, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{consts.Category: categoryId}
	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1))); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// FindManyByDepartmentID 根据开课院系ID分页查询课程
func (r *CourseRepo) FindManyByDepartmentID(ctx context.Context, departmentId int32, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{consts.Department: departmentId}
	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1))); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// GetDepartmentsByName 根据课程名称查询开课院系
func (r *CourseRepo) GetDepartmentsByName(ctx context.Context, name string) ([]int32, error) {
	results, err := r.conn.Distinct(ctx, consts.Department, bson.M{consts.Name: name})
	if err != nil {
		return nil, err
	}
	ids := make([]int32, 0, len(results))
	for _, result := range results {
		if id, ok := result.(int32); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetCategoriesByName 根据课程名称查询课程分类
func (r *CourseRepo) GetCategoriesByName(ctx context.Context, name string) ([]int32, error) {
	results, err := r.conn.Distinct(ctx, consts.Categories, bson.M{consts.Name: name})
	if err != nil {
		return nil, err
	}
	ids := make([]int32, 0, len(results))
	for _, result := range results {
		if id, ok := result.(int32); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetCampusesByName 根据课程名称查询校区
func (r *CourseRepo) GetCampusesByName(ctx context.Context, name string) ([]int32, error) {
	results, err := r.conn.Distinct(ctx, consts.Campuses, bson.M{consts.Name: name})
	if err != nil {
		return nil, err
	}
	ids := make([]int32, 0, len(results))
	for _, result := range results {
		if id, ok := result.(int32); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetSuggestions 根据课程名称模糊分页查询课程
func (r *CourseRepo) GetSuggestions(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, error) {
	courses := []*model.Course{}
	if err := r.conn.Find(ctx, &courses, bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: name, Options: "i"}}}, page.FindPageOption(param)); err != nil {
		return nil, err
	}
	return courses, nil
}
