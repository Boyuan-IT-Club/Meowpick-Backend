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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// GetCourse 精确搜索一个课程，返回课程元信息
// @router /api/course/:courseId [GET]
func GetCourse(c *gin.Context) {
	var resp *dto.GetCourse
	var err error

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().CourseService.GetOneCourse(c, c.Param("courseId"))
	PostProcess(c, nil, resp, err)
}

// GetCourseDepartments .
// @router /api/course/departs [GET]
func GetCourseDepartments(c *gin.Context) {
	var req dto.GetCourseDepartmentsReq
	var resp *dto.GetCourseDepartmentsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().CourseService.GetDepartments(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseCategories .
// @router /api/course/categories [GET]
func GetCourseCategories(c *gin.Context) {
	var req dto.GetCourseCategoriesReq
	var resp *dto.GetCourseCategoriesResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().CourseService.GetCategories(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseCampuses .
// @router /api/course/campuses [GET]
func GetCourseCampuses(c *gin.Context) {
	var req dto.GetCourseCampusesReq
	var resp *dto.GetCourseCampusesResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().CourseService.GetCampuses(c, &req)
	PostProcess(c, req, resp, err)
}
