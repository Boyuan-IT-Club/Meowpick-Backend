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

type CommentVO struct {
	ID       string   `json:"id"`       // 评论ID
	CourseID string   `json:"courseId"` // 所属正式课程ID
	Content  string   `json:"content"`  // 评论正文
	UserID   string   `json:"userId"`   // 评论发布者用户ID
	Tags     []string `json:"tags"`     // 评论标签列表，可为空
	*LikeVO           // 当前用户的点赞状态及评论点赞总数
	ExtraInfo
	CreatedAt time.Time `json:"createdAt"` // 发布时间
	UpdatedAt time.Time `json:"updatedAt"` // 最近更新时间
}

type ExtraInfo struct {
	Name       string   `json:"name"`       // “我的评论”中附带的课程名称；普通课程评论列表中为空
	Category   string   `json:"category"`   // “我的评论”中附带的课程分类；普通课程评论列表中为空
	Department string   `json:"department"` // “我的评论”中附带的课程开课院系；普通课程评论列表中为空
	Teachers   []string `json:"teachers"`   // “我的评论”中附带的教师姓名与职称拼接字符串列表；普通课程评论列表中为空
}

// CreateCommentReq 对应 /api/comment/add 的请求体
type CreateCommentReq struct {
	CourseID string   `json:"courseId" binding:"required"` // 所属正式课程ID，必填
	Content  string   `json:"content" binding:"required"`  // 评论正文，必填
	Tags     []string `json:"tags"`                        // 可选标签列表
}

// CreateCommentResp 对应 /api/comment/add 的响应体
type CreateCommentResp struct {
	*Resp
	*CommentVO
}

// GetTotalCourseCommentsCountResp 对应 /api/search/total 的响应体
type GetTotalCourseCommentsCountResp struct {
	*Resp
	Count int64 `json:"count"` // 系统中所有未删除课程评论总数
}

// GetMyCommentsReq 是前端请求“我的吐槽”时，需要传递的数据结构。
type GetMyCommentsReq struct {
	*PageParam
}

// ListCourseCommentsReq 是前端分页请求某一课程下的评论时，需要传递的数据结构。
type ListCourseCommentsReq struct {
	ID       string `form:"courseId" binding:"required_without=LegacyID"` // 课程ID；兼容旧参数 id，但新调用统一使用 courseId
	LegacyID string `form:"id" swaggerignore:"true"`
	*PageParam
}

// ListCourseCommentsResp 是后端返回给前端的、分页的评论历史数据。
type ListCourseCommentsResp struct {
	*Resp
	Total    int64        `json:"total"`    // 该课程未删除评论总数
	Comments []*CommentVO `json:"comments"` // 当前分页评论，按创建时间倒序
}

// GetMyCommentsResp “我的吐槽” 比一般的CommentVO多了一些课程的信息
type GetMyCommentsResp struct {
	*Resp
	Total    int64        `json:"total"`    // 当前用户未删除评论总数
	Comments []*CommentVO `json:"comments"` // 当前分页评论，附带对应课程信息及当前点赞状态
}
