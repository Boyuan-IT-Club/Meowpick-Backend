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

package model

import "time"

const (
	ProposalCreateCourse  = "create_course"
	ProposalUpdateCourse  = "update_course"
	ProposalUpdateTeacher = "update_teacher"
)

// ProposalPatch preserves omitted fields separately from explicit empty values.
type ProposalPatch struct {
	Name       *string             `bson:"name,omitempty" json:"name,omitempty"`
	Code       *string             `bson:"code,omitempty" json:"code,omitempty"`
	Department *string             `bson:"department,omitempty" json:"department,omitempty"`
	Category   *string             `bson:"category,omitempty" json:"category,omitempty"`
	Title      *string             `bson:"title,omitempty" json:"title,omitempty"`
	Campuses   *[]string           `bson:"campuses,omitempty" json:"campuses,omitempty"`
	Teachers   *[]*ProposalTeacher `bson:"teachers,omitempty" json:"teachers,omitempty"`
}

type ProposalDecision struct {
	BeforeValues    *ProposalPatch `bson:"beforeValues,omitempty"`
	AfterValues     *ProposalPatch `bson:"afterValues,omitempty"`
	CreatedTeachers []*Teacher     `bson:"createdTeachers,omitempty"`

	ID                string    `bson:"_id"`
	Type              string    `bson:"type"`
	TargetID          string    `bson:"targetId"`
	ProposalIDs       []string  `bson:"proposalIds"`
	TriggerID         string    `bson:"triggerId"`
	BeforeCourse      *Course   `bson:"beforeCourse,omitempty"`
	AfterCourse       *Course   `bson:"afterCourse,omitempty"`
	BeforeTeacher     *Teacher  `bson:"beforeTeacher,omitempty"`
	AfterTeacher      *Teacher  `bson:"afterTeacher,omitempty"`
	CreatedTeacherIDs []string  `bson:"createdTeacherIds"`
	CreatedAt         time.Time `bson:"createdAt"`
	Revoked           bool      `bson:"revoked"`
	RevokedAt         time.Time `bson:"revokedAt,omitempty"`
}

func (p *Proposal) EffectiveType() string {
	if p.Type == "" {
		return ProposalCreateCourse
	}
	return p.Type
}
