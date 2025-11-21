package dto

import "time"

// SearchHistoryVO 是返回给前端的、单条搜索历史的“视图对象”。
// 它对应 OpenAPI 文档中的 SearchHistoryVO。
type SearchHistoryVO struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetSearchHistoriesResp 是返回给前端的搜索历史列表的响应体。
type GetSearchHistoriesResp struct {
	*Resp
	Histories []*SearchHistoryVO `json:"histories"`
}

type SearchSuggestionsVO struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type GetSearchSuggestReq struct {
	Keyword string `form:"keyword" binding:"required"`
	*PageParam
}

type GetSearchSuggestResp struct {
	*Resp
	Suggestions []*SearchSuggestionsVO `json:"suggestions"`
}
