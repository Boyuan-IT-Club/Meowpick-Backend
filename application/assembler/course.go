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
	"sync"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/google/wire"
)

var _ ICourseAssembler = (*CourseAssembler)(nil)

type ICourseAssembler interface {
	ToCourseVO(ctx context.Context, c *model.Course) (*dto.CourseVO, error)
	ToCourse(ctx context.Context, vo *dto.CourseVO) (*model.Course, error)
	ToCourseVOList(ctx context.Context, courses []*model.Course) ([]*dto.CourseVO, error)
	ToCourseList(ctx context.Context, vos []*dto.CourseVO) ([]*model.Course, error)
	ToPaginatedCourses(cxt context.Context, courses []*model.Course, total int64, pageParam *dto.PageParam) (*dto.PaginatedCourses, error)
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

// ToCourseVO 单个Course转CourseVO (DB to VO) 包含优化过的tag查询
func (a *CourseAssembler) ToCourseVO(ctx context.Context, c *model.Course) (*dto.CourseVO, error) {
	// 获得相关课程link并转化为VO
	var linkVOs []*dto.CourseInLinkVO
	if c.LinkedCourses != nil {
		for _, linkedCourse := range c.LinkedCourses {
			// 直接使用LinkedCourses中的数据，不需要再查询数据库
			linkVOs = append(linkVOs, &dto.CourseInLinkVO{
				ID:   linkedCourse.ID,
				Name: linkedCourse.Name,
			})
		}
	}

	// 获得课程前三多的tag (使用goroutine异步获取提高性能)
	tagCountChan := make(chan map[string]int, 1)
	go func() {
		tagCount, err := a.CommentRepo.CountTagsByCourseID(ctx, c.ID)
		if err != nil {
			log.CtxError(ctx, "CountTagsByCourseID failed for courseID=%s: %v", c.ID, err)
			tagCountChan <- make(map[string]int)
		} else {
			tagCountChan <- tagCount
		}
	}()

	// 获取校区列表
	var campuses []string
	for _, campusId := range c.Campuses {
		campusName := mapping.Data.GetCampusNameByID(campusId)
		if campusName != "" {
			campuses = append(campuses, campusName)
		}
	}

	// 获得教师VO
	var teachers []*dto.TeacherVO
	for _, tid := range c.TeacherIDs {
		dbTeacher, err := a.TeacherRepo.FindByID(ctx, tid)
		if err != nil {
			log.CtxError(ctx, "Find Teacher Failed, teacherID: %s, error: %v", tid, err)
			continue
		}
		if dbTeacher != nil {
			teachers = append(teachers, &dto.TeacherVO{
				ID:         dbTeacher.ID,
				Name:       dbTeacher.Name,
				Title:      dbTeacher.Title,
				Department: mapping.Data.GetDepartmentNameByID(dbTeacher.Department),
			})
		}
	}

	// 等待tagCount结果
	tagCount := <-tagCountChan

	return &dto.CourseVO{
		ID:         c.ID,
		Name:       c.Name,
		Code:       c.Code,
		Category:   mapping.Data.GetCategoryNameByID(c.Category),
		Campuses:   campuses,
		Department: mapping.Data.GetDepartmentNameByID(c.Department),
		Link:       linkVOs,
		Teachers:   teachers,
		TagCount:   tagCount,
	}, nil
}

// ToCourse 单个CourseVO转Course (VO to DB)
func (a *CourseAssembler) ToCourse(ctx context.Context, vo *dto.CourseVO) (*model.Course, error) {
	if vo == nil {
		return nil, nil
	}

	// 将校区名称转换为ID，使用StaticData的反向映射方法
	var campusIDs []int32
	for _, campusName := range vo.Campuses {
		campusID := mapping.Data.GetCampusIDByName(campusName)
		if campusID != 0 {
			campusIDs = append(campusIDs, campusID)
		}
	}

	// 获取分类ID，使用StaticData的反向映射方法
	categoryID := mapping.Data.GetCategoryIDByName(vo.Category)

	// 获取部门ID，使用StaticData的反向映射方法
	departmentID := mapping.Data.GetDepartmentIDByName(vo.Department)

	// 获取教师ID
	var teacherIDs []string
	for _, teacher := range vo.Teachers {
		teacherIDs = append(teacherIDs, teacher.ID)
	}

	return &model.Course{
		ID:            vo.ID,
		Name:          vo.Name,
		Code:          vo.Code,
		Category:      categoryID,
		Campuses:      campusIDs,
		Department:    departmentID,
		LinkedCourses: nil, // 不再支持相关课程
		TeacherIDs:    teacherIDs,
	}, nil
}

// ToCourseVOList Course数组转CourseVO数组 (DB Array to VO Array)
func (a *CourseAssembler) ToCourseVOList(ctx context.Context, courses []*model.Course) ([]*dto.CourseVO, error) {
	if len(courses) == 0 {
		return []*dto.CourseVO{}, nil
	}

	courseVOs := make([]*dto.CourseVO, len(courses))

	// 使用goroutine并发处理，提高性能
	type result struct {
		index int
		vo    *dto.CourseVO
		err   error
	}

	resultChan := make(chan result, len(courses))
	var wg sync.WaitGroup

	for i, c := range courses {
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
			return nil, r.err
		}
		courseVOs[r.index] = r.vo
	}

	return courseVOs, nil
}

// ToCourseList CourseVO数组转Course数组 (VO Array to DB Array)
func (a *CourseAssembler) ToCourseList(ctx context.Context, vos []*dto.CourseVO) ([]*model.Course, error) {
	if len(vos) == 0 {
		return []*model.Course{}, nil
	}

	courses := make([]*model.Course, 0, len(vos))

	for _, vo := range vos {
		dbCourse, err := a.ToCourse(ctx, vo)
		if err != nil {
			return nil, err
		}
		if dbCourse != nil {
			courses = append(courses, dbCourse)
		}
	}

	return courses, nil
}

// ToPaginatedCourses Course数组转paginatedCourses
func (a *CourseAssembler) ToPaginatedCourses(cxt context.Context, courses []*model.Course, total int64, pageParam *dto.PageParam) (*dto.PaginatedCourses, error) {
	courseVOs, err := a.ToCourseVOList(cxt, courses)
	if err != nil {
		return &dto.PaginatedCourses{}, err
	}

	return &dto.PaginatedCourses{
		Courses:   courseVOs,
		Total:     total,
		PageParam: pageParam,
	}, nil
}
