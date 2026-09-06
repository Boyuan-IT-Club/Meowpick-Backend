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

type GetTeachersReq struct {
	TeacherID string `form:"teacherId"`
	*PageParam
}

type GetTeachersResp struct {
	*Resp
	*PaginatedCourses
}

type CreateTeacherReq struct {
	Name       string `json:"name" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Department string `json:"department" binding:"required"`
}

type CreateTeacherResp struct {
	*Resp
	*TeacherVO
}

type TeacherVO struct {
	ID         string `json:"id"`         // 教师ID；提案引用已有教师时传入，新教师可为空
	Name       string `json:"name"`       // 教师姓名
	Title      string `json:"title"`      // 教师职称
	Department string `json:"department"` // 教师所属院系；提案场景允许为空，表示暂未维护
}

type GetTeacherSuggestionsReq struct {
	Keyword string `form:"keyword" binding:"required"`
	*PageParam
}

type GetTeacherSuggestionsResp struct {
	*Resp
	Teachers []*TeacherVO `json:"teachers"`
}
