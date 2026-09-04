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
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Code        string               `json:"code"`
	Category    string               `json:"category"`
	Campuses    []string             `json:"campuses"`
	Department  string               `json:"department"`
	Teachers    []*TeacherVO         `json:"teachers"`
	TagCount    map[string]int64     `json:"tagCount"`
	Contributor *CourseContributorVO `json:"contributor,omitempty"`
}

// CourseContributorVO describes the proposal source of a dynamically added
// course. UserID and Username are omitted when the proposal author chose not
// to expose their nickname.
type CourseContributorVO struct {
	ProposalID   string `json:"proposalId"`
	UserID       string `json:"userId,omitempty"`
	Username     string `json:"username,omitempty"`
	ShowUsername bool   `json:"showUsername"`
}

type ListCoursesReq struct {
	Keyword string `form:"keyword"`
	Type    string `form:"type"` // teacher or course
	*PageParam
}

type ListCoursesResp struct {
	*Resp
	*PaginatedCourses
}

type GetCourseReq struct {
	CourseID string `form:"courseId" binding:"required"`
}

type GetCourseResp struct {
	*Resp
	Course *CourseVO `json:"course"`
}

type GetCourseDepartmentsReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseDepartmentsResp struct {
	*Resp
	Departments []string `json:"departments"`
}

type GetCourseCategoriesReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseCategoriesResp struct {
	*Resp
	Categories []string `json:"categories"`
}

type GetCourseCampusesReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseCampusesResp struct {
	*Resp
	Campuses []string `json:"campuses"`
}

type PaginatedCourses struct {
	Courses []*CourseVO `json:"courses"` // 当前页的课程列表
	Total   int64       `json:"total"`   // 符合条件的总记录数
	*PageParam
}
