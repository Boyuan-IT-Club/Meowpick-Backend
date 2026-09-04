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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ ICourseRepo = (*CourseRepo)(nil)

const (
	CourseCollectionName      = "course"
	CourseProposalIDIndexName = "course_proposal_id_unique"
)

type ICourseRepo interface {
	FindByID(ctx context.Context, id string) (*model.Course, error)
	FindManyByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByNameLike(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByTeacherID(ctx context.Context, teacherId string, param *dto.PageParam) ([]*model.Course, int64, error)
	FindRecentByTeacherIDs(ctx context.Context, teacherIDs []string, limitPerTeacher int64) (map[string][]*model.Course, error)
	FindManyByCategoryID(ctx context.Context, categoryId int32, param *dto.PageParam) ([]*model.Course, int64, error)
	FindManyByDepartmentID(ctx context.Context, departmentId int32, param *dto.PageParam) ([]*model.Course, int64, error)

	GetDepartmentsByName(ctx context.Context, name string) ([]int32, error)
	GetCategoriesByName(ctx context.Context, name string) ([]int32, error)
	GetCampusesByName(ctx context.Context, name string) ([]int32, error)
	GetSuggestionsByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error)
	GetSuggestionsByCode(ctx context.Context, code string, param *dto.PageParam) ([]*model.Course, int64, error)

	IsCourseInExistingCourses(ctx context.Context, vo *model.Course) (bool, error)
	FindByNameAndCode(ctx context.Context, name, code string) ([]*model.Course, error)
	FindByProposalID(ctx context.Context, proposalID string) (*model.Course, error)
	FindByProposalIDIncludeDeleted(ctx context.Context, proposalID string) (*model.Course, error)
	SoftDeleteByID(ctx context.Context, courseID string) error
	Insert(ctx context.Context, course *model.Course) error
	UpdateCourse(ctx context.Context, course *model.Course) error
}

type CourseRepo struct {
	conn *monc.Model
}

// Insert 插入一个新的课程
func (r *CourseRepo) Insert(ctx context.Context, course *model.Course) error {
	_, err := r.conn.InsertOneNoCache(ctx, course)
	return err
}
func NewCourseRepo(cfg *config.Config) (*CourseRepo, error) {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, CourseCollectionName, cfg.Cache)
	repository := &CourseRepo{conn: conn}
	_, err := conn.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: consts.ProposalID, Value: 1}},
		Options: options.Index().
			SetName(CourseProposalIDIndexName).
			SetUnique(true).
			SetPartialFilterExpression(bson.M{consts.ProposalID: bson.M{"$type": "string", "$gt": ""}}),
	})
	if err != nil {
		return nil, err
	}
	return repository, nil
}

