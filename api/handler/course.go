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

package handler

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/gin-gonic/gin"
)

// GetCourse godoc
// @Summary 获取课程信息
// @Description 登录后按课程ID获取未被软删除的正式课程，包含课程分类、开课院系、校区、教师、评论标签统计，以及可选的来源提案贡献者信息。课程由匿名提案创建时 contributor 仅返回 proposalId 和 showUsername=false，不返回用户身份
// @Tags course
// @Produce json
// @Param courseId path string true "课程ID"
// @Success 200 {object} Response[dto.GetCourseResp]
// @Router /api/course/{courseId} [get]
func GetCourse(c *gin.Context) {
	var req dto.GetCourseReq
	var resp *dto.GetCourseResp
	var err error

	req.CourseID = c.Param(consts.CtxCourseID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetCourse(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseDepartments godoc
// @Summary 获取课程开课院系
// @Description 登录后按完整课程名称精确匹配所有未删除同名课程，返回这些课程涉及的去重开课院系名称；没有同名课程时返回空数组，历史映射缺失时可能返回“未知开课院系”
// @Tags course
// @Produce json
// @Param keyword query string true "课程名称关键词"
// @Success 200 {object} Response[dto.GetCourseDepartmentsResp]
// @Router /api/course/departments [get]
func GetCourseDepartments(c *gin.Context) {
	var req dto.GetCourseDepartmentsReq
	var resp *dto.GetCourseDepartmentsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetDepartments(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseCategories godoc
// @Summary 获取课程分类
// @Description 登录后按完整课程名称精确匹配所有未删除同名课程，返回这些课程涉及的去重课程分类名称；没有同名课程时返回空数组，历史映射缺失时可能返回“未知分类”
// @Tags course
// @Produce json
// @Param keyword query string true "课程名称关键词"
// @Success 200 {object} Response[dto.GetCourseCategoriesResp]
// @Router /api/course/categories [get]
func GetCourseCategories(c *gin.Context) {
	var req dto.GetCourseCategoriesReq
	var resp *dto.GetCourseCategoriesResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetCategories(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseCampuses godoc
// @Summary 获取课程开课校区
// @Description 登录后按完整课程名称精确匹配所有未删除同名课程，返回这些课程涉及的去重开设校区名称；没有同名课程时返回空数组，历史映射缺失时可能返回“未知校区”
// @Tags course
// @Produce json
// @Param keyword query string true "课程名称关键词"
// @Success 200 {object} Response[dto.GetCourseCampusesResp]
// @Router /api/course/campuses [get]
func GetCourseCampuses(c *gin.Context) {
	var req dto.GetCourseCampusesReq
	var resp *dto.GetCourseCampusesResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetCampuses(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListCourses godoc
// @Summary 搜索课程列表
// @Description 登录后分页搜索未删除课程。type=course 时按课程名称大小写不敏感模糊匹配；type=teacher 时 keyword 精确接受教师姓名或无分隔的“姓名+职称”，姓名命中返回所有同名教师课程的并集，姓名+职称命中返回所有同名同职称教师课程的并集；不接受姓名与职称间的空格或连字符。type=category 或 department 时按映射完整名称精确匹配。未知 type 返回参数错误。非空 keyword 会异步写入当前用户搜索历史，重复关键词更新时间且仅保留最近15条
// @Tags course
// @Accept json
// @Produce json
// @Param body body dto.ListCoursesReq true "搜索类型、关键词和分页；type 支持 course/teacher/category/department"
// @Success 200 {object} Response[dto.ListCoursesResp]
// @Router /api/search [post]
func ListCourses(c *gin.Context) {
	var req dto.ListCoursesReq
	var resp *dto.ListCoursesResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	if req.Keyword != "" {
		go func() {
			cCopy := c.Copy()
			if errCopy := provider.Get().SearchHistoryService.LogSearch(cCopy, req.Keyword); errCopy != nil {
				logs.CtxErrorf(cCopy, "[SearchHistoryService] [LogSearch] error: %v", errCopy)
			}
		}()
	}

	resp, err = provider.Get().CourseService.ListCourses(c, &req)
	PostProcess(c, &req, resp, err)
}
