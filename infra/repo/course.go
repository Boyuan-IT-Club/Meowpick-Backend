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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ ICourseRepo = (*CourseRepo)(nil)

const (
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
	GetSuggestionsByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, error)

	IsCourseInExistingCourses(ctx context.Context, courseVO *dto.CourseVO) (bool, error)
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
	if err := r.conn.Find(ctx, &courses, filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
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
	if err := r.conn.Find(ctx, &courses, filter,
		page.FindPageOption(param).SetSort(bson.D{
			{consts.CreatedAt, -1},
			{consts.ID, 1}, // 添加_id作为二级排序，确保排序稳定性
		}),
	); err != nil {
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
	if err := r.conn.Find(ctx, &courses, filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
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
	if err := r.conn.Find(ctx, &courses, filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
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

// GetSuggestionsByName 根据课程名称模糊分页查询课程
func (r *CourseRepo) GetSuggestionsByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, error) {
	courses := []*model.Course{}
	if err := r.conn.Find(ctx, &courses,
		bson.M{consts.Name: bson.M{"$regex": primitive.Regex{Pattern: name, Options: "i"}}},
		page.FindPageOption(param),
	); err != nil {
		return nil, err
	}
	return courses, nil
}

// IsCourseInExistingCourses 检查课程是否已经存在于现有课程中
// 比较的字段包括: Name, Code, Department, Category, Campuses, TeacherIDs
func (s *CourseRepo) IsCourseInExistingCourses(ctx context.Context, courseVO *dto.CourseVO) (bool, error) {
	// 将DTO中的值转换为ID形式以便数据库查询
	departmentID := mapping.Data.GetDepartmentIDByName(courseVO.Department)
	categoryID := mapping.Data.GetCategoryIDByName(courseVO.Category)

	// 将校区名称转换为ID
	campusIDs := make([]int32, len(courseVO.Campuses))
	for i, campus := range courseVO.Campuses {
		campusIDs[i] = mapping.Data.GetCampusIDByName(campus)
	}

	// 处理教师ID - 在创建新课程时，TeacherIDs初始化为空切片
	// 所以这里我们也应该允许空的TeacherIDs匹配

	// 构造查询条件
	filter := bson.M{
		"name":       courseVO.Name,
		"code":       courseVO.Code,
		"department": departmentID,
		"category":   categoryID,
		"campuses":   bson.M{"$all": campusIDs, "$size": len(campusIDs)},
	}

	// 如果提供了教师信息，则也加入查询条件
	if len(courseVO.Teachers) > 0 {
		teacherIDs := make([]string, len(courseVO.Teachers))
		for i, teacher := range courseVO.Teachers {
			teacherIDs[i] = teacher.ID
		}
		filter["teacherIds"] = bson.M{"$all": teacherIDs, "$size": len(teacherIDs)}
	} else {
		// 如果没有提供教师信息，则查询teacherIds为空或者不存在的记录
		filter["$or"] = []bson.M{
			{"teacherIds": bson.M{"$exists": false}},
			{"teacherIds": bson.M{"$size": 0}},
		}
	}

	// 查询课程是否存在
	count, err := s.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, errorx.WrapByCode(err, errno.ErrProposalCourseFindInCoursesFailed,
			errorx.KV("operation", "check course existence"))
	}

	return count > 0, nil
}
