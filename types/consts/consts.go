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

var PageSize int64 = 10

// 数据库相关
const (
	ID         = "_id"
	Status     = "status"
	CreatedAt  = "createdAt"
	UpdatedAt  = "updatedAt"
	UserId     = "userId"
	Query      = "query"
	Deleted    = "deleted"
	TargetId   = "targetId"
	Active     = "active"
	CourseId   = "courseId"
	OpenId     = "openId"
	TeacherIds = "teacherIds"
	Categories = "categories"
	Department = "department"
	Campuses   = "campuses"
	Code       = "code"
	Name       = "name"
)

// 元素类别相关（如课程、评论、老师）
const (
	CourseType int32 = 101 + iota
	CommentType
)

// 业务相关
const (
	ContextUserID = "userID"
	ContextTarget = "targetID"
	ContextToken  = "token"
)

// 限制相关
const (
	SearchHistoryLimit = 15
)

// 类型相关
const (
	Category = "category"
	Course   = "course"
	Teacher  = "teacher"
)
