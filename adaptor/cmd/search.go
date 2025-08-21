package cmd

// LogSearchReq 是用于我们临时测试接口的数据结构
type LogSearchReq struct {
	Query string `json:"keyword" binding:"required"`
}

type LogSearchResp struct {
	Code int    `json:"-"`
	Msg  string `json:"-"`
}
