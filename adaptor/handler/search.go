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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// GetSearchHistory 获得最近15条搜索历史
// @router /api/search/recent [GET]
func GetSearchHistory(c *gin.Context) {
	var err error
	var resp *dto.GetSearchHistoriesResp

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().SearchHistoryService.GetSearchHistoryByUserId(c)
	PostProcess(c, nil, resp, err)
}

// GetSearchSuggestions 输入框有文本更新时 显示搜索建议
// @router /api/search/suggest
func GetSearchSuggestions(c *gin.Context) {
	var err error
	var req dto.GetSearchSuggestReq
	var resp *dto.GetSearchSuggestResp
	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, req, nil, err)
		return
	}
	resp, err = provider.Get().SearchService.GetSearchSuggestions(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListCourses 用户点击🔍时，若req里type为"course"，模糊搜索课程，返回课程VO列表
// 若req里type为"teacher"，精确搜索教师开设的课程VO列表
// @router /api/search
func ListCourses(c *gin.Context) {
	var req dto.ListCoursesReq
	var resp *dto.ListCoursesResp
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		// 如果这里出错，err 就被赋值了。我们直接 return，
		// defer 会自动捕获这个 err 并处理错误响应。
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))

	if req.Keyword != "" {
		keyword := req.Keyword
		// 使用 gin.Context 的副本，安全传入 goroutine
		cCopy := c.Copy()
		go func() {
			if err = provider.Get().SearchHistoryService.LogSearch(cCopy, keyword); err != nil {
				log.CtxError(cCopy, "记录搜索历史失败: %v", err)
			}
		}()
	}

	if req.Type == consts.Course {
		resp, err = provider.Get().CourseService.ListCourses(c, &req)
	} else if req.Type == consts.Teacher {
		resp, err = provider.Get().TeacherService.ListCoursesByTeacher(c, &req)
	} else {
		resp, err = provider.Get().SearchService.ListCoursesByType(c, &req) // 根据req中的Type字段，根据Category或department查询课程
	}

	PostProcess(c, &req, resp, err)
}

// ListTeachers 用户点击🔍时模糊搜索，返回教师VO列表
// @router /api/search/teacher
func ListTeachers(c *gin.Context) {
	// TODO 实现接口

}
