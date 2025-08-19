package cmd

import "time"

// SearchHistoryVO 是返回给前端的、单条搜索历史的“视图对象”。
// 它对应 OpenAPI 文档中的 SearchHistoryVO。
type SearchHistoryVO struct {
	ID       string    `json:"id"`
	Text     string    `json:"text"`
	CreateAt time.Time `json:"createAt"`
}
