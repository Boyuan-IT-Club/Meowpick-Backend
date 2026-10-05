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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
)

func copyAs[T any](value interface{}) *T {
	if value == nil {
		return nil
	}
	data, _ := json.Marshal(value)
	var target *T
	_ = json.Unmarshal(data, &target)
	return target
}
func valueHash(value interface{}) string {
	data, _ := json.Marshal(value)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
func textPointer(value string) *string { return &value }
func normalizedSet(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	sort.Strings(result)
	return result
}
func normalizeCourse(course *dto.ProposalCourseVO) {
	if course == nil {
		return
	}
	course.Name = strings.TrimSpace(course.Name)
	course.Code = strings.TrimSpace(course.Code)
	course.Category = strings.TrimSpace(course.Category)
	course.Department = strings.TrimSpace(course.Department)
	course.Campuses = normalizedSet(course.Campuses)
	for _, teacher := range course.Teachers {
		if teacher != nil {
			teacher.ID = strings.TrimSpace(teacher.ID)
			teacher.Name = strings.TrimSpace(teacher.Name)
			teacher.Title = strings.TrimSpace(teacher.Title)
			teacher.Department = strings.TrimSpace(teacher.Department)
		}
	}
}
func teacherIdentities(teachers []*dto.TeacherVO) []string {
	values := []string{}
	for _, teacher := range teachers {
		if teacher != nil {
			if teacher.ID != "" {
				values = append(values, "id:"+teacher.ID)
			} else {
				values = append(values, "new:"+strings.TrimSpace(teacher.Name))
			}
		}
	}
	return normalizedSet(values)
}
func newCourseKey(course *dto.ProposalCourseVO) string {
	if course == nil {
		return ""
	}
	return valueHash([]interface{}{strings.TrimSpace(course.Name), strings.TrimSpace(course.Code), strings.TrimSpace(course.Category), strings.TrimSpace(course.Department), normalizedSet(course.Campuses), teacherIdentities(course.Teachers)})
}
func patchValues(patch *dto.ProposalPatch) map[string]interface{} {
	values := map[string]interface{}{}
	if patch == nil {
		return values
	}
	for key, value := range map[string]*string{"name": patch.Name, "code": patch.Code, "department": patch.Department, "category": patch.Category, "title": patch.Title} {
		if value != nil {
			values[key] = *value
		}
	}
	if patch.Campuses != nil {
		values["campuses"] = normalizedSet(*patch.Campuses)
	}
	if patch.Teachers != nil {
		teachers := copyAs[[]*dto.TeacherVO](patch.Teachers)
		sort.Slice(*teachers, func(i, j int) bool { return valueHash((*teachers)[i]) < valueHash((*teachers)[j]) })
		values["teachers"] = *teachers
	}
	return values
}
func patchFromValues(values map[string]interface{}) *dto.ProposalPatch {
	return copyAs[dto.ProposalPatch](values)
}
func patchKey(patch *dto.ProposalPatch) string { return valueHash(patchValues(patch)) }
func coursePatch(course *dto.ProposalCourseVO) *dto.ProposalPatch {
	if course == nil {
		return nil
	}
	return &dto.ProposalPatch{Name: textPointer(course.Name), Code: textPointer(course.Code), Category: textPointer(course.Category), Department: textPointer(course.Department), Campuses: &course.Campuses, Teachers: &course.Teachers}
}
func teacherPatch(teacher *dto.TeacherVO) *dto.ProposalPatch {
	if teacher == nil {
		return nil
	}
	return &dto.ProposalPatch{Name: textPointer(teacher.Name), Title: textPointer(teacher.Title), Department: textPointer(teacher.Department)}
}
func teacherRelationValues(value interface{}) []string {
	teachers := copyAs[[]*dto.TeacherVO](value)
	values := []string{}
	if teachers == nil {
		return values
	}
	for _, teacher := range *teachers {
		if teacher == nil {
			values = append(values, "null")
		} else if teacher.ID != "" {
			values = append(values, "id:"+teacher.ID)
		} else {
			values = append(values, "new:"+valueHash(teacher))
		}
	}
	return normalizedSet(values)
}
func fieldEqual(field string, a, b interface{}) bool {
	if field == "teachers" {
		return valueHash(teacherRelationValues(a)) == valueHash(teacherRelationValues(b))
	}
	return valueHash(a) == valueHash(b)
}
func validatePatch(kind string, patch *dto.ProposalPatch) error {
	if patch == nil || len(patchValues(patch)) == 0 {
		return errorx.New(errno.ErrProposalNoChanges)
	}
	allowed := map[string]bool{"name": true, "department": true}
	if kind == model.ProposalUpdateTeacher {
		allowed["title"] = true
	} else if kind == model.ProposalUpdateCourse {
		for _, key := range []string{"code", "category", "campuses", "teachers"} {
			allowed[key] = true
		}
	} else {
		return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "type"))
	}
	for key, value := range patchValues(patch) {
		if !allowed[key] {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "suggested."+key))
		}
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if text == "" && !(kind == model.ProposalUpdateTeacher && (key == "title" || key == "department")) {
				return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "suggested."+key))
			}
			switch key {
			case "name":
				*patch.Name = text
			case "code":
				*patch.Code = text
			case "category":
				*patch.Category = text
			case "department":
				*patch.Department = text
			case "title":
				*patch.Title = text
			}
		}
	}
	if patch.Campuses != nil {
		values := normalizedSet(*patch.Campuses)
		patch.Campuses = &values
		if len(values) == 0 {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "campuses"))
		}
	}
	if patch.Teachers != nil {
		course := &dto.ProposalCourseVO{Teachers: *patch.Teachers}
		normalizeCourse(course)
		patch.Teachers = &course.Teachers
		for _, teacher := range course.Teachers {
			if teacher == nil || teacher.Name == "" {
				return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "teachers"))
			}
		}
	}
	return nil
}
func selectChanged(before, suggested *dto.ProposalPatch) (*dto.ProposalPatch, *dto.ProposalPatch) {
	originals := patchValues(before)
	changes := patchValues(suggested)
	base := map[string]interface{}{}
	for field, value := range changes {
		if fieldEqual(field, originals[field], value) {
			delete(changes, field)
		} else {
			base[field] = originals[field]
		}
	}
	return patchFromValues(base), patchFromValues(changes)
}
func proposalAutoKey(proposal *model.Proposal) string {
	if proposal.EffectiveType() == model.ProposalCreateCourse {
		return newCourseKey(copyAs[dto.ProposalCourseVO](proposal.Course))
	}
	values := patchValues(copyAs[dto.ProposalPatch](proposal.Suggested))
	if teachers, ok := values["teachers"]; ok {
		values["teachers"] = teacherRelationValues(teachers)
	}
	return valueHash([]interface{}{proposal.EffectiveType(), proposal.TargetID, values})
}
func courseDifferences(original, final *dto.ProposalCourseVO) []dto.ProposalDifference {
	result := []dto.ProposalDifference{}
	a, b := patchValues(coursePatch(original)), patchValues(coursePatch(final))
	for _, field := range []string{"name", "code", "department", "category", "campuses", "teachers"} {
		if !fieldEqual(field, a[field], b[field]) {
			result = append(result, dto.ProposalDifference{Field: field, Original: a[field], Suggested: b[field]})
		}
	}
	return result
}
func modificationScore(suggested, before, final *dto.ProposalPatch) int64 {
	a, b, c := patchValues(suggested), patchValues(before), patchValues(final)
	var score int64
	for field, value := range c {
		if !fieldEqual(field, b[field], value) {
			if score == 0 {
				score = 1
			}
			if fieldEqual(field, a[field], value) {
				score++
			}
		}
	}
	if score > 5 {
		score = 5
	}
	return score
}
