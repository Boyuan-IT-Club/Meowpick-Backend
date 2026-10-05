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
	"strings"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
)

func TestNewCourseAutomaticGroupBoundary(t *testing.T) {
	baseline := &dto.ProposalCourseVO{Name: "课程", Code: "C1", Department: "院系", Category: "分类", Campuses: []string{"校区一", "校区二"}, Teachers: []*dto.TeacherVO{{Name: "新教师", Title: "讲师"}, {ID: "existing", Name: "已有教师"}}}
	cases := []struct {
		name   string
		mutate func(*dto.ProposalCourseVO)
		same   bool
	}{
		{"new teacher titles are ignored", func(p *dto.ProposalCourseVO) { p.Teachers[0].Title = "教授" }, true},
		{"new teacher department is not identity", func(p *dto.ProposalCourseVO) { p.Teachers[0].Department = "另一院系" }, true},
		{"set order ignored", func(p *dto.ProposalCourseVO) {
			p.Campuses[0], p.Campuses[1] = p.Campuses[1], p.Campuses[0]
			p.Teachers[0], p.Teachers[1] = p.Teachers[1], p.Teachers[0]
		}, true},
		{"outer whitespace ignored", func(p *dto.ProposalCourseVO) { p.Name = " 课程 "; p.Code = " C1 " }, true},
		{"course name", func(p *dto.ProposalCourseVO) { p.Name = "课程二" }, false},
		{"code", func(p *dto.ProposalCourseVO) { p.Code = "C2" }, false},
		{"department", func(p *dto.ProposalCourseVO) { p.Department = "院系二" }, false},
		{"category", func(p *dto.ProposalCourseVO) { p.Category = "分类二" }, false},
		{"campus", func(p *dto.ProposalCourseVO) { p.Campuses = []string{"校区一"} }, false},
		{"existing IDs", func(p *dto.ProposalCourseVO) { p.Teachers[1].ID = "other" }, false},
		{"existing and new never conflate", func(p *dto.ProposalCourseVO) { p.Teachers[1].ID = "" }, false},
		{"teacher subset", func(p *dto.ProposalCourseVO) { p.Teachers = p.Teachers[:1] }, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			value := copyAs[dto.ProposalCourseVO](baseline)
			tt.mutate(value)
			if got := newCourseKey(baseline) == newCourseKey(value); got != tt.same {
				t.Fatalf("same=%v want %v", got, tt.same)
			}
		})
	}
}
func TestPatchDistinguishesClearingAndOmission(t *testing.T) {
	empty := ""
	patch := &dto.ProposalPatch{Title: &empty}
	if err := validatePatch(model.ProposalUpdateTeacher, patch); err != nil {
		t.Fatal(err)
	}
	current := &dto.ProposalPatch{Name: textPointer("老师"), Title: textPointer("讲师"), Department: textPointer("")}
	before, changes := selectChanged(current, patch)
	if changes.Name != nil || changes.Title == nil || *changes.Title != "" || before.Title == nil || *before.Title != "讲师" {
		t.Fatalf("before=%#v changes=%#v", before, changes)
	}
	if err := validatePatch(model.ProposalUpdateTeacher, &dto.ProposalPatch{Name: &empty}); err == nil {
		t.Fatal("clearing name accepted")
	}
	if err := validatePatch(model.ProposalUpdateTeacher, &dto.ProposalPatch{Code: textPointer("x")}); err == nil {
		t.Fatal("course field accepted for teacher")
	}
}
func TestTeacherRelationIgnoresEditableDisplayFields(t *testing.T) {
	a := []*dto.TeacherVO{{ID: "t", Name: "老师", Title: "讲师"}}
	b := []*dto.TeacherVO{{ID: "t", Name: "更正姓名", Title: "教授"}}
	if !fieldEqual("teachers", a, b) {
		t.Fatal("same existing identity mistaken for relation change")
	}
	b[0].ID = "other"
	if fieldEqual("teachers", a, b) {
		t.Fatal("different existing identity conflated")
	}
}
func TestModificationContributionCountsOnlyActualAcceptedFields(t *testing.T) {
	before := &dto.ProposalPatch{Name: textPointer("A"), Code: textPointer("1"), Department: textPointer("原院系")}
	suggested := &dto.ProposalPatch{Name: textPointer("B"), Code: textPointer("2"), Department: textPointer("建议院系")}
	final := &dto.ProposalPatch{Name: textPointer("B"), Code: textPointer("1"), Department: textPointer("管理员院系")}
	if score := modificationScore(suggested, before, final); score != 2 {
		t.Fatalf("score=%d want base1+accepted1", score)
	}
	if score := modificationScore(suggested, before, before); score != 0 {
		t.Fatalf("no-op score=%d", score)
	}
}
func TestFeedbackUnicodeLength(t *testing.T) {
	for _, tt := range []struct {
		text  string
		valid bool
	}{{"  ", false}, {strings.Repeat("猫", 2000), true}, {strings.Repeat("猫", 2001), false}, {" 正文 ", true}} {
		_, err := feedbackText(tt.text)
		if (err == nil) != tt.valid {
			t.Fatalf("length=%d valid=%v err=%v", len(tt.text), tt.valid, err)
		}
	}
}

func TestCourseModificationGroupUsesTeacherIdentity(t *testing.T) {
	first := &model.Proposal{Type: model.ProposalUpdateCourse, TargetID: "course", Suggested: copyAs[model.ProposalPatch](&dto.ProposalPatch{Teachers: &[]*dto.TeacherVO{{ID: "one", Name: "甲", Title: "讲师"}, {ID: "two", Name: "乙"}}})}
	second := copyAs[model.Proposal](first)
	second.Suggested.Teachers = &[]*model.ProposalTeacher{{TeacherID: "two", Name: "乙的新名"}, {TeacherID: "one", Name: "甲", Title: "教授"}}
	if proposalAutoKey(first) != proposalAutoKey(second) {
		t.Fatal("same unordered relation was split by mutable names/titles")
	}
}
