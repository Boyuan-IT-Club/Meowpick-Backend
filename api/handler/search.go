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

// GetSearchHistories godoc
// @Summary 获取最近搜索历史
// @Description 获取最近搜索历史
// @Tags search
// @Produce json
// @Success 200 {object} dto.GetSearchHistoriesResp
// @Router /api/search/recent [get]
func GetSearchHistories(c *gin.Context) {
	var err error
	var resp *dto.GetSearchHistoriesResp

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err = provider.Get().SearchHistoryService.GetSearchHistory(c)
	PostProcess(c, nil, resp, err)
}

// GetSearchSuggestions godoc
// @Summary 获取搜索建议
// @Description 根据关键词获取搜索建议
// @Description 根据 type 不同执行不同的搜索逻辑：
// @Description - course：模糊分页搜索课程
// @Description - teacher：精确分页搜索教师开设的课程
// @Description - category：精确分页搜索该类别下的课程
// @Description - department：精确分页搜索该开课院系下的课程
// @Tags search
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Success 200 {object} dto.GetSearchSuggestionsResp
// @Router /api/search/suggest [get]
func GetSearchSuggestions(c *gin.Context) {
	var err error
	var req dto.GetSearchSuggestionsReq
	var resp *dto.GetSearchSuggestionsResp

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err = provider.Get().SearchService.GetSearchSuggestions(c, &req)
	PostProcess(c, &req, resp, err)
}
