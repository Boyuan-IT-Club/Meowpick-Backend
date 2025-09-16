package cmd

import "time"

// LogSearchReq 是用于我们临时测试接口的数据结构
type LogSearchReq struct {
	Query string `json:"keyword" binding:"required"`
}

// SearchHistoryVO 是返回给前端的、单条搜索历史的“视图对象”。
// 它对应 OpenAPI 文档中的 SearchHistoryVO。
type SearchHistoryVO struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetSearchHistoryResp 是返回给前端的搜索历史列表的响应体。
type GetSearchHistoryResp struct {
	*Resp
	History []*SearchHistoryVO `json:"history"`
}

type SearchSuggestionsVO struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type GetSearchSuggestReq struct {
	Keyword  string `form:"keyword" binding:"required"`
	Page     int64  `form:"page,default=1" json:"page"`
	PageSize int64  `form:"pageSize,default=10" json:"pageSize"`
}

type GetSearchSuggestResp struct {
	*Resp
	List []*SearchSuggestionsVO `json:"list"`
}
