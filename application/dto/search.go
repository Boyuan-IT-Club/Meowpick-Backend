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
	Histories []*SearchHistoryVO `json:"histories"`
}

type GetSearchSuggestionsReq struct {
	Keyword string `form:"keyword" binding:"required"`
	*PageParam
}

type GetSearchSuggestionsResp struct {
	*Resp
	Suggestions []*SearchSuggestionsVO `json:"suggestions"`
}

type SearchSuggestionsVO struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// SearchHistoryVO 是返回给前端的、单条搜索历史的“视图对象”。
// 它对应 OpenAPI 文档中的 SearchHistoryVO。
type SearchHistoryVO struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"createdAt"`
}
