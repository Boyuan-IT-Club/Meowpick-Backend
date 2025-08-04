package cmd

type SearchReq struct {
	Keyword string `json:"keyword" binding:"required"`
	// 搜索类型，由前端指定，如："course", "teacher", "department" 等
	Type string `json:"type" binding:"required"`
}

// SearchResultItem 是一个“通用”的搜索结果条目。因为搜索课程返回的是课程信息，搜索老师返回的是老师信息，它们的结构不同，所以我们用 interface{} 来表示这里可以存放任何类型的数据。
type SearchResultItem interface{}

// SearchResp 是后端返回给前端的、通用的、分页的搜索结果，匹配了 PageEntityObject 的结构
type SearchResp struct {
	// 本次查询总共找到了多少条记录
	Total int64 `json:"total"`
	// 当前页的结果列表，里面存放的是具体的课程信息或者老师信息等
	Row []SearchResultItem `json:"row"`
}
