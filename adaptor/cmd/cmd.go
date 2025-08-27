package cmd

type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type QueryParam struct {
	Page     int
	PageSize int
}

func Success() *Resp {
	return &Resp{
		Code: 0,
		Msg:  "success",
	}
}
