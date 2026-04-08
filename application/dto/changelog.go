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
	IP           string `json:"ip"`
	UserAgent    string `json:"userAgent"`
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

// ListProposalLogsGroupedReq 按提案聚合的日志列表请求参数
type ListProposalLogsGroupedReq struct {
	*PageParam
}

// ListProposalLogsGroupedResp 按提案聚合的日志列表响应
type ListProposalLogsGroupedResp struct {
	*Resp
	Total     int64              `json:"total"`
	Proposals []*ProposalLogVO   `json:"proposals"`
}

// ProposalLogVO 提案日志展示对象
type ProposalLogVO struct {
	ProposalID  string          `json:"proposalId"`
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	Status      string          `json:"status"`
	Course      *CourseVO       `json:"course"`
	Creator     *CreatorVO      `json:"creator"`
	AdminAction *AdminActionVO  `json:"adminAction,omitempty"`
}

// CreatorVO 创建者信息
type CreatorVO struct {
	CreatorID   string `json:"creatorId"`
	CreatorName string `json:"creatorName"`
	CreateTime  string `json:"createTime"`
}

// AdminActionVO 管理员操作信息
type AdminActionVO struct {
	AdminID    string `json:"adminId"`
	AdminName  string `json:"adminName"`
	Action     string `json:"action"` // approve/reject/delete
	ActionTime string `json:"actionTime"`
	Reason     string `json:"reason,omitempty"`
}

// ListProposalLogsTimelineReq 扁平化时间线日志请求参数
type ListProposalLogsTimelineReq struct {
	*PageParam
}

// ListProposalLogsTimelineResp 扁平化时间线日志响应
type ListProposalLogsTimelineResp struct {
	*Resp
	Total int64                    `json:"total"`
	Logs  []*ProposalTimelineLogVO `json:"logs"`
}

// ProposalTimelineLogVO 提案时间线日志展示对象
type ProposalTimelineLogVO struct {
	LogID            string                 `json:"logId"`
	ProposalID       string                 `json:"proposalId,omitempty"`
	ActionType       string                 `json:"actionType"` // CREATE/APPROVE/REJECT/DELETE/UPDATE/GRANT_ADMIN/REVOKE_ADMIN
	OperatorID       string                 `json:"operatorId"`
	OperatorName     string                 `json:"operatorName"`
	ActionTime       string                 `json:"actionTime"`
	ProposalSnapshot *ProposalSnapshotVO    `json:"proposalSnapshot,omitempty"`
	Details          map[string]interface{} `json:"details,omitempty"`
}

// ProposalSnapshotVO 提案快照信息
type ProposalSnapshotVO struct {
	Title      string `json:"title"`
	CourseName string `json:"courseName,omitempty"`
	Department string `json:"department,omitempty"`
	Category   string `json:"category,omitempty"`
}
