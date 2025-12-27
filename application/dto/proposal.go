package dto

import "time"

// CreateProposalReq 新增投票请求参数
type CreateProposalReq struct {
	Title   string    `json:"title" binding:"required"`
	Content string    `json:"content" binding:"required"`
	Status  string    `json:"status" binding:"required"`
	Course  *CourseVO `json:"course" binding:"required"`
}

// CreateProposalResp 新增投票响应
type CreateProposalResp struct {
	*Resp
	ProposalID string `json:"proposalId"` // 提案ID
}

type ProposalVO struct {
	ID      string `json:"id"`
	UserID  string `json:"userId"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Status  string `json:"status"` // pending / approved / rejected
	Deleted bool   `json:"deleted"`

	Course *CourseVO `json:"course"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// type XxxProposalReq struct { }

// type XxxProposalResp struct { }
