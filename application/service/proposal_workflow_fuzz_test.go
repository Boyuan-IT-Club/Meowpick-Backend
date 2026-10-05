// Copyright 2026 Boyuan-IT-Club
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
	"encoding/json"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
)

func FuzzProposalValuesDoNotPanicOrChangeOnSecondNormalization(f *testing.F) {
	for _, seed := range []string{
		`null`, `{}`, `{"teachers":[null]}`, `{"campuses":[null," ","普陀校区"]}`,
		`{"name":"\u0000课程.*[]","teachers":[{"name":" 猫😀 ","id":"a"},{"name":"猫😀","id":"b"}]}`,
		`{"name":" 课程 ","code":"A","campuses":["二","一","一"],"teachers":[{"name":"老师"}]}`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 65536 {
			t.Skip()
		}
		var course dto.ProposalCourseVO
		if json.Unmarshal([]byte(data), &course) == nil {
			normalizeCourse(&course)
			first := valueHash(&course)
			normalizeCourse(&course)
			if valueHash(&course) != first {
				t.Fatal("course normalization not idempotent")
			}
			_ = newCourseKey(&course)
			_ = validateProposalInput("course", &course)
		}
		var patch dto.ProposalPatch
		if json.Unmarshal([]byte(data), &patch) == nil {
			for _, kind := range []string{model.ProposalUpdateCourse, model.ProposalUpdateTeacher} {
				candidate := copyAs[dto.ProposalPatch](&patch)
				if validatePatch(kind, candidate) == nil {
					first := valueHash(candidate)
					if err := validatePatch(kind, candidate); err != nil || valueHash(candidate) != first {
						t.Fatal("valid patch normalization not stable", err)
					}
					before, changes := selectChanged(candidate, candidate)
					if len(patchValues(before)) != 0 || len(patchValues(changes)) != 0 {
						t.Fatal("unchanged patch generated changes")
					}
				}
			}
		}
	})
}
