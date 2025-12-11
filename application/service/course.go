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

package service

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ ICourseService = (*CourseService)(nil)

type ICourseService interface {
	GetOneCourse(ctx context.Context, courseID string) (*dto.GetCourse, error)
	ListCourses(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error)
	GetDepartments(ctx context.Context, req *dto.GetCourseDepartmentsReq) (*dto.GetCourseDepartmentsResp, error)
	GetCategories(ctx context.Context, req *dto.GetCourseCategoriesReq) (*dto.GetCourseCategoriesResp, error)
	GetCampuses(ctx context.Context, req *dto.GetCourseCampusesReq) (*dto.GetCourseCampusesResp, error)
}

type CourseService struct {
	CourseRepo      *repo.CourseRepo
	CommentRepo     *repo.CommentRepo
	TeacherRepo     *repo.TeacherRepo
	CourseAssembler *assembler.CourseAssembler
}

var CourseServiceSet = wire.NewSet(
	wire.Struct(new(CourseService), "*"),
	wire.Bind(new(ICourseService), new(*CourseService)),
)

// GetOneCourse 精确搜索，返回课程的元信息CourseVO
func (s *CourseService) GetOneCourse(ctx context.Context, courseID string) (*dto.GetCourse, error) {
	dbCourse, err := s.CourseRepo.FindByID(ctx, courseID)
	if err != nil || dbCourse == nil { // 使用id搜索不应出现找不到的情况
		return nil, err
	}

	courseVO, err := s.CourseAssembler.ToCourseVO(ctx, dbCourse)
	if err != nil {
		log.CtxError(ctx, "CourseDB To CourseVO error: %v", err)
		return nil, errorx.ErrCourseDB2VO
	}

	return &dto.GetCourse{
		Resp:   dto.Success(),
		Course: courseVO,
	}, nil
}

// ListCourses 返回课程的分页结果
// 当req.Type为"course"时，模糊分页搜索课程
// 当req.Type为"teacher"时，精确分页搜索教师开设的课程
// 当req.Type为"category"时，精确分页搜索该类别下的课程
// 当req.Type为"department"时，精确分页搜索该开课院系下的课程
func (s *CourseService) ListCourses(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.ContextUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 区分不同的搜索方式
	var err error
	var total int64
	var dbs []*model.Course
	switch req.Type {
	case consts.ReqCourse:
		dbs, total, err = s.CourseRepo.FindManyByNameLike(ctx, req.Keyword, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "CourseRepo FindManyByNameLike error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed)
		}
	case consts.ReqTeacher:
		tid, err := s.TeacherRepo.FindIDByName(ctx, req.Keyword)
		if err != nil {
			logs.CtxErrorf(ctx, "CourseRepo FindIDByName error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrTeacherIDNotFound, errorx.KV("name", req.Keyword))
		}
		dbs, total, err = s.CourseRepo.FindManyByTeacherID(ctx, tid, req.PageParam)
	case consts.ReqCategory:
		cid := mapping.Data.GetCategoryIDByName(req.Keyword)
		dbs, total, err = s.CourseRepo.FindManyByCategoryID(ctx, cid, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "CourseRepo FindManyByCategoryID error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed)
		}
	case consts.ReqDepartment:
		did := mapping.Data.GetDepartmentIDByName(req.Keyword)
		dbs, total, err = s.CourseRepo.FindManyByDepartmentID(ctx, did, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "CourseRepo FindManyByDepartmentID error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed)
		}
	default:
		logs.CtxErrorf(ctx, "CourseService ListCourses error: invalid type %s", req.Type)
		return nil, errorx.New(
			errno.ErrCourseInvalidParam,
			errorx.KV("key", "type"),
			errorx.KV("value", req.Type),
		)
	}

	// 转换为分页结果
	pcs, err := s.CourseAssembler.ToPaginatedCourses(ctx, dbs, total, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "CourseAssembler ToPaginatedCourses error: %v", err)
		return nil, errorx.WrapByCode(
			err,
			errno.ErrCourseCvtFailed,
			errorx.KV("src", "dbs"), errorx.KV("dst", "pcs"),
		)
	}

	return &dto.ListCoursesResp{
		Resp:             dto.Success(),
		PaginatedCourses: pcs,
	}, nil
}

func (s *CourseService) GetDepartments(ctx context.Context, req *dto.GetCourseDepartmentsReq) (*dto.GetCourseDepartmentsResp, error) {
	departsIDs, err := s.CourseRepo.GetDepartmentsByName(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	departs := make([]string, 0, len(departsIDs))
	for _, dbDepart := range departsIDs {
		departs = append(departs, mapping.Data.GetDepartmentNameByID(dbDepart))
	}

	response := &dto.GetCourseDepartmentsResp{
		Resp:        dto.Success(),
		Departments: departs,
	}

	return response, nil
}

func (s *CourseService) GetCategories(ctx context.Context, req *dto.GetCourseCategoriesReq) (*dto.GetCourseCategoriesResp, error) {

	categoriesIDs, err := s.CourseRepo.GetCategories(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	categories := make([]string, 0, len(categoriesIDs))
	for _, dbCategory := range categoriesIDs {
		categories = append(categories, mapping.Data.GetCategoryNameByID(dbCategory))
	}

	response := &dto.GetCourseCategoriesResp{
		Resp:       dto.Success(),
		Categories: categories,
	}
	return response, nil
}

func (s *CourseService) GetCampuses(ctx context.Context, req *dto.GetCourseCampusesReq) (*dto.GetCourseCampusesResp, error) {

	campusesIDs, err := s.CourseRepo.GetCampusesByName(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	campuses := make([]string, 0, len(campusesIDs))
	for _, dbCampus := range campusesIDs {
		campuses = append(campuses, mapping.Data.GetCategoryNameByID(dbCampus))
	}

	response := &dto.GetCourseCampusesResp{
		Resp:     dto.Success(),
		Campuses: campuses,
	}
	return response, nil
}
