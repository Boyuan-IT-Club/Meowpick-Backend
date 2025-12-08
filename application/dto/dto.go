package dto

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
	SetPage(page int64)
	SetPageSize(pageSize int64)
}

type PageParam struct {
	Page     int64 `form:"page" json:"page"`
	PageSize int64 `form:"pageSize" json:"pageSize"`
}

func (p *PageParam) UnWrap() (int64, int64) {
	if p.Page < 0 {
		p.Page = 0
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 10
	}

	return p.Page, p.PageSize
}

func (p *PageParam) GetPage() int64 {
	return p.Page
}

func (p *PageParam) GetPageSize() int64 {
	return p.PageSize
}

func (p *PageParam) SetPage(page int64) {
	p.Page = page
}

func (p *PageParam) SetPageSize(pageSize int64) {
	p.PageSize = pageSize
}
