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

// ProposalCourseVO 提案中的课程信息
type ProposalCourseVO struct {
	ID         string       `json:"id,omitempty"` // 课程ID；创建提案时无需传入，finalCourse 返回正式课程时可能携带
	Name       string       `json:"name"`         // 课程名称，去除首尾空白后不能为空
	Code       string       `json:"code"`         // 课程代码，可为空
	Category   string       `json:"category"`     // 课程分类名称，不能为空；审批时可落库为新分类
	Campuses   []string     `json:"campuses"`     // 开设校区名称列表，至少一项且必须是系统已有校区，不允许由提案创建新校区
	Department string       `json:"department"`   // 课程开课院系，不能为空；与教师所属院系是两个独立字段
	Teachers   []*TeacherVO `json:"teachers"`     // 任课教师列表；每项姓名必填，id 为空表示新增教师，department 可为空
}

// CreateProposalReq 新增投票请求参数
type CreateProposalReq struct {
	Title        string            `json:"title" binding:"required"`  // 提案标题，去除首尾空白后不能为空
	Content      string            `json:"content"`                   // 提案补充说明，可为空
	Course       *ProposalCourseVO `json:"course" binding:"required"` // 提议新增的课程完整信息
	ShowUsername bool              `json:"showUsername"`              // 是否允许公开展示提案创建者昵称；false 表示匿名展示
}

// CreateProposalResp 新增投票响应
type CreateProposalResp struct {
	*Resp
	ProposalID string      `json:"proposalId"` // 新建提案ID
	Proposal   *ProposalVO `json:"proposal"`   // 新建后的完整待审核提案
}

type ProposalVO struct {
	ID           string            `json:"id"`           // 提案ID
	UserID       string            `json:"userId"`       // 提案创建者用户ID；是否展示昵称由 showUsername 控制
	Title        string            `json:"title"`        // 提案标题
	Content      string            `json:"content"`      // 提案补充说明，可能为空
	Status       string            `json:"status"`       // 提案状态：pending 待审核、approved 已通过、rejected 已拒绝
	Deleted      bool              `json:"deleted"`      // 是否已被创建者软删除
	RejectReason string            `json:"rejectReason"` // 拒绝理由；未拒绝或未填写理由时为空
	*LikeVO                        // 当前用户的点赞状态及提案点赞数
	Course       *ProposalCourseVO `json:"course"`                 // 用户最初提交的课程内容，审批时不会被 finalCourse 覆盖
	FinalCourse  *ProposalCourseVO `json:"finalCourse,omitempty"`  // 审批通过后的正式课程；历史/列表/筛选接口对已通过提案返回，详情接口仅创建者或管理员可见，课程已删除或查询失败时省略
	ShowUsername bool              `json:"showUsername"`           // 是否允许公开展示创建者昵称
	Contribution int64             `json:"contribution,omitempty"` // 本提案结算贡献值；仅创建者可见，其他用户响应中为 -1
	CreatedAt    time.Time         `json:"createdAt"`              // 提案创建时间
	UpdatedAt    time.Time         `json:"updatedAt"`              // 最近一次编辑或审批时间
}

// ListProposalReq 对应 /api/proposal/list 的请求体（分页）
type ListProposalReq struct {
	Status string `json:"status" form:"status"` // 状态筛选：pending、approved、rejected；管理员不传时查询全部，普通用户始终只查询 approved
	*PageParam
}

// FilterProposalReq 对应 /api/proposal/filter 的请求参数（分页筛选）
type FilterProposalReq struct {
	Statuses   []string `form:"status"`     // 状态多选；支持重复 query 参数及 JSON 数组字符串，普通用户传值会被忽略并固定为 approved
	Campuses   []string `form:"campus"`     // 校区多选；支持重复 query 参数及 JSON 数组字符串，值必须是已有校区名称
	Department string   `form:"department"` // 课程开课院系名称，精确匹配；为空时不筛选
	Category   string   `form:"category"`   // 课程分类名称，精确匹配；为空时不筛选
	*PageParam
}

// ListProposalResp 对应 /api/proposal/list 的响应体
type ListProposalResp struct {
	*Resp
	Total     int64         `json:"total"`     // 符合当前权限与筛选条件的提案总数
	Proposals []*ProposalVO `json:"proposals"` // 当前分页提案列表，无结果时为空数组
}

type GetProposalReq struct {
	ProposalID string `json:"proposalId"`
}

type GetProposalResp struct {
	*Resp
	Proposal *ProposalVO `json:"proposal"`
}

