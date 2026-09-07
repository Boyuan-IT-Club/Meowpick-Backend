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

package service

import (
	"reflect"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
)

func TestTeacherSuggestionLabel(t *testing.T) {
	tests := []struct {
		name        string
		teacherName string
		title       string
		want        string
	}{
		{name: "with title", teacherName: "张三", title: "教授", want: "张三 - 教授"},
		{name: "without title", teacherName: "张三", title: "", want: "张三"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := teacherSuggestionLabel(tt.teacherName, tt.title); got != tt.want {
				t.Fatalf("teacherSuggestionLabel(%q, %q) = %q, want %q", tt.teacherName, tt.title, got, tt.want)
			}
		})
	}
}

func TestValidateProposalInputRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		course *dto.ProposalCourseVO
	}{
		{name: "blank title", title: " ", course: &dto.ProposalCourseVO{Name: "课程", Department: "院系", Category: "分类", Campuses: []string{"校区"}}},
		{name: "missing course", title: "标题"},
		{name: "blank course name", title: "标题", course: &dto.ProposalCourseVO{Department: "院系", Category: "分类", Campuses: []string{"校区"}}},
		{name: "blank department", title: "标题", course: &dto.ProposalCourseVO{Name: "课程", Category: "分类", Campuses: []string{"校区"}}},
		{name: "blank category", title: "标题", course: &dto.ProposalCourseVO{Name: "课程", Department: "院系", Campuses: []string{"校区"}}},
		{name: "missing campuses", title: "标题", course: &dto.ProposalCourseVO{Name: "课程", Department: "院系", Category: "分类"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateProposalInput(tt.title, tt.course); err == nil {
				t.Fatal("validateProposalInput() error = nil, want error")
			}
		})
	}
}

func TestValidateProposalInputAllowsTeacherWithoutDepartment(t *testing.T) {
	course := &dto.ProposalCourseVO{
		Name:       "测试课程",
		Department: "软件工程学院",
		Category:   "专业必修",
		Campuses:   []string{"普陀校区"},
		Teachers: []*dto.TeacherVO{{
			Name:  "新增教师",
			Title: "讲师",
		}},
	}

	if err := validateProposalInput("测试提案", course); err != nil {
		t.Fatalf("validateProposalInput() rejected teacher without department: %v", err)
	}
}

func TestShouldDeleteTeacher(t *testing.T) {
	tests := []struct {
		name               string
		courseReferenced   bool
		proposalReferenced bool
		want               bool
	}{
		{name: "unreferenced", want: true},
		{name: "referenced by course", courseReferenced: true},
		{name: "referenced by proposal", proposalReferenced: true},
		{name: "referenced by both", courseReferenced: true, proposalReferenced: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDeleteTeacher(tt.courseReferenced, tt.proposalReferenced); got != tt.want {
				t.Fatalf("shouldDeleteTeacher(%v, %v) = %v, want %v", tt.courseReferenced, tt.proposalReferenced, got, tt.want)
			}
		})
	}
}

func TestShouldDeleteMapping(t *testing.T) {
	tests := []struct {
		name              string
		courseReferenced  bool
		teacherReferenced bool
		want              bool
	}{
		{name: "unreferenced", want: true},
		{name: "referenced by course", courseReferenced: true},
		{name: "department referenced by teacher", teacherReferenced: true},
		{name: "referenced by course and teacher", courseReferenced: true, teacherReferenced: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDeleteMapping(tt.courseReferenced, tt.teacherReferenced); got != tt.want {
				t.Fatalf("shouldDeleteMapping(%v, %v) = %v, want %v", tt.courseReferenced, tt.teacherReferenced, got, tt.want)
			}
		})
	}
}

func TestRevokedCourseMappingReferencesIncludesDeletedTeacherDepartments(t *testing.T) {
	course := &model.Course{Department: 10, Category: 20}
	teachers := []*model.Teacher{
		{Department: 30},
		{Department: 10},
		{Department: 30},
		{Department: 0},
		nil,
	}
	want := []mappingReference{
		{mappingType: model.MappingTypeDepartment, code: 10},
		{mappingType: model.MappingTypeCategory, code: 20},
		{mappingType: model.MappingTypeDepartment, code: 30},
	}
	if got := revokedCourseMappingReferences(course, teachers); !reflect.DeepEqual(got, want) {
		t.Fatalf("revokedCourseMappingReferences() = %#v, want %#v", got, want)
	}
}
