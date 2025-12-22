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

// CreateChangeLogReq 新增变更日志请求参数
type CreateChangeLogReq struct {
	TargetID     string `json:"targetId" binding:"required"`
	TargetType   int32  `json:"targetType" binding:"required"`
	Action       int32  `json:"action" binding:"required"`
	Content      string `json:"content" binding:"required"`
	UpdateSource int32  `json:"updateSource" binding:"required"`
	ProposalID   string `json:"proposalId"`
}

// CreateChangeLogResp 新增变更日志响应
type CreateChangeLogResp struct {
	*Resp
	ChangeLogID string `json:"changeLogId"` // 变更日志ID
}

type ChangeLogVO struct {
	ID           string    `json:"id"`
	TargetID     string    `json:"targetId"`
	TargetType   int32     `json:"targetType"`
	Action       int32     `json:"action"`
	Content      string    `json:"content"`
	UpdateSource int32     `json:"updateSource"`
	ProposalID   string    `json:"proposalId,omitempty"`
	UserID       string    `json:"userId"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
