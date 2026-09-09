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

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSearchSuggestionsVOJSON(t *testing.T) {
	teacherPayload, err := json.Marshal(SearchSuggestionsVO{
		Type:        "teacher",
		Name:        "张丹",
		Title:       "副教授",
		SearchValue: "张丹副教授",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"name":"张丹"`, `"title":"副教授"`, `"searchValue":"张丹副教授"`} {
		if !strings.Contains(string(teacherPayload), field) {
			t.Fatalf("teacher payload %s does not contain %s", teacherPayload, field)
		}
	}

	coursePayload, err := json.Marshal(SearchSuggestionsVO{Type: "course", Name: "测试课程", SearchValue: "测试课程"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(coursePayload), `"title"`) || !strings.Contains(string(coursePayload), `"searchValue":"测试课程"`) {
		t.Fatalf("course payload = %s", coursePayload)
	}
}
