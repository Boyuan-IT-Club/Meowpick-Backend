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

// ListChangeLogsReq 变更记录列表查询请求
type ListChangeLogsReq struct {
	Type    string `json:"type" binding:"omitempty,oneof=course proposal teacher user"` // 可选目标类型：course、proposal、teacher、user；为空时不过滤类型
	Keyword string `json:"keyword"`                                                     // 可选日志内容关键词，大小写不敏感模糊匹配
	*PageParam
}

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
	ID           string    `json:"id"`                   // 变更日志ID
	TargetID     string    `json:"targetId"`             // 被操作业务对象ID
	TargetType   int32     `json:"targetType"`           // 目标类型内部编号
	Action       int32     `json:"action"`               // 操作类型内部编号
	Content      string    `json:"content"`              // 操作说明或拒绝理由等日志正文
	UpdateSource int32     `json:"updateSource"`         // 操作来源内部编号，例如用户或管理员
	ProposalID   string    `json:"proposalId,omitempty"` // 与提案相关时返回提案ID，否则省略
	UserID       string    `json:"userId"`               // 操作者用户ID
	UpdatedAt    time.Time `json:"updatedAt"`            // 操作记录时间
}

// ListProposalLogsGroupedReq 按提案聚合的日志列表请求参数
type ListProposalLogsGroupedReq struct {
	*PageParam
}

// ListChangeLogsResp 变更记录列表响应
type ListChangeLogsResp struct {
	*Resp      `json:",inline"`
	Total      int64          `json:"total"`      // 符合筛选条件的日志总数
	ChangeLogs []*ChangeLogVO `json:"changeLogs"` // 当前分页日志，按时间倒序
}

// ListProposalLogsGroupedResp 按提案聚合的日志列表响应
type ListProposalLogsGroupedResp struct {
	*Resp
	Total     int64            `json:"total"`     // 提案总数
	Proposals []*ProposalLogVO `json:"proposals"` // 当前分页提案及其最近一次管理员操作摘要
}

// ProposalLogVO 提案日志展示对象
type ProposalLogVO struct {
	ProposalID  string            `json:"proposalId"`            // 提案ID
	Title       string            `json:"title"`                 // 提案标题
	Content     string            `json:"content"`               // 提案补充说明
	Status      string            `json:"status"`                // 当前提案状态
	Course      *ProposalCourseVO `json:"course"`                // 用户提交的课程快照
	Creator     *CreatorVO        `json:"creator"`               // 提案创建者信息
	AdminAction *AdminActionVO    `json:"adminAction,omitempty"` // 最近一次管理员操作；尚无管理员操作时省略
}

// CreatorVO 创建者信息
type CreatorVO struct {
	CreatorID   string `json:"creatorId"`   // 提案创建者用户ID
	CreatorName string `json:"creatorName"` // 创建者昵称；未设置时回退为 OpenID
	CreateTime  string `json:"createTime"`  // 提案创建时间，格式 YYYY-MM-DD HH:mm:ss
}

// AdminActionVO 管理员操作信息
type AdminActionVO struct {
	AdminID    string `json:"adminId"`          // 操作管理员用户ID
	AdminName  string `json:"adminName"`        // 管理员昵称；未设置时回退为 OpenID
	Action     string `json:"action"`           // 最近动作名称，例如 approve、reject、delete、update
	ActionTime string `json:"actionTime"`       // 操作时间，格式 YYYY-MM-DD HH:mm:ss
	Reason     string `json:"reason,omitempty"` // 日志正文或拒绝理由，无内容时省略
}

// ListProposalLogsTimelineReq 扁平化时间线日志请求参数
type ListProposalLogsTimelineReq struct {
	*PageParam
}

// ListProposalLogsTimelineResp 扁平化时间线日志响应
type ListProposalLogsTimelineResp struct {
	*Resp
	Total int64                    `json:"total"` // 全部变更日志总数
	Logs  []*ProposalTimelineLogVO `json:"logs"`  // 当前分页时间线日志，严格按操作时间倒序
}

// ProposalTimelineLogVO 提案时间线日志展示对象
type ProposalTimelineLogVO struct {
	LogID            string                 `json:"logId"`                      // 变更日志ID
	ProposalID       string                 `json:"proposalId,omitempty"`       // 关联提案ID；非提案操作时省略
	ActionType       string                 `json:"actionType"`                 // CREATE/APPROVE/REJECT/DELETE/UPDATE/GRANT_ADMIN/REVOKE_ADMIN
	OperatorID       string                 `json:"operatorId"`                 // 操作者用户ID
	OperatorName     string                 `json:"operatorName"`               // 操作者昵称；未设置时回退为 OpenID
	ActionTime       string                 `json:"actionTime"`                 // 操作时间，格式 YYYY-MM-DD HH:mm:ss
	ProposalSnapshot *ProposalSnapshotVO    `json:"proposalSnapshot,omitempty"` // 能找到关联提案时返回当前提案摘要
	Details          map[string]interface{} `json:"details,omitempty"`          // 额外操作信息；当前主要通过 content 返回日志正文
}

// ProposalSnapshotVO 提案快照信息
type ProposalSnapshotVO struct {
	Title      string `json:"title"`                // 提案标题
	CourseName string `json:"courseName,omitempty"` // 提案课程名称
	Department string `json:"department,omitempty"` // 提案课程开课院系
	Category   string `json:"category,omitempty"`   // 提案课程分类
}
