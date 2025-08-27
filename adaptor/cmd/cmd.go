package cmd

type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func Success() *Resp {
	return &Resp{
		Code: 0,
		Msg:  "success",
	}
}

type IPageParam interface {
	UnWrap() (int64, int64)
	GetPage() int64
	GetPageSize() int64
	SetPage(size int64)
	SetPageSize(size int64)
}

type PageParam struct {
	Page     int64
	PageSize int64
}

func (qp *PageParam) UnWrap() (int64, int64) {
	if qp.Page <= 0 {
		qp.Page = 1
	}
	if qp.PageSize <= 0 || qp.PageSize > 100 {
		qp.PageSize = 10
	}
	return qp.Page, qp.PageSize
}

func (qp *PageParam) GetPage() int64 {
	return qp.Page
}

func (qp *PageParam) GetPageSize() int64 {
	return qp.PageSize
}

func (qp *PageParam) SetPage(page int64) {
	qp.Page = page
}

func (qp *PageParam) SetPageSize(pageSize int64) {
	qp.PageSize = pageSize
}
