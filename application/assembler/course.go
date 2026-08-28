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

package assembler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mappingNamesContextKey struct{}

type mappingNamesByType map[model.MappingType]map[int32]string

var _ ICourseAssembler = (*CourseAssembler)(nil)

type ICourseAssembler interface {
	ToCourseVO(ctx context.Context, db *model.Course) (*dto.CourseVO, error)
	ToCourseDB(ctx context.Context, vo *dto.CourseVO) (*model.Course, error)
	ToCourseDBDryRun(ctx context.Context, vo *dto.CourseVO) (*model.Course, error)
	ToCourseDBDryRunFromProposalCourse(ctx context.Context, vo *dto.ProposalCourseVO) (*model.Course, error)
	ToCourseDBFromProposalCourse(ctx context.Context, vo *dto.ProposalCourseVO) (*model.Course, error)
	ToProposalCourseDB(ctx context.Context, vo *dto.ProposalCourseVO) (*model.ProposalCourse, error)
	ToProposalCourseVO(ctx context.Context, db *model.ProposalCourse) (*dto.ProposalCourseVO, error)
	ToProposalCourseVOFromCourse(ctx context.Context, db *model.Course) (*dto.ProposalCourseVO, error)
	ToCourseVOArray(ctx context.Context, dbs []*model.Course) ([]*dto.CourseVO, error)
	ToCourseDBArray(ctx context.Context, vos []*dto.CourseVO) ([]*model.Course, error)
	ToPaginatedCourses(cxt context.Context, dbs []*model.Course, total int64, pageParam *dto.PageParam) (*dto.PaginatedCourses, error)
}

type CourseAssembler struct {
	CommentRepo *repo.CommentRepo
	TeacherRepo *repo.TeacherRepo
	CourseRepo  *repo.CourseRepo
}

var CourseAssemblerSet = wire.NewSet(
	wire.Struct(new(CourseAssembler), "*"),
	wire.Bind(new(ICourseAssembler), new(*CourseAssembler)),
)

