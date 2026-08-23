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
			ID: "teacher-1", Value: "张三", Label: "张三 - 教授", Courses: &courses,
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
	})
}
