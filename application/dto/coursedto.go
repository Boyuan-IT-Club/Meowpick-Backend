package dto

import (
	"context"
	"sync"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

type ICourseDTO interface {
	// ToCourseVO 单个Course转CourseVO (DB to VO) 包含优化过的tag查询
	ToCourseVO(ctx context.Context, c *course.Course) (*cmd.CourseVO, error)
	// ToCourse 单个CourseVO转Course (VO to DB)
	ToCourse(ctx context.Context, vo *cmd.CourseVO) (*course.Course, error)
	// ToCourseVOList Course数组转CourseVO数组 (DB Array to VO Array)
	ToCourseVOList(ctx context.Context, courses []*course.Course) ([]*cmd.CourseVO, error)
	// ToCourseList CourseVO数组转Course数组 (VO Array to DB Array)
	ToCourseList(ctx context.Context, vos []*cmd.CourseVO) ([]*course.Course, error)
	// ToPaginatedCourses Course数组转paginatedCourses
	ToPaginatedCourses(cxt context.Context, courses []*course.Course, total int64, pageParam *cmd.PageParam) (*cmd.PaginatedCourses, error)
}

type CourseDTO struct {
	CommentMapper *comment.MongoMapper
	TeacherMapper *teacher.MongoMapper
	CourseMapper  *course.MongoMapper
	StaticData    *consts.StaticData
}

var CourseDTOSet = wire.NewSet(
	wire.Struct(new(CourseDTO), "*"),
	wire.Bind(new(ICourseDTO), new(*CourseDTO)),
)

// 单个Course转CourseVO (DB to VO) 包含优化过的tag查询
func (d *CourseDTO) ToCourseVO(ctx context.Context, c *course.Course) (*cmd.CourseVO, error) {
	// 获得相关课程link并转化为VO
	var linkVOs []*cmd.CourseInLinkVO
	if c.LinkedCourses != nil {
		for _, linkedCourse := range c.LinkedCourses {
			// 直接使用LinkedCourses中的数据，不需要再查询数据库
			linkVOs = append(linkVOs, &cmd.CourseInLinkVO{
				ID:   linkedCourse.ID,
				Name: linkedCourse.Name,
			})
		}
	}

	// 获得课程前三多的tag (使用goroutine异步获取提高性能)
	tagCountChan := make(chan map[string]int, 1)
	go func() {
		tagCount, err := d.CommentMapper.CountCourseTag(ctx, c.ID)
		if err != nil {
			log.CtxError(ctx, "CountCourseTag failed for courseID=%s: %v", c.ID, err)
			tagCountChan <- make(map[string]int)
		} else {
			tagCountChan <- tagCount
		}
	}()

	// 获取校区列表
	var campuses []string
	for _, campusID := range c.Campuses {
		campusName := d.StaticData.GetCampusNameByID(campusID)
		if campusName != "" {
			campuses = append(campuses, campusName)
		}
	}

	// 获得教师VO
	var teachers []*cmd.TeacherVO
	for _, tid := range c.TeacherIDs {
		dbTeacher, err := d.TeacherMapper.FindOneTeacherByID(ctx, tid)
		if err != nil {
			log.CtxError(ctx, "Find Teacher Failed, teacherID: %s, error: %v", tid, err)
			continue
		}
		if dbTeacher != nil {
			teachers = append(teachers, &cmd.TeacherVO{
				ID:         dbTeacher.ID,
				Name:       dbTeacher.Name,
				Title:      dbTeacher.Title,
				Department: d.StaticData.GetDepartmentNameByID(dbTeacher.Department),
			})
		}
	}

	// 等待tagCount结果
	tagCount := <-tagCountChan

	return &cmd.CourseVO{
		ID:         c.ID,
		Name:       c.Name,
		Code:       c.Code,
		Category:   d.StaticData.GetCategoryNameByID(c.Category),
		Campuses:   campuses,
		Department: d.StaticData.GetDepartmentNameByID(c.Department),
		Link:       linkVOs,
		Teachers:   teachers,
		TagCount:   tagCount,
	}, nil
}

// 单个CourseVO转Course (VO to DB)
func (d *CourseDTO) ToCourse(ctx context.Context, vo *cmd.CourseVO) (*course.Course, error) {
	if vo == nil {
		return nil, nil
	}

	// 将校区名称转换为ID，使用StaticData的反向映射方法
	var campusIDs []int32
	for _, campusName := range vo.Campuses {
		campusID := d.StaticData.GetCampusIDByName(campusName)
		if campusID != 0 {
			campusIDs = append(campusIDs, campusID)
		}
	}

	// 获取分类ID，使用StaticData的反向映射方法
	categoryID := d.StaticData.GetCategoryIDByName(vo.Category)

	// 获取部门ID，使用StaticData的反向映射方法
	departmentID := d.StaticData.GetDepartmentIDByName(vo.Department)

	// 获取教师ID
	var teacherIDs []string
	for _, teacher := range vo.Teachers {
		teacherIDs = append(teacherIDs, teacher.ID)
	}

	return &course.Course{
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

// Course数组转CourseVO数组 (DB Array to VO Array)
func (d *CourseDTO) ToCourseVOList(ctx context.Context, courses []*course.Course) ([]*cmd.CourseVO, error) {
	if len(courses) == 0 {
		return []*cmd.CourseVO{}, nil
	}

	courseVOs := make([]*cmd.CourseVO, len(courses))

	// 使用goroutine并发处理，提高性能
	type result struct {
		index int
		vo    *cmd.CourseVO
		err   error
	}

	resultChan := make(chan result, len(courses))
	var wg sync.WaitGroup

	for i, c := range courses {
		wg.Add(1)
		go func(index int, dbCourse *course.Course) {
			defer wg.Done()
			vo, err := d.ToCourseVO(ctx, dbCourse)
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

// CourseVO数组转Course数组 (VO Array to DB Array)
func (d *CourseDTO) ToCourseList(ctx context.Context, vos []*cmd.CourseVO) ([]*course.Course, error) {
	if len(vos) == 0 {
		return []*course.Course{}, nil
	}

	courses := make([]*course.Course, 0, len(vos))

	for _, vo := range vos {
		dbCourse, err := d.ToCourse(ctx, vo)
		if err != nil {
			return nil, err
		}
		if dbCourse != nil {
			courses = append(courses, dbCourse)
		}
	}

	return courses, nil
}

func (d *CourseDTO) ToPaginatedCourses(cxt context.Context, courses []*course.Course, total int64, pageParam *cmd.PageParam) (*cmd.PaginatedCourses, error) {
	courseVOs, err := d.ToCourseVOList(cxt, courses)
	if err != nil {
		return &cmd.PaginatedCourses{}, err
	}

	return &cmd.PaginatedCourses{
		Courses:   courseVOs,
		Total:     total,
		PageParam: pageParam,
	}, nil
}
