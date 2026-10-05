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
	Code       string       `json:"code"`         // 课程代码，去除首尾空白后不能为空
	Category   string       `json:"category"`     // 课程分类名称，不能为空；审批时可落库为新分类
	Campuses   []string     `json:"campuses"`     // 开设校区名称列表，至少一项且必须是系统已有校区，不允许由提案创建新校区
	Department string       `json:"department"`   // 课程开课院系，不能为空；与教师所属院系是两个独立字段
	Teachers   []*TeacherVO `json:"teachers"`     // 任课教师列表；每项姓名必填，id 为空表示新增教师，department 可为空
}

// CreateProposalReq 新增投票请求参数
type CreateProposalReq struct {
	Type      string         `json:"type"`                // create_course、update_course、update_teacher；缺失兼容 create_course
	TargetID  string         `json:"targetId,omitempty"`  // 修改提案必填，目标正式课程或教师ID
	Suggested *ProposalPatch `json:"suggested,omitempty"` // 修改提案只提交拟改字段；未传表示不修改

	Title        string            `json:"-"`                // 仅内部旧数据兼容，不接收或返回title
	Content      string            `json:"content"`          // 提案补充说明，可为空
	Course       *ProposalCourseVO `json:"course,omitempty"` // 提议新增的课程完整信息
	ShowUsername bool              `json:"showUsername"`     // 是否允许公开展示提案创建者昵称；false 表示匿名展示
}

// CreateProposalResp 新增投票响应
type CreateProposalResp struct {
	PendingDuplicateIDs []string `json:"pendingDuplicateIds"` // 非阻断提示：相同业务建议的其他作者待审提案

	*Resp
	ProposalID string      `json:"proposalId"` // 新建提案ID
	Proposal   *ProposalVO `json:"proposal"`   // 新建后的完整待审核提案
}

// ResubmitProposalReq 重新提交被拒绝提案的完整内容。
type ResubmitProposalReq struct {
	Type      string         `json:"type"`                // create_course、update_course、update_teacher；缺失兼容 create_course
	TargetID  string         `json:"targetId,omitempty"`  // 修改提案必填，目标正式课程或教师ID
	Suggested *ProposalPatch `json:"suggested,omitempty"` // 修改提案只提交拟改字段；未传表示不修改

	ProposalID   string            `json:"-"`
	Title        string            `json:"-"`                // 仅内部历史兼容
	Content      string            `json:"content"`          // 新提案补充说明，可为空
	Course       *ProposalCourseVO `json:"course,omitempty"` // 新提案的完整课程信息
	ShowUsername bool              `json:"showUsername"`     // 是否允许公开展示新提案创建者昵称
}

// ResubmitProposalResp 返回被替换的旧提案和新建提案。
type ResubmitProposalResp struct {
	*Resp
	PreviousProposalID string      `json:"previousProposalId"` // 已软删除的原拒绝提案ID
	ProposalID         string      `json:"proposalId"`         // 新建待审核提案ID
	Proposal           *ProposalVO `json:"proposal"`           // 新建后的完整待审核提案
}