type RejectProposalReq struct {
	ProposalID string `json:"proposalId"` // 提案ID，由 URL path 写入，请求体无需传
	Reason     string `json:"reason"`     // 拒绝理由，可为空
}

type RejectProposalResp struct {
	*Resp
	Rejected     bool  `json:"rejected"`     // 是否成功拒绝
	PendingCount int64 `json:"pendingCount"` // 操作后剩余待审核提案数量
}

type ToggleProposalReq struct {
	ProposalID  string            `json:"proposalID"`  // 提案ID，由 URL path 写入，请求体无需传
	FinalCourse *ProposalCourseVO `json:"finalCourse"` // 管理员最终确认的课程；省略、null 或空请求体时使用用户原始 course
}

type ToggleProposalResp struct {
	Proposal    bool  `json:"proposal"`    // 是否成功通过提案
	ProposalCnt int64 `json:"proposalCnt"` // 操作后剩余待审核提案数量
	*Resp
}

type RevokeProposalReq struct {
	ProposalID string `json:"-"`          // 从 URL path 获取
	ActionType string `json:"actionType"` // 必填；approve 撤回已通过提案，reject 撤回已拒绝提案
}

type RevokeProposalResp struct {
	*Resp
	ProposalID string `json:"proposalId"` // 已恢复为 pending 的提案ID
}

type DeleteProposalReq struct {
	ProposalID string `json:"proposalId"`
}

type DeleteProposalResp struct {
	*Resp
	ProposalID string    `json:"proposalId"` // 被软删除的提案ID
	DeletedAt  time.Time `json:"deletedAt"`  // 本次删除完成时间
	OperatorID string    `json:"operatorId"` // 执行删除的提案创建者用户ID
	Deleted    bool      `json:"deleted"`    // 是否成功软删除
}

// UpdateProposalReq 更新提案请求参数
type UpdateProposalReq struct {
	ProposalID string            `json:"-"`
	Title      string            `json:"title" binding:"required"`   // 更新后的提案标题，不能为空
	Content    string            `json:"content" binding:"required"` // 更新后的补充说明，当前接口要求非空
	Course     *ProposalCourseVO `json:"course" binding:"required"`  // 更新后的完整课程信息，不支持仅传部分字段
}

// UpdateProposalResp 更新提案响应参数
type UpdateProposalResp struct {
	*Resp      `json:",inline"`
	ProposalID string `json:"proposalId"` // 更新成功的提案ID
}

// GetProposalSuggestionsReq 获取提案搜索建议请求
type GetProposalSuggestionsReq struct {
	Keyword string `form:"keyword" binding:"required"` // 提案标题模糊搜索关键词，不能为空
	*PageParam
}

// GetProposalFieldSuggestionsReq 获取提案字段建议请求
type GetProposalFieldSuggestionsReq struct {
	Field   string `form:"field" binding:"required"`   // 建议字段：department、category、campus、courseName、courseCode、teacherName
	Keyword string `form:"keyword" binding:"required"` // 匹配关键词，不能为空
	*PageParam
}

// GetProposalFieldSuggestionsResp 获取提案字段建议响应
type GetProposalFieldSuggestionsResp struct {
	*Resp
	Field       string               `json:"field"`       // 原样返回请求的字段类型
	Suggestions []*FieldSuggestionVO `json:"suggestions"` // 当前分页建议列表
	Total       int64                `json:"total"`       // 匹配建议总数
}

// FieldSuggestionVO 字段建议视图对象
type FieldSuggestionVO struct {
	ID      string         `json:"id,omitempty"`      // 建议关联实体ID；教师、课程等数据库建议可能返回
	Value   string         `json:"value"`             // 选中建议后写入请求字段的值
	Label   string         `json:"label"`             // 前端展示文本；教师有职称时格式为“姓名 - 职称”
	Courses *[]CourseBrief `json:"courses,omitempty"` // 仅 teacherName 建议返回，固定存在且最多两门；其他字段省略
}

// CourseBrief 教师字段建议中展示的课程简要信息。
type CourseBrief struct {
	ID   string `json:"id"`   // 未删除课程ID
	Name string `json:"name"` // 课程名称
}

// GetMyProposalsReq 获取我的提案请求
type GetMyProposalsReq struct {
	*PageParam
}

// GetMyProposalsResp 获取我的提案响应
type GetMyProposalsResp struct {
	*Resp
	Total     int64         `json:"total"`     // 当前用户全部提案总数，包含所有状态及已删除提案
	Proposals []*ProposalVO `json:"proposals"` // 当前分页的个人提案历史
}
