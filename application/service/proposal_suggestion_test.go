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

import "testing"

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
