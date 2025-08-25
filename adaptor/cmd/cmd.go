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
