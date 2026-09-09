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
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
)

func TestReferencedMappingIDs(t *testing.T) {
	matches := []int32{4, 2, 9, 2}
	active := []int32{2, 3, 4}
	want := []int32{4, 2, 2}
	if got := referencedMappingIDs(matches, active); !reflect.DeepEqual(got, want) {
		t.Fatalf("referencedMappingIDs() = %v, want %v", got, want)
	}
}

func TestTeacherSuggestionRank(t *testing.T) {
	tests := []struct {
		name        string
		keyword     string
		teacherName string
		title       string
		want        int
	}{
		{name: "exact name", keyword: "张丹", teacherName: "张丹", title: "副教授", want: 0},
		{name: "exact search value", keyword: "张丹副教授", teacherName: "张丹", title: "副教授", want: 0},
		{name: "name prefix", keyword: "张", teacherName: "张丹", title: "副教授", want: 10},
		{name: "name substring", keyword: "丹", teacherName: "张丹", title: "副教授", want: 20},
		{name: "title substring", keyword: "副教授", teacherName: "张丹", title: "副教授", want: 21},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := teacherSuggestionRank(tt.keyword, tt.teacherName, tt.title); got != tt.want {
				t.Fatalf("teacherSuggestionRank() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGroupTeacherSuggestionCandidatesMergesSearchValue(t *testing.T) {
	teachers := []*model.Teacher{
		{ID: "b", Name: "张丹", Title: "副教授"},
		{ID: "a", Name: "张丹", Title: "副教授"},
		{ID: "c", Name: "张丹副", Title: "教授"},
		{ID: "d", Name: "张丹", Title: "讲师"},
	}
	groups := groupTeacherSuggestionCandidates(teachers, "张丹")
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if got := groups[0].Teacher.Name + groups[0].Teacher.Title; got != "张丹副教授" {
		t.Fatalf("first search value = %q, want 张丹副教授", got)
	}
	if groups[0].Teacher.ID != "a" {
		t.Fatalf("representative ID = %q, want a", groups[0].Teacher.ID)
	}
	wantIDs := []string{"a", "b", "c"}
	if !reflect.DeepEqual(groups[0].TeacherIDs, wantIDs) {
		t.Fatalf("teacher IDs = %v, want %v", groups[0].TeacherIDs, wantIDs)
	}
}

func TestSortAndPageSearchSuggestionsUsesRelevanceBeforeType(t *testing.T) {
	candidates := []*rankedSearchSuggestion{
		{VO: &dto.SearchSuggestionsVO{Type: consts.SuggestionTargetTypeCourse, Name: "课程张丹", SearchValue: "课程张丹"}, Rank: 20, TypePriority: 0},
		{VO: &dto.SearchSuggestionsVO{Type: consts.SuggestionTargetTypeTeacher, Name: "张丹", Title: "副教授", SearchValue: "张丹副教授"}, Rank: 10, TypePriority: 1},
		{VO: &dto.SearchSuggestionsVO{Type: consts.SuggestionTargetTypeCourse, Name: "张丹", SearchValue: "张丹"}, Rank: 0, TypePriority: 0},
	}
	got := sortAndPageSearchSuggestions(candidates, &dto.PageParam{Page: 1, PageSize: 2})
	if len(got) != 2 || got[0].SearchValue != "张丹" || got[1].SearchValue != "张丹副教授" {
		t.Fatalf("suggestions = %#v", got)
	}
}

func TestRecentCourseBriefsForTeacherIDsMergesAndSorts(t *testing.T) {
	now := time.Now()
	shared := &model.Course{ID: "shared", Name: "共享课", CreatedAt: now.Add(-time.Hour)}
	byTeacher := map[string][]*model.Course{
		"a": {
			{ID: "old", Name: "旧课", CreatedAt: now.Add(-2 * time.Hour)},
			shared,
		},
		"b": {
			{ID: "new", Name: "新课", CreatedAt: now},
			shared,
		},
	}
	got := recentCourseBriefsForTeacherIDs([]string{"a", "b"}, byTeacher, 2)
	want := []dto.CourseBrief{{ID: "new", Name: "新课"}, {ID: "shared", Name: "共享课"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recentCourseBriefsForTeacherIDs() = %#v, want %#v", got, want)
	}
}
