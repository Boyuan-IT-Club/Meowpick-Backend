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

// CourseVO 传递给前端的课程类型 模糊搜索和精确搜索结果都可用此类型
type CourseVO struct {
	ID          string               `json:"id"`                    // 正式课程ID
	Name        string               `json:"name"`                  // 课程名称
	Code        string               `json:"code"`                  // 课程代码，历史数据中可能为空
	Category    string               `json:"category"`              // 课程分类名称；映射缺失时返回未知分类
	Campuses    []string             `json:"campuses"`              // 开设校区名称列表
	Department  string               `json:"department"`            // 课程开课院系名称；不是教师所属院系
	Teachers    []*TeacherVO         `json:"teachers"`              // 任课教师详情；教师所属院系暂未维护时返回未知开课院系
	TagCount    map[string]int64     `json:"tagCount"`              // 当前未删除评论中出现次数最多的至多三个非空标签及计数
	Contributor *CourseContributorVO `json:"contributor,omitempty"` // 课程由审批提案创建时返回来源信息；历史课程无来源提案时省略
}

// CourseContributorVO describes the proposal source of a dynamically added
// course. UserID and Username are omitted when the proposal author chose not
// to expose their nickname.
type CourseContributorVO struct {
	ProposalID   string `json:"proposalId"`         // 创建该课程的已通过提案ID
	UserID       string `json:"userId,omitempty"`   // 提案允许展示昵称时返回创建者用户ID，匿名时省略
	Username     string `json:"username,omitempty"` // 提案允许展示昵称时返回当前昵称，未设置昵称时可能为空
	ShowUsername bool   `json:"showUsername"`       // 来源提案是否允许公开展示创建者昵称
}

type ListCoursesReq struct {
	Keyword string `form:"keyword"` // 搜索值：course 时为课程名关键词；其他类型时为教师、分类或开课院系的完整名称
	Type    string `form:"type"`    // 必填搜索类型：course、teacher、category、department
	*PageParam
}

type ListCoursesResp struct {
	*Resp
	*PaginatedCourses
}

type GetCourseReq struct {
	CourseID string `form:"courseId" binding:"required"` // 正式课程ID，由 URL path 写入
}

type GetCourseResp struct {
	*Resp
	Course *CourseVO `json:"course"` // 未被软删除的课程完整信息
}

type GetCourseDepartmentsReq struct {
	Keyword string `form:"keyword"` // 完整课程名称，精确匹配未删除课程
}

type GetCourseDepartmentsResp struct {
	*Resp
	Departments []string `json:"departments"` // 同名未删除课程涉及的去重开课院系名称
}

type GetCourseCategoriesReq struct {
	Keyword string `form:"keyword"` // 完整课程名称，精确匹配未删除课程
}

type GetCourseCategoriesResp struct {
	*Resp
	Categories []string `json:"categories"` // 同名未删除课程涉及的去重课程分类名称
}

type GetCourseCampusesReq struct {
	Keyword string `form:"keyword"` // 完整课程名称，精确匹配未删除课程
}

type GetCourseCampusesResp struct {
	*Resp
	Campuses []string `json:"campuses"` // 同名未删除课程涉及的去重开设校区名称
}

type PaginatedCourses struct {
	Courses []*CourseVO `json:"courses"` // 当前页的课程列表
	Total   int64       `json:"total"`   // 符合条件的总记录数
	*PageParam
}