type ProposalVO struct {
	Type            string         `json:"type"`
	TargetID        string         `json:"targetId,omitempty"`
	DisplayName     string         `json:"displayName"`
	Suggested       *ProposalPatch `json:"suggested,omitempty"`
	Before          *ProposalPatch `json:"before,omitempty"`
	Final           *ProposalPatch `json:"final,omitempty"`
	DecisionBatchID string         `json:"decisionBatchId,omitempty"`

	ID           string            `json:"id"`               // 提案ID
	UserID       string            `json:"userId,omitempty"` // 匿名时对其他普通用户省略；作者和管理员可见
	Title        string            `json:"-"`                // 内部历史标题，不对外返回
	Content      string            `json:"content"`          // 提案补充说明，可能为空
	Status       string            `json:"status"`           // 提案状态：pending 待审核、approved 已通过、rejected 已拒绝
	Deleted      bool              `json:"deleted"`          // 是否已被创建者软删除
	RejectReason string            `json:"rejectReason"`     // 拒绝理由；未拒绝或未填写理由时为空
	*LikeVO                        // 当前用户的点赞状态及提案点赞数
	Course       *ProposalCourseVO `json:"course,omitempty"`       // 用户最初提交的课程内容，审批时不会被 finalCourse 覆盖
	FinalCourse  *ProposalCourseVO `json:"finalCourse,omitempty"`  // 管理员最终课程快照；pending时为管理员修改后的待审内容，approved时为审批当时快照
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

// SuggestProposalReq 对应 /api/proposal/suggest 的请求参数（分页搜索与筛选）
type SuggestProposalReq struct {
	Type string `form:"type"` // 可选提案类型

	Keyword    string   `form:"keyword"`    // 按显示名、课程名和兼容历史标题进行普通文本模糊搜索
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
	ProposalIDs []string `json:"proposalIds,omitempty"` // 额外勾选的待审提案，与path提案一起拒绝，共用reason；不会自动扩展范围

	ProposalID string `json:"proposalId"` // 提案ID，由 URL path 写入，请求体无需传
	Reason     string `json:"reason"`     // 拒绝理由，可为空
}

type RejectProposalResp struct {
	DecisionBatchID string   `json:"decisionBatchId,omitempty"`
	ProposalIDs     []string `json:"proposalIds"`
	TargetID        string   `json:"targetId,omitempty"`

	*Resp
	Rejected     bool  `json:"rejected"`     // 是否成功拒绝
	PendingCount int64 `json:"pendingCount"` // 操作后剩余待审核提案数量
}

type ToggleProposalReq struct {
	ProposalIDs          []string       `json:"proposalIds,omitempty"`          // 管理员手动选择共同审批的待审新增课程提案；各自自动组也会纳入
	Final                *ProposalPatch `json:"final,omitempty"`                // 修改类型最终字段；省略时优先采用管理员修改值，未修改过则采用suggested
	PreviewToken         string         `json:"previewToken,omitempty"`         // 审批确认必填，先调用preview取得；变化时重新preview
	ConfirmedNewTeachers []string       `json:"confirmedNewTeachers,omitempty"` // 有同名正式教师时，明确仍新建的教师姓名

	ProposalID  string            `json:"proposalID"`  // 提案ID，由 URL path 写入，请求体无需传
	Title       string            `json:"-"`           // 内部历史兼容，不接收标题
	FinalCourse *ProposalCourseVO `json:"finalCourse"` // 新增类型的最终完整课程；省略时优先采用管理员修改值，未修改过则采用原始course
}

type ToggleProposalResp struct {
	DecisionBatchID string   `json:"decisionBatchId,omitempty"`
	ProposalIDs     []string `json:"proposalIds"`
	TargetID        string   `json:"targetId,omitempty"`

	Proposal    bool  `json:"proposal"`    // 是否成功通过提案
	ProposalCnt int64 `json:"proposalCnt"` // 操作后剩余待审核提案数量
	*Resp
}

type RevokeProposalReq struct {
	ProposalID string `json:"-"`          // 从 URL path 获取
	ActionType string `json:"actionType"` // 必填；approve 仅可在最近审批通过后不足 24 小时内撤回，reject 撤回已拒绝提案且无时间限制
}

type RevokeProposalResp struct {
	DecisionBatchID string   `json:"decisionBatchId,omitempty"`
	ProposalIDs     []string `json:"proposalIds"`
	TargetID        string   `json:"targetId,omitempty"`

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
	Type      string         `json:"type"`                // create_course、update_course、update_teacher；缺失兼容 create_course
	TargetID  string         `json:"targetId,omitempty"`  // 修改提案必填，目标正式课程或教师ID
	Suggested *ProposalPatch `json:"suggested,omitempty"` // 修改提案只提交拟改字段；未传表示不修改

	ProposalID string            `json:"-"`
	Title      string            `json:"-"`                // 内部历史兼容，不接收
	Content    string            `json:"content"`          // 内部兼容；管理员更新不覆盖作者说明
	Course     *ProposalCourseVO `json:"course,omitempty"` // 新增类型管理员修改后的待审资料，完整列表
}

// UpdateProposalResp 更新提案响应参数
type UpdateProposalResp struct {
	*Resp      `json:",inline"`
	ProposalID string `json:"proposalId"` // 主提案ID；管理员修改待审提案时同步自动组各成员
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
	Title   *string        `json:"title,omitempty"`   // teacherName 建议固定返回教师职称（历史教师无职称时为空字符串）；其他字段省略
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
