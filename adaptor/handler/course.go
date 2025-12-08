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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// GetOneCourse 精确搜索一个课程，返回课程元信息
// @router /api/course/:courseId [GET]
func GetOneCourse(c *gin.Context) {
	var resp *dto.GetOneCourseResp
	var err error

	resp, err = provider.Get().CourseService.GetOneCourse(c, c.Param("courseId"))
	PostProcess(c, nil, resp, err)
}

// GetCourseDepartments xxx
// @router /api/course/departs [GET]
func GetCourseDepartments(ctx *gin.Context) {
	var req *dto.GetCoursesDepartmentsReq
	var resp *dto.GetCoursesDepartmentsResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		return
	}
	resp, err = provider.Get().CourseService.GetDepartments(ctx, req)
	PostProcess(ctx, req, resp, err)
}

// GetCourseCategories xxx
// @router /api/course/categories [GET]
func GetCourseCategories(ctx *gin.Context) {
	var req *dto.GetCourseCategoriesReq
	var resp *dto.GetCourseCategoriesResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		return
	}
	resp, err = provider.Get().CourseService.GetCategories(ctx, req)
	PostProcess(ctx, req, resp, err)
}

func GetCourseCampuses(ctx *gin.Context) {
	var req *dto.GetCourseCampusesReq
	var resp *dto.GetCourseCampusesResp
	var err error
	if err = ctx.ShouldBindQuery(&req); err != nil {
		return
	}
	resp, err = provider.Get().CourseService.GetCampuses(ctx, req)
	PostProcess(ctx, req, resp, err)
}
