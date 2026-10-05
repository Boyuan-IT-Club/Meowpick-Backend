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

// ProposalPatch distinguishes omission from clearing. title/department may be
// empty for update_teacher. teachers/campuses replace the complete unordered list.
type ProposalPatch struct {
	Name       *string       `json:"name,omitempty"`
	Code       *string       `json:"code,omitempty"`
	Department *string       `json:"department,omitempty"`
	Category   *string       `json:"category,omitempty"`
	Title      *string       `json:"title,omitempty"`
	Campuses   *[]string     `json:"campuses,omitempty"`
	Teachers   *[]*TeacherVO `json:"teachers,omitempty"`
}

type ProposalDifference struct {
	Field     string      `json:"field"`
	Original  interface{} `json:"original"`
	Current   interface{} `json:"current"`
	Suggested interface{} `json:"suggested"`
	State     string      `json:"state"` // unchanged/realized/conflict
}

type ProposalCandidate struct {
	Proposal             *ProposalVO          `json:"proposal"`
	Automatic            bool                 `json:"automatic"`
	Differences          []ProposalDifference `json:"differences"`
	ExpectedContribution int64                `json:"expectedContribution"`
}

type ApprovalPreviewResp struct {
	*Resp
	PreviewToken      string                  `json:"previewToken"`
	Members           []*ProposalCandidate    `json:"members"`           // 自动组及手动选择成员的自动组闭包
	Suspicious        []*ProposalCandidate    `json:"suspicious"`        // 同名、尚未选择的待审新增课程提案
	ExistingCourses   []*ProposalCourseVO     `json:"existingCourses"`   // 同名正式课程，手动核对或拒绝
	TeacherCandidates map[string][]*TeacherVO `json:"teacherCandidates"` // 拟新建姓名的同名正式教师
	Conflicts         []ProposalDifference    `json:"conflicts"`
	FinalCourse       *ProposalCourseVO       `json:"finalCourse,omitempty"`
	Final             *ProposalPatch          `json:"final,omitempty"`
	CanApprove        bool                    `json:"canApprove"`
}

type EntityHistoryReq struct {
	TargetID   string `json:"-"`
	TargetType string `json:"-"`
	*PageParam
}

type EntityHistoryItem struct {
	Legacy          bool                   `json:"legacy"` // 旧审批无不可变最终快照时为true，final省略，不能用当前资料冒充创建时资料
	DecisionBatchID string                 `json:"decisionBatchId"`
	Type            string                 `json:"type"`
	ProposalIDs     []string               `json:"proposalIds"`
	Contributors    []*CourseContributorVO `json:"contributors"`
	Before          *ProposalPatch         `json:"before,omitempty"`
	Final           *ProposalPatch         `json:"final,omitempty"`
	Suggestions     []*ProposalPatch       `json:"suggestions"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type EntityHistoryResp struct {
	*Resp
	Total   int64                `json:"total"`
	History []*EntityHistoryItem `json:"history"`
}

type GetTeacherResp struct {
	*Resp
	Teacher      *TeacherVO             `json:"teacher"`
	Contributors []*CourseContributorVO `json:"contributors"`
}
