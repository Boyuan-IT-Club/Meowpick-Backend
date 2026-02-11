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

// ListChangeLogReq 变更记录列表查询请求
type ListChangeLogReq struct {
	TargetType string `json:"targetType" binding:"required,one of=course proposal teacher user"`
	TargetID   string `json:"targetId" binding:"required"`
	*PageParam
}

// CreateChangeLogReq 新增变更记录请求
type CreateChangeLogReq struct {
	TargetType   string `json:"targetType" binding:"required,one of=course proposal teacher user"`
	TargetID     string `json:"targetId" binding:"required"`
	Action       string `json:"action" binding:"required,one of=create update delete"`
	Content      string `json:"content" binding:"required"`
	UpdateSource string `json:"updateSource" binding:"required,one of=manual system"`
	ProposalID   string `json:"proposalId,omitempty"`
}

// ChangeLogVO 变更记录视图对象
type ChangeLogVO struct {
	ID           string    `json:"id"`
	TargetID     string    `json:"targetId"`
	TargetType   string    `json:"targetType"`
	Action       string    `json:"action"`
	Content      string    `json:"content"`
	UserID       string    `json:"userId"`
	UpdateSource string    `json:"updateSource"`
	ProposalID   string    `json:"proposalId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ListChangeLogResp 变更记录列表响应
type ListChangeLogResp struct {
	*Resp      `json:",inline"`
	Total      int64          `json:"total"`
	ChangeLogs []*ChangeLogVO `json:"changeLogs"`
}

// CreateChangeLogResp 新增变更记录响应
type CreateChangeLogResp struct {
	*Resp
	ChangeLogID string `json:"changeLogId"`
}
