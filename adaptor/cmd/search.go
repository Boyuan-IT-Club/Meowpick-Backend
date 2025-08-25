package cmd

import "time"

// LogSearchReq 是用于我们临时测试接口的数据结构
type LogSearchReq struct {
	Query string `json:"keyword" binding:"required"`
}

// SearchHistoryVO 是返回给前端的、单条搜索历史的“视图对象”。
// 它对应 OpenAPI 文档中的 SearchHistoryVO。
type SearchHistoryVO struct {
	ID       string    `json:"id"`
	Text     string    `json:"text"`
	CreateAt time.Time `json:"createAt"`
}

// GetSearchHistoryResp 是返回给前端的搜索历史列表的响应体。
type GetSearchHistoryResp struct {
	*Resp
	History *[]SearchHistoryVO `json:"history"`
}
