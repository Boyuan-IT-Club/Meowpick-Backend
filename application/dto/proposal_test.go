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
	"testing"
)

func TestFieldSuggestionVOCoursesJSON(t *testing.T) {
	t.Run("non-teacher suggestion omits courses", func(t *testing.T) {
		payload, err := json.Marshal(FieldSuggestionVO{ID: "1", Value: "高等数学", Label: "高等数学"})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err = json.Unmarshal(payload, &got); err != nil {
			t.Fatal(err)
		}
		if _, exists := got["courses"]; exists {
			t.Fatalf("courses should be omitted: %s", payload)
		}
	})

	t.Run("teacher without courses emits empty array", func(t *testing.T) {
		courses := []CourseBrief{}
		payload, err := json.Marshal(FieldSuggestionVO{
			ID: "teacher-1", Value: "张三", Label: "张三 - 教授", Title: "教授", Courses: &courses,
		})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err = json.Unmarshal(payload, &got); err != nil {
			t.Fatal(err)
		}
		value, exists := got["courses"]
		if !exists {
			t.Fatalf("courses should be present: %s", payload)
		}
		items, ok := value.([]any)
		if !ok || len(items) != 0 {
			t.Fatalf("courses = %#v, want empty array", value)
		}
		if got["title"] != "教授" {
			t.Fatalf("title = %#v, want 教授", got["title"])
		}
	})
}

func TestListProposalRespJSONContract(t *testing.T) {
	payload, err := json.Marshal(ListProposalResp{
		Resp:  Success(),
		Total: 1,
		Proposals: []*ProposalVO{{
			ID:    "proposal-1",
			Title: "测试提案",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err = json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got["total"] != float64(1) {
		t.Fatalf("total = %#v, want 1", got["total"])
	}
	proposals, ok := got["proposals"].([]any)
	if !ok || len(proposals) != 1 {
		t.Fatalf("proposals = %#v, want one item", got["proposals"])
	}
	if _, exists := got["suggestions"]; exists {
		t.Fatalf("legacy suggestions field should not be present: %s", payload)
	}
}
