package cmd

type CreateLikeReq struct {
	TargetID string `json:"targetID"`
}

type LikeVO struct {
	Like    bool  `json:"like"`
	LikeCnt int64 `json:"likeCnt"`
}

type LikeResp struct {
	*LikeVO
	*Resp
}
