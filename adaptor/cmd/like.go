package cmd

type CreateLikeReq struct {
	TargetID string `json:"targetID"`
}

type GetLikeStatusReq struct {
	Target string `json:"targetID" binding:"required"`
	Uid    string `json:"userID" binding:"required"`
}

type LikeVO struct {
	Like    bool  `json:"like"`
	LikeCnt int64 `json:"like_cnt"`
}
type LikeResp struct {
	*LikeVO
	*Resp
}