// ToCourseVO 单个CourseDB转CourseVO (DB to VO)
func (a *CourseAssembler) ToCourseVO(ctx context.Context, db *model.Course) (*dto.CourseVO, error) {
	// 获得课程前三多的tag
	tagCountChan := make(chan map[string]int64, 1)
	go func() {
		tagCount, err := a.CommentRepo.GetTagsByCourseID(ctx, db.ID)
		if err != nil {
			logs.CtxErrorf(ctx, "[CommentRepo] [GetTagsByCourseID] error: %v", err)
			tagCountChan <- make(map[string]int64)
		} else {
			tagCountChan <- tagCount
		}
	}()

	// 获取校区列表
	campuses := make([]string, 0)
	for _, campusId := range db.Campuses {
		campusName := mappingNameFromContext(ctx, model.MappingTypeCampus, campusId)
		if campusName != "" {
			campuses = append(campuses, campusName)
		}
	}

	// 获得教师VO
	teacherVOs := make([]*dto.TeacherVO, 0, len(db.TeacherIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, tid := range db.TeacherIDs {
		wg.Add(1)
		go func(teacherID string) {
			defer wg.Done()
			teacher, err := a.TeacherRepo.FindByID(ctx, teacherID)
			if err != nil {
				logs.CtxErrorf(ctx, "[TeacherRepo] [FindByID] find teacher %s error: %v", teacherID, err)
				return
			}

			if teacher != nil {
				mu.Lock()
				teacherVOs = append(teacherVOs, &dto.TeacherVO{
					ID:         teacher.ID,
					Name:       teacher.Name,
					Title:      teacher.Title,
					Department: mappingNameFromContext(ctx, model.MappingTypeDepartment, teacher.Department),
				})
				mu.Unlock()
			}
		}(tid)
	}
	wg.Wait()

	// 等待tagCount结果
	tagCount := <-tagCountChan

	return &dto.CourseVO{
		ID:         db.ID,
		Name:       db.Name,
		Code:       db.Code,
		Category:   mappingNameFromContext(ctx, model.MappingTypeCategory, db.Category),
		Campuses:   campuses,
		Department: mappingNameFromContext(ctx, model.MappingTypeDepartment, db.Department),
		Teachers:   teacherVOs,
		TagCount:   tagCount,
	}, nil
}

// ToCourseDB 单个CourseVO转CourseDB (VO to DB)(会执行自动注册)
func (a *CourseAssembler) ToCourseDB(ctx context.Context, vo *dto.CourseVO) (*model.Course, error) {
	if vo == nil {
		return nil, nil
	}
	// 将校区名称转换为ID
	var campusIDs []int32
	for _, campus := range vo.Campuses {
		campusID, err := mapping.Data.ResolveCampus(ctx, campus)
		if err != nil {
			return nil, err
		}
		campusIDs = append(campusIDs, campusID)
	}

	departmentID, err := mapping.Data.ResolveOrCreateDepartment(ctx, vo.Department)
	if err != nil {
		return nil, fmt.Errorf("resolve course department: %w", err)
	}

	categoryID, err := mapping.Data.ResolveOrCreateCategory(ctx, vo.Category)
	if err != nil {
		return nil, fmt.Errorf("resolve course category: %w", err)
	}
	// 处理教师 - 自动创建不存在的教师
	var teacherIDs []string
	for _, teacher := range vo.Teachers {
		// 检查教师是否已存在
		existingTeacherID, err := a.TeacherRepo.GetIDByName(ctx, teacher.Name)
		if err != nil {
			return nil, fmt.Errorf("find teacher %q: %w", teacher.Name, err)
		}

		var teacherID string
		if existingTeacherID != "" {
			// 教师已存在，使用现有ID
			teacherID = existingTeacherID
		} else {
			// 教师不存在，创建新教师
			teacherDepartment, err := mapping.Data.ResolveOrCreateDepartment(ctx, teacher.Department)
			if err != nil {
				return nil, fmt.Errorf("resolve teacher department: %w", err)
			}
			now := time.Now().UTC()
			newTeacher := &model.Teacher{
				ID:         primitive.NewObjectID().Hex(),
				Name:       teacher.Name,
				Title:      teacher.Title,
				Department: teacherDepartment,
				CreatedAt:  now,
				UpdatedAt:  now,
			}

			if err := a.TeacherRepo.Insert(ctx, newTeacher); err != nil {
				return nil, fmt.Errorf("insert teacher %q: %w", teacher.Name, err)
			}
			teacherID = newTeacher.ID
		}

		teacherIDs = append(teacherIDs, teacherID)
	}

	return &model.Course{
		ID:         vo.ID,
		Name:       vo.Name,
		Code:       vo.Code,
		Category:   categoryID,
		Campuses:   campusIDs,
		Department: departmentID,
		TeacherIDs: teacherIDs,
	}, nil
}

// ToCourseDBDryRun CourseVO转CourseDB (VO to DB) - 不执行自动注册
func (a *CourseAssembler) ToCourseDBDryRun(ctx context.Context, vo *dto.CourseVO) (*model.Course, error) {
	// 将校区名称转换为ID
	var campusIDs []int32
	for _, campus := range vo.Campuses {
		campusID := mapping.Data.GetCampusIDByName(campus)
		if campusID != 0 {
			campusIDs = append(campusIDs, campusID)
		}
	}

	// 处理院系
	departmentID := mapping.Data.GetDepartmentIDByName(vo.Department)

	// 处理课程类别
	categoryID := mapping.Data.GetCategoryIDByName(vo.Category)

	// 处理教师
	var teacherIDs []string
	for _, teacher := range vo.Teachers {
		// 已有正式教师ID，直接复用；新教师（无ID）在 DryRun 中不参与匹配
		if teacher.ID != "" {
			teacherIDs = append(teacherIDs, teacher.ID)
		}
	}

	return &model.Course{
		ID:         vo.ID,
		Name:       vo.Name,
		Code:       vo.Code,
		Category:   categoryID,
		Campuses:   campusIDs,
		Department: departmentID,
		TeacherIDs: teacherIDs,
	}, nil
}

// ToCourseDBDryRunFromProposalCourse ProposalCourseVO转CourseDB (VO to DB) - 不执行自动注册
func (a *CourseAssembler) ToCourseDBDryRunFromProposalCourse(ctx context.Context, vo *dto.ProposalCourseVO) (*model.Course, error) {
	if vo == nil {
		return nil, nil
	}
	// 将校区名称转换为ID
	var campusIDs []int32
	for _, campus := range vo.Campuses {
		campusID := mapping.Data.GetCampusIDByName(campus)
		if campusID != 0 {
			campusIDs = append(campusIDs, campusID)
		}
	}

	// 处理院系
	departmentID := mapping.Data.GetDepartmentIDByName(vo.Department)

	// 处理课程类别
	categoryID := mapping.Data.GetCategoryIDByName(vo.Category)

	// 处理教师
	var teacherIDs []string
	for _, teacher := range vo.Teachers {
		// 已有正式教师ID，直接复用；新教师（无ID）在 DryRun 中不参与匹配
		if teacher.ID != "" {
			teacherIDs = append(teacherIDs, teacher.ID)
		}
	}

	return &model.Course{
		ID:         vo.ID,
		Name:       vo.Name,
		Code:       vo.Code,
		Category:   categoryID,
		Campuses:   campusIDs,
		Department: departmentID,
		TeacherIDs: teacherIDs,
	}, nil
}

// ToCourseDBFromProposalCourse ProposalCourseVO转CourseDB (VO to DB) - 执行自动注册
func (a *CourseAssembler) ToCourseDBFromProposalCourse(ctx context.Context, vo *dto.ProposalCourseVO) (*model.Course, error) {
	if vo == nil {
		return nil, nil
	}
	// 将校区名称转换为ID
	var campusIDs []int32
	for _, campus := range vo.Campuses {
		campusID, err := mapping.Data.ResolveCampus(ctx, campus)
		if err != nil {
			return nil, err
		}
		campusIDs = append(campusIDs, campusID)
	}

	departmentID, err := mapping.Data.ResolveOrCreateDepartment(ctx, vo.Department)
	if err != nil {
		return nil, fmt.Errorf("resolve course department: %w", err)
	}

	categoryID, err := mapping.Data.ResolveOrCreateCategory(ctx, vo.Category)
	if err != nil {
		return nil, fmt.Errorf("resolve course category: %w", err)
	}

	// 处理教师 - 已有ID直接复用；无ID视为用户手动输入的新教师，直接创建
	var teacherIDs []string
	for _, teacher := range vo.Teachers {
		// 用户从建议列表选择的已有教师，直接复用其ID
		if teacher.ID != "" {
			teacherIDs = append(teacherIDs, teacher.ID)
			continue
		}

		// 无ID：视为新教师，直接创建
		teacherDepartment, err := mapping.Data.ResolveOrCreateDepartment(ctx, teacher.Department)
		if err != nil {
			return nil, fmt.Errorf("resolve teacher department: %w", err)
		}
		now := time.Now().UTC()
		newTeacher := &model.Teacher{
			ID:         primitive.NewObjectID().Hex(),
			Name:       teacher.Name,
			Title:      teacher.Title,
			Department: teacherDepartment,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := a.TeacherRepo.Insert(ctx, newTeacher); err != nil {
			return nil, fmt.Errorf("insert teacher %q: %w", teacher.Name, err)
		}
		teacherIDs = append(teacherIDs, newTeacher.ID)
	}

	return &model.Course{
		ID:         vo.ID,
		Name:       vo.Name,
		Code:       vo.Code,
		Category:   categoryID,
		Campuses:   campusIDs,
		Department: departmentID,
		TeacherIDs: teacherIDs,
	}, nil
}

// ToProposalCourseDB ProposalCourseVO转ProposalCourse (VO to DB)
func (a *CourseAssembler) ToProposalCourseDB(ctx context.Context, vo *dto.ProposalCourseVO) (*model.ProposalCourse, error) {
	if vo == nil {
		return nil, nil
	}
	// 直接映射，不涉及自动注册和ID转换
	var teachers []*model.ProposalTeacher
	for _, t := range vo.Teachers {
		teachers = append(teachers, &model.ProposalTeacher{
			Name:       t.Name,
			Department: t.Department,
			Title:      t.Title,
			TeacherID:  t.ID,
		})
	}

	return &model.ProposalCourse{
		Name:       vo.Name,
		Code:       vo.Code,
		Teachers:   teachers,
		Department: vo.Department,
		Category:   vo.Category,
		Campuses:   vo.Campuses,
		Deleted:    false,
	}, nil
}

// ToProposalCourseVO ProposalCourse转ProposalCourseVO (DB to VO)
func (a *CourseAssembler) ToProposalCourseVO(ctx context.Context, db *model.ProposalCourse) (*dto.ProposalCourseVO, error) {
	if db == nil {
		return nil, nil
	}
	// 直接映射
	var teachers []*dto.TeacherVO
	for _, t := range db.Teachers {
		teachers = append(teachers, &dto.TeacherVO{
			Name:       t.Name,
			Department: t.Department,
			Title:      t.Title,
			ID:         t.TeacherID,
		})
	}

	return &dto.ProposalCourseVO{
		Name:       db.Name,
		Code:       db.Code,
		Category:   db.Category,
		Campuses:   db.Campuses,
		Department: db.Department,
		Teachers:   teachers,
	}, nil
}

// ToProposalCourseVOFromCourse 正式课程Course转ProposalCourseVO (DB to VO)
// 用于已通过提案关联的正式课程信息展示，不涉及自动注册
func (a *CourseAssembler) ToProposalCourseVOFromCourse(ctx context.Context, db *model.Course) (*dto.ProposalCourseVO, error) {
	if db == nil {
		return nil, nil
	}

	// 获取校区名称列表
	campuses := make([]string, 0, len(db.Campuses))
	for _, campusId := range db.Campuses {
		campusName := mappingNameFromContext(ctx, model.MappingTypeCampus, campusId)
		if campusName != "" {
			campuses = append(campuses, campusName)
		}
	}

	// 获取教师VO
	teacherVOs := make([]*dto.TeacherVO, 0, len(db.TeacherIDs))
	for _, tid := range db.TeacherIDs {
		teacher, err := a.TeacherRepo.FindByID(ctx, tid)
		if err != nil {
			logs.CtxErrorf(ctx, "[TeacherRepo] [FindByID] find teacher %s error: %v", tid, err)
			continue
		}
		if teacher != nil {
			teacherVOs = append(teacherVOs, &dto.TeacherVO{
				ID:         teacher.ID,
				Name:       teacher.Name,
				Title:      teacher.Title,
				Department: mappingNameFromContext(ctx, model.MappingTypeDepartment, teacher.Department),
			})
		}
	}

	return &dto.ProposalCourseVO{
		ID:         db.ID,
		Name:       db.Name,
		Code:       db.Code,
		Category:   mappingNameFromContext(ctx, model.MappingTypeCategory, db.Category),
		Campuses:   campuses,
		Department: mappingNameFromContext(ctx, model.MappingTypeDepartment, db.Department),
		Teachers:   teacherVOs,
	}, nil
}

// ToCourseVOArray CourseDB数组转CourseVO数组 (DB Array to VO Array)
func (a *CourseAssembler) ToCourseVOArray(ctx context.Context, dbs []*model.Course) ([]*dto.CourseVO, error) {
	if len(dbs) == 0 {
		logs.CtxWarnf(ctx, "[CourseAssembler] [ToCourseVOArray] empty course db array")
		return []*dto.CourseVO{}, nil
	}

	// Resolve course-level mappings in three HMGET calls, irrespective of the
	// number of courses. Individual conversion reads these prefetched values.
	ctx = withPrefetchedCourseMappingNames(ctx, dbs)
	courseVOs := make([]*dto.CourseVO, len(dbs))

	type result struct {
		index int
		vo    *dto.CourseVO
		err   error
	}

	resultChan := make(chan result, len(dbs))
	var wg sync.WaitGroup

	for i, c := range dbs {
		wg.Add(1)
		go func(index int, dbCourse *model.Course) {
			defer wg.Done()
			vo, err := a.ToCourseVO(ctx, dbCourse)
			resultChan <- result{index: index, vo: vo, err: err}
		}(i, c)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果，保持顺序
	for r := range resultChan {
		if r.err != nil {
			logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseVO] error: %v", r.err)
			return nil, r.err
		}
		courseVOs[r.index] = r.vo
	}

	return courseVOs, nil
}

// ToCourseDBArray CourseVO数组转CourseDB数组 (VO Array to DB Array)
func (a *CourseAssembler) ToCourseDBArray(ctx context.Context, vos []*dto.CourseVO) ([]*model.Course, error) {
	if len(vos) == 0 {
		logs.CtxWarnf(ctx, "[CourseAssembler] [ToCourseDBArray] empty course vo array")
		return []*model.Course{}, nil
	}

	courses := make([]*model.Course, 0, len(vos))

	for _, vo := range vos {
		db, err := a.ToCourseDB(ctx, vo)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseDB] error: %v", err)
			return nil, err
		}
		if db != nil {
			courses = append(courses, db)
		}
	}

	return courses, nil
}

// ToPaginatedCourses CourseDB数组转paginatedCourses
func (a *CourseAssembler) ToPaginatedCourses(cxt context.Context, courses []*model.Course, total int64, pageParam *dto.PageParam) (*dto.PaginatedCourses, error) {
	vos, err := a.ToCourseVOArray(cxt, courses)

	if err != nil {
		logs.CtxErrorf(cxt, "[CourseAssembler] [ToCourseVOArray] error: %v", err)
		return nil, err
	}

	return &dto.PaginatedCourses{
		Courses:   vos,
		Total:     total,
		PageParam: pageParam,
	}, nil
}

func withPrefetchedCourseMappingNames(ctx context.Context, courses []*model.Course) context.Context {
	codesByType := map[model.MappingType][]int32{
		model.MappingTypeDepartment: {},
		model.MappingTypeCategory:   {},
		model.MappingTypeCampus:     {},
	}
	seen := map[model.MappingType]map[int32]struct{}{
		model.MappingTypeDepartment: {},
		model.MappingTypeCategory:   {},
		model.MappingTypeCampus:     {},
	}
	add := func(mappingType model.MappingType, code int32) {
		if _, ok := seen[mappingType][code]; ok {
			return
		}
		seen[mappingType][code] = struct{}{}
		codesByType[mappingType] = append(codesByType[mappingType], code)
	}
	for _, course := range courses {
		if course == nil {
			continue
		}
		add(model.MappingTypeDepartment, course.Department)
		add(model.MappingTypeCategory, course.Category)
		for _, campus := range course.Campuses {
			add(model.MappingTypeCampus, campus)
		}
	}

	unknownByType := map[model.MappingType]string{
		model.MappingTypeDepartment: "未知开课院系",
		model.MappingTypeCategory:   "未知分类",
		model.MappingTypeCampus:     "未知校区",
	}
	values := make(mappingNamesByType, len(codesByType))
	for mappingType, codes := range codesByType {
		names := mapping.Data.GetNamesByCodes(ctx, mappingType, codes, unknownByType[mappingType])
		values[mappingType] = make(map[int32]string, len(codes))
		for i, code := range codes {
			values[mappingType][code] = names[i]
		}
	}
	return context.WithValue(ctx, mappingNamesContextKey{}, values)
}

func mappingNameFromContext(ctx context.Context, mappingType model.MappingType, code int32) string {
	if values, ok := ctx.Value(mappingNamesContextKey{}).(mappingNamesByType); ok {
		if name := values[mappingType][code]; name != "" {
			return name
		}
	}
	switch mappingType {
	case model.MappingTypeCampus:
		return mapping.Data.GetCampusNameByID(code)
	case model.MappingTypeDepartment:
		return mapping.Data.GetDepartmentNameByID(code)
	case model.MappingTypeCategory:
		return mapping.Data.GetCategoryNameByID(code)
	default:
		return ""
	}
}
