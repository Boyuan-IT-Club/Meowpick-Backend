package cmd

type CreateLikeReq struct {
	TargetID string `json:"targetID"`
}

type GetLikeStatusReq struct {
	Target string `json:"targetID" binding:"required"`
	Uid    string `json:"userID" binding:"required"`
}

type LikeResp struct {
	*Resp
	Like    bool  `json:"like"`
	LikeCnt int64 `json:"like_cnt"` // 本地乐观更新需要这次点赞前，该评论
}
