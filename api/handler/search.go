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
	"github.com/gin-gonic/gin"
)

// GetSearchHistories godoc
// @Summary 获取最近搜索历史
// @Description 登录后返回当前用户最近15条课程搜索历史，按最近搜索时间倒序。相同关键词会复用原记录并更新时间；搜索建议不会写入历史，只有提交非空课程搜索时才异步记录
// @Tags search
// @Produce json
// @Success 200 {object} Response[dto.GetSearchHistoriesResp]
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
// @Description 登录后并行匹配未删除课程名称，以及至少被一门未删除课程引用的教师、课程分类和课程开课院系；按 course、teacher、category、department 的顺序合并，最终最多返回 pageSize 条。任一数据库建议查询失败时整个请求失败。此接口只返回建议，不写入搜索历史
// @Tags search
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "最终建议条数上限，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.GetSearchSuggestionsResp]
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