// FindByID 根据课程ID查询课程
func (r *CourseRepo) FindByID(ctx context.Context, id string) (*model.Course, error) {
	course := &model.Course{}
	if err := r.conn.FindOneNoCache(ctx, course, bson.M{
		consts.ID:      id,
		consts.Deleted: bson.M{"$ne": true},
	}); err != nil {
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
	filter := bson.M{consts.Name: name, consts.Deleted: bson.M{"$ne": true}}
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
	filter := bson.M{
		consts.Name:    bson.M{"$regex": primitive.Regex{Pattern: name, Options: "i"}},
		consts.Deleted: bson.M{"$ne": true},
	}
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
	filter := bson.M{consts.TeacherIDs: teacherId, consts.Deleted: bson.M{"$ne": true}}
	if err := r.conn.Find(ctx, &courses, filter,
		page.FindPageOption(param).SetSort(bson.D{
			{Key: consts.CreatedAt, Value: -1},
			{Key: consts.ID, Value: 1}, // 添加_id作为二级排序，确保排序稳定性
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

type teacherCourseGroup struct {
	TeacherID string `bson:"_id"`
	Courses   []struct {
		ID   string `bson:"id"`
		Name string `bson:"name"`
	} `bson:"courses"`
}

// FindRecentByTeacherIDs 批量查询每位教师最近创建的未删除课程。
func (r *CourseRepo) FindRecentByTeacherIDs(
	ctx context.Context,
	teacherIDs []string,
	limitPerTeacher int64,
) (map[string][]*model.Course, error) {
	result := make(map[string][]*model.Course, len(teacherIDs))
	if len(teacherIDs) == 0 || limitPerTeacher <= 0 {
		return result, nil
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			consts.TeacherIDs: bson.M{"$in": teacherIDs},
			consts.Deleted:    bson.M{"$ne": true},
		}},
		{"$unwind": "$" + consts.TeacherIDs},
		{"$match": bson.M{consts.TeacherIDs: bson.M{"$in": teacherIDs}}},
		{"$sort": bson.D{
			{Key: consts.TeacherIDs, Value: 1},
			{Key: consts.CreatedAt, Value: -1},
			{Key: consts.ID, Value: 1},
		}},
		{"$group": bson.M{
			"_id": "$" + consts.TeacherIDs,
			"courses": bson.M{"$push": bson.M{
				"id":   "$" + consts.ID,
				"name": "$" + consts.Name,
			}},
		}},
		{"$project": bson.M{
			"courses": bson.M{"$slice": bson.A{"$courses", limitPerTeacher}},
		}},
	}

	var groups []teacherCourseGroup
	if err := r.conn.Aggregate(ctx, &groups, pipeline); err != nil {
		return nil, err
	}

	for _, group := range groups {
		courses := make([]*model.Course, 0, len(group.Courses))
		for _, brief := range group.Courses {
			courses = append(courses, &model.Course{ID: brief.ID, Name: brief.Name})
		}
		result[group.TeacherID] = courses
	}
	return result, nil
}

// FindManyByCategoryID 根据课程分类ID分页查询课程
func (r *CourseRepo) FindManyByCategoryID(ctx context.Context, categoryId int32, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{consts.Category: categoryId, consts.Deleted: bson.M{"$ne": true}}
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
	filter := bson.M{consts.Department: departmentId, consts.Deleted: bson.M{"$ne": true}}
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
	results, err := r.conn.Distinct(ctx, consts.Department, bson.M{
		consts.Name:    name,
		consts.Deleted: bson.M{"$ne": true},
	})
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
	results, err := r.conn.Distinct(ctx, consts.Category, bson.M{
		consts.Name:    name,
		consts.Deleted: bson.M{"$ne": true},
	})
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
	results, err := r.conn.Distinct(ctx, consts.Campuses, bson.M{
		consts.Name:    name,
		consts.Deleted: bson.M{"$ne": true},
	})
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
func (r *CourseRepo) GetSuggestionsByName(ctx context.Context, name string, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{
		consts.Name:    bson.M{"$regex": primitive.Regex{Pattern: name, Options: "i"}},
		consts.Deleted: bson.M{"$ne": true},
	}

	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param)); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// GetSuggestionsByCode 根据课程代码模糊分页查询课程
func (r *CourseRepo) GetSuggestionsByCode(ctx context.Context, code string, param *dto.PageParam) ([]*model.Course, int64, error) {
	courses := []*model.Course{}
	filter := bson.M{
		consts.Code:    bson.M{"$regex": primitive.Regex{Pattern: code, Options: "i"}},
		consts.Deleted: bson.M{"$ne": true},
	}

	if err := r.conn.Find(ctx, &courses, filter, page.FindPageOption(param)); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// IsCourseInExistingCourses 检查课程是否已经存在于现有课程中
// 比较的字段包括: Name, Code, Department, Category, Campuses, TeacherIDs
func (r *CourseRepo) IsCourseInExistingCourses(ctx context.Context, vo *model.Course) (bool, error) {
	filter := bson.M{
		consts.Name:       vo.Name,
		consts.Code:       vo.Code,
		consts.Department: vo.Department,
		consts.Category:   vo.Category,
		consts.Campuses:   vo.Campuses,
		consts.TeacherIDs: vo.TeacherIDs,
		consts.Deleted:    false, // 只检查未删除的提案
	}

	count, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindByNameAndCode 根据课程名称和代码查找未删除的课程
func (r *CourseRepo) FindByNameAndCode(ctx context.Context, name, code string) ([]*model.Course, error) {
	courses := []*model.Course{}
	filter := bson.M{
		consts.Name:    name,
		consts.Code:    code,
		consts.Deleted: bson.M{"$ne": true},
	}
	if err := r.conn.Find(ctx, &courses, filter); err != nil {
		return nil, err
	}
	return courses, nil
}

// FindByProposalID 根据来源提案ID查询未删除的课程，未找到时返回 nil
func (r *CourseRepo) FindByProposalID(ctx context.Context, proposalID string) (*model.Course, error) {
	course := &model.Course{}
	if err := r.conn.FindOneNoCache(ctx, course, bson.M{
		consts.ProposalID: proposalID,
		consts.Deleted:    bson.M{"$ne": true},
	}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return course, nil
}

// SoftDeleteByID 软删除课程（设置deleted为true）
func (r *CourseRepo) SoftDeleteByID(ctx context.Context, courseID string) error {
	filter := bson.M{consts.ID: courseID}
	update := bson.M{
		"$set": bson.M{
			consts.Deleted:   true,
			consts.UpdatedAt: time.Now(),
		},
	}
	_, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	return err
}

// FindByProposalIDIncludeDeleted 根据来源提案ID查询关联的正式课程（包含已软删除的课程），用于审批恢复与贡献值重算
func (r *CourseRepo) FindByProposalIDIncludeDeleted(ctx context.Context, proposalID string) (*model.Course, error) {
	course := &model.Course{}
	if err := r.conn.FindOneNoCache(ctx, course, bson.M{consts.ProposalID: proposalID}); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return course, nil
}

// UpdateCourse 用新的课程信息更新课程并恢复（deleted翻转为false），用于撤回后重新审批
func (r *CourseRepo) UpdateCourse(ctx context.Context, course *model.Course) error {
	filter := bson.M{consts.ID: course.ID}
	update := bson.M{
		"$set": bson.M{
			consts.Name:       course.Name,
			consts.Code:       course.Code,
			consts.TeacherIDs: course.TeacherIDs,
			consts.Department: course.Department,
			consts.Category:   course.Category,
			consts.Campuses:   course.Campuses,
			consts.Deleted:    false,
			consts.ProposalID: course.ProposalID,
			consts.UpdatedAt:  time.Now(),
		},
	}
	_, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	return err
}
