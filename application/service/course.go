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
	ListCourses(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error)

	GetCourse(ctx context.Context, req *dto.GetCourseReq) (*dto.GetCourseResp, error)
	GetDepartments(ctx context.Context, req *dto.GetCourseDepartmentsReq) (*dto.GetCourseDepartmentsResp, error)
	GetCategories(ctx context.Context, req *dto.GetCourseCategoriesReq) (*dto.GetCourseCategoriesResp, error)
	GetCampuses(ctx context.Context, req *dto.GetCourseCampusesReq) (*dto.GetCourseCampusesResp, error)
}

type CourseService struct {
	CourseRepo      *repo.CourseRepo
	TeacherRepo     *repo.TeacherRepo
	CourseAssembler *assembler.CourseAssembler
}

var CourseServiceSet = wire.NewSet(
	wire.Struct(new(CourseService), "*"),
	wire.Bind(new(ICourseService), new(*CourseService)),
)

// ListCourses 返回课程的分页结果
// 当req.Type为"course"时，模糊分页搜索课程
// 当req.Type为"teacher"时，精确分页搜索教师开设的课程
// 当req.Type为"category"时，精确分页搜索该类别下的课程
// 当req.Type为"department"时，精确分页搜索该开课院系下的课程
func (s *CourseService) ListCourses(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 区分不同的搜索方式
	var err error
	var total int64
	var courses []*model.Course
	switch req.Type {
	case consts.ReqCourse:
		courses, total, err = s.CourseRepo.FindManyByNameLike(ctx, req.Keyword, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [FindManyByNameLike] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed, errorx.KV("name", req.Keyword))
		}
	case consts.ReqTeacher:
		tid, err := s.TeacherRepo.GetIDByName(ctx, req.Keyword)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [GetIDByName] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrTeacherFindFailed, errorx.KV("name", req.Keyword))
		}
		courses, total, err = s.CourseRepo.FindManyByTeacherID(ctx, tid, req.PageParam)
	case consts.ReqCategory:
		cid := mapping.Data.GetCategoryIDByName(req.Keyword)
		courses, total, err = s.CourseRepo.FindManyByCategoryID(ctx, cid, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [FindManyByCategoryID] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed,
				errorx.KV("key", consts.ReqCategory), errorx.KV("value", req.Keyword))
		}
	case consts.ReqDepartment:
		did := mapping.Data.GetDepartmentIDByName(req.Keyword)
		courses, total, err = s.CourseRepo.FindManyByDepartmentID(ctx, did, req.PageParam)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseRepo] [FindManyByDepartmentID] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed,
				errorx.KV("key", consts.ReqDepartment), errorx.KV("value", req.Keyword))
		}
	default:
		logs.CtxErrorf(ctx, "[CourseService] [ListCourses] invalid type: %s", req.Type)
		return nil, errorx.New(
			errno.ErrCourseInvalidParam,
			errorx.KV("key", consts.ReqType),
			errorx.KV("value", req.Type),
		)
	}

	// 转换为分页结果
	pcs, err := s.CourseAssembler.ToPaginatedCourses(ctx, courses, total, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToPaginatedCourses] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "database coursers"), errorx.KV("dst", "paginated courses"),
		)
	}

	return &dto.ListCoursesResp{
		Resp:             dto.Success(),
		PaginatedCourses: pcs,
	}, nil
}

// GetCourse 精确搜索一个课程，返回课程元信息
func (s *CourseService) GetCourse(ctx context.Context, req *dto.GetCourseReq) (*dto.GetCourseResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 搜索课程
	course, err := s.CourseRepo.FindByID(ctx, req.CourseID)
	if err != nil || course == nil { // 使用id搜索不应出现找不到的情况
		return nil, errorx.WrapByCode(err, errno.ErrCourseFindFailed,
			errorx.KV("key", consts.CourseID), errorx.KV("value", req.CourseID))
	}

	// 转换为VO
	vo, err := s.CourseAssembler.ToCourseVO(ctx, course)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseVO] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCourseCvtFailed,
			errorx.KV("src", "database course"), errorx.KV("dst", "course vo"))
	}

	return &dto.GetCourseResp{
		Resp:   dto.Success(),
		Course: vo,
	}, nil
}

// GetDepartments 根据课程名称查询开课院系
func (s *CourseService) GetDepartments(ctx context.Context, req *dto.GetCourseDepartmentsReq) (*dto.GetCourseDepartmentsResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 搜索院系
	ids, err := s.CourseRepo.GetDepartmentsByName(ctx, req.Keyword)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [GetDepartmentsByName] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCourseGetDepartmentsFailed, errorx.KV("name", req.Keyword))
	}

	// 转换为院系名称列表
	departments := []string{}
	for _, id := range ids {
		departments = append(departments, mapping.Data.GetDepartmentNameByID(id))
	}

	return &dto.GetCourseDepartmentsResp{
		Resp:        dto.Success(),
		Departments: departments,
	}, nil
}

// GetCategories 根据课程名称查询课程分类
func (s *CourseService) GetCategories(ctx context.Context, req *dto.GetCourseCategoriesReq) (*dto.GetCourseCategoriesResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	ids, err := s.CourseRepo.GetCategoriesByName(ctx, req.Keyword)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [GetCategoriesByName] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCourseGetCategoriesFailed, errorx.KV("name", req.Keyword))
	}

	// 转换为课程分类名称列表
	categories := make([]string, 0, len(ids))
	for _, id := range ids {
		categories = append(categories, mapping.Data.GetCategoryNameByID(id))
	}

	return &dto.GetCourseCategoriesResp{
		Resp:       dto.Success(),
		Categories: categories,
	}, nil
}

// GetCampuses 根据课程名称查询开课校区
func (s *CourseService) GetCampuses(ctx context.Context, req *dto.GetCourseCampusesReq) (*dto.GetCourseCampusesResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	ids, err := s.CourseRepo.GetCampusesByName(ctx, req.Keyword)
	if err != nil {
		logs.CtxErrorf(ctx, "[CourseRepo] [GetCampusesByName] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrCourseGetCampusesFailed, errorx.KV("name", req.Keyword))
	}

	// 转换为校区名称列表
	campuses := make([]string, 0, len(ids))
	for _, id := range ids {
		campuses = append(campuses, mapping.Data.GetCategoryNameByID(id))
	}

	return &dto.GetCourseCampusesResp{
		Resp:     dto.Success(),
		Campuses: campuses,
	}, nil
}
