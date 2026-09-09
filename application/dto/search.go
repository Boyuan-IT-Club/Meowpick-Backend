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

package dto

import "time"

// GetSearchHistoriesResp 是返回给前端的搜索历史列表的响应体。
type GetSearchHistoriesResp struct {
	*Resp
	Histories []*SearchHistoryVO `json:"histories"` // 当前用户最近15条搜索历史，按最近搜索时间倒序
}

type GetSearchSuggestionsReq struct {
	Keyword string `form:"keyword" binding:"required"` // 同时匹配课程名、教师名、分类名和开课院系名的关键词
	*PageParam
}

type GetSearchSuggestionsResp struct {
	*Resp
	Suggestions []*SearchSuggestionsVO `json:"suggestions"` // 合并后的建议列表，最多 pageSize 条
}

type SearchSuggestionsVO struct {
	Type        string `json:"type"`            // 建议类型：course、teacher、category、department
	Name        string `json:"name"`            // 建议实体名称；教师建议仅为姓名，不包含职称
	Title       string `json:"title,omitempty"` // 仅教师建议返回职称；其他类型省略
	SearchValue string `json:"searchValue"`     // 点击后传给 /api/search 的精确搜索值；教师为无分隔的姓名+职称（职称为空时等于姓名），其他类型等于 name
}

// SearchHistoryVO 是返回给前端的、单条搜索历史的“视图对象”。
// 它对应 OpenAPI 文档中的 SearchHistoryVO。
type SearchHistoryVO struct {
	ID        string    `json:"id"`        // 搜索历史记录ID
	Query     string    `json:"query"`     // 曾提交的非空课程搜索关键词
	CreatedAt time.Time `json:"createdAt"` // 最近一次搜索该关键词的时间
}
