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

package consts

import "time"

// 数据库字段相关
const (
	ID         = "_id"
	Status     = "status"
	CreatedAt  = "createdAt"
	UpdatedAt  = "updatedAt"
	DeletedAt  = "deletedAt"
	UserID     = "userId"
	Query      = "query"
	Deleted    = "deleted"
	TargetID   = "targetId"
	Active     = "active"
	CourseID   = "courseId"
	OpenID     = "openId"
	TeacherIDs = "teacherIds"
	Category   = "category"
	Department = "department"
	Campuses   = "campuses"
	Code       = "code"
	Name       = "name"
	Tags       = "tags"
	Count      = "count"
	TargetType = "targetType"
)

// 缓存相关
const (
	CacheSearchHistoryKeyPrefix = "meowpick:searchhistory:"
	CacheCommentKeyPrefix       = "meowpick:comment:"
	CacheLikeKeyPrefix          = "meowpick:like:"
	CacheUserKeyPrefix          = "meowpick:user:"
	CacheTeacherKeyPrefix       = "meowpick:teacher:"
	CacheCourseKeyPrefix        = "meowpick:course:"
	CacheProposalKeyPrefix      = "meowpick:proposal:"

	CacheCommentCountTTL   = 12 * time.Hour
	CacheLikeStatusTTL     = 10 * time.Minute
	CacheProposalStatusTTL = 10 * time.Minute
)

// 上下文相关
const (
	CtxUserID     = "userId"
	CtxToken      = "token"
	CtxLikeID     = "likeId"
	CtxCourseID   = "courseId"
	CtxProposalID = "proposalId"
)

// Request 相关
const (
	ReqCourse     = "course"
	ReqTeacher    = "teacher"
	ReqDepartment = "department"
	ReqCategory   = "category"
	ReqOpenID     = "openId"
	ReqType       = "type"
	ReqCourseID   = "courseId"
	ReqTargetID   = "targetId"
	ReqTitle      = "title"
	ReqProposalID = "proposalId"
)

// 限制相关
const (
	SearchHistoryLimit = 15
)

// 提案状态相关
const (
	ProposalStatusPending  = "pending"  // 待审核
	ProposalStatusApproved = "approved" // 已通过
	ProposalStatusRejected = "rejected" // 已拒绝
)

// 点赞目标类型相关
const (
	LikeTargetTypeComment  = "comment"
	LikeTargetTypeProposal = "proposal"
)
