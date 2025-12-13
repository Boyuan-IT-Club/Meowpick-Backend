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
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/gin-gonic/gin"
)

// GetSearchHistories 获得最近15条搜索历史
// @router /api/search/recent [GET]
func GetSearchHistories(c *gin.Context) {
	var err error
	var resp *dto.GetSearchHistoriesResp

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err = provider.Get().SearchHistoryService.GetSearchHistory(c)
	PostProcess(c, nil, resp, err)
}

// GetSearchSuggestions 输入框有文本更新时显示搜索建议
// @router /api/search/suggest
func GetSearchSuggestions(c *gin.Context) {
	var err error
	var req dto.GetSearchSuggestionsReq
	var resp *dto.GetSearchSuggestionsResp

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err = provider.Get().SearchService.GetSearchSuggestions(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListCourses 用户点击“搜索”按钮或点击某一项后展示课程列表
// @router /api/search
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
