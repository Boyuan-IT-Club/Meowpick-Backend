// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dto

import "time"

// CreateProposalReq 新增投票请求参数
type CreateProposalReq struct {
	Title   string    `json:"title" binding:"required"`
	Content string    `json:"content" binding:"required"`
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
	*LikeVO
	Course    *CourseVO `json:"course"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ListProposalReq 对应 /api/proposal/list 的请求体（分页）
type ListProposalReq struct {
	Status string `json:"status"` // pending / approved / rejected / 空则为全部
	*PageParam
}

type ToggleProposalReq struct {
	ProposalID string `json:"proposalID"`
}

type GetProposalReq struct {
	ProposalID string `json:"proposalId"`
}

// ListProposalResp 对应 /api/proposal/list 的响应体
type ListProposalResp struct {
	*Resp
	Total     int64         `json:"total"`
	Proposals []*ProposalVO `json:"proposals"`
}

type ToggleProposalResp struct {
	Proposal    bool  `json:"proposal"`
	ProposalCnt int64 `json:"proposalCnt"`
	*Resp
}

type GetProposalResp struct {
	*Resp
	Proposal *ProposalVO `json:"proposal"`
}
