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
	"sort"
	"strings"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
)

const noSuggestionMatch = int(^uint(0) >> 1)
const maxSuggestionCandidatesPerType int64 = 1000

type teacherSuggestionGroup struct {
	Teacher    *model.Teacher
	TeacherIDs []string
	Rank       int
}

type rankedSearchSuggestion struct {
	VO           *dto.SearchSuggestionsVO
	Rank         int
	TypePriority int
}

func searchSuggestionCandidateLimit(param *dto.PageParam) int64 {
	if param == nil {
		return 10
	}
	pageNumber, pageSize := param.UnWrap()
	if pageNumber > maxSuggestionCandidatesPerType/pageSize {
		return maxSuggestionCandidatesPerType
	}
	limit := pageNumber * pageSize
	if limit > maxSuggestionCandidatesPerType {
		return maxSuggestionCandidatesPerType
	}
	return limit
}

func textSuggestionRank(keyword, value string) int {
	keyword = strings.ToLower(keyword)
	value = strings.ToLower(value)
	switch {
	case keyword == value:
		return 0
	case strings.HasPrefix(value, keyword):
		return 10
	case strings.Contains(value, keyword):
		return 20
	default:
		return noSuggestionMatch
	}
}

func teacherSuggestionRank(keyword, name, title string) int {
	searchValue := name + title
	if rank := textSuggestionRank(keyword, name); rank == 0 {
		return 0
	}
	if rank := textSuggestionRank(keyword, searchValue); rank == 0 {
		return 1 // An exact name match outranks an exact combined name-and-title match.
	}
	if rank := textSuggestionRank(keyword, name); rank != noSuggestionMatch {
		return rank
	}
	if rank := textSuggestionRank(keyword, searchValue); rank != noSuggestionMatch {
		return rank + 1 // A name match outranks a title-only/combined-value match.
	}
	return noSuggestionMatch
}

func groupTeacherSuggestionCandidates(teachers []*model.Teacher, keyword string) []*teacherSuggestionGroup {
	bySearchValue := make(map[string]*teacherSuggestionGroup, len(teachers))
	for _, teacher := range teachers {
		if teacher == nil {
			continue
		}
		searchValue := teacher.Name + teacher.Title
		rank := teacherSuggestionRank(keyword, teacher.Name, teacher.Title)
		if rank == noSuggestionMatch {
			continue
		}
		group := bySearchValue[searchValue]
		if group == nil {
			group = &teacherSuggestionGroup{Teacher: teacher, Rank: rank}
			bySearchValue[searchValue] = group
		}
		group.TeacherIDs = append(group.TeacherIDs, teacher.ID)
		if rank < group.Rank || (rank == group.Rank && teacher.ID < group.Teacher.ID) {
			group.Teacher = teacher
			group.Rank = rank
		}
	}

	groups := make([]*teacherSuggestionGroup, 0, len(bySearchValue))
	for _, group := range bySearchValue {
		sort.Strings(group.TeacherIDs)
		groups = append(groups, group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Rank != groups[j].Rank {
			return groups[i].Rank < groups[j].Rank
		}
		left := groups[i].Teacher.Name + groups[i].Teacher.Title
		right := groups[j].Teacher.Name + groups[j].Teacher.Title
		if left != right {
			return left < right
		}
		return groups[i].Teacher.ID < groups[j].Teacher.ID
	})
	return groups
}

func pageTeacherSuggestionGroups(groups []*teacherSuggestionGroup, param *dto.PageParam) []*teacherSuggestionGroup {
	if param == nil {
		param = &dto.PageParam{Page: 1, PageSize: 10}
	}
	pageNumber, pageSize := param.UnWrap()
	start := (pageNumber - 1) * pageSize
	if start >= int64(len(groups)) {
		return []*teacherSuggestionGroup{}
	}
	end := start + pageSize
	if end > int64(len(groups)) {
		end = int64(len(groups))
	}
	return groups[start:end]
}

func recentCourseBriefsForTeacherIDs(
	teacherIDs []string,
	coursesByTeacher map[string][]*model.Course,
	limit int,
) []dto.CourseBrief {
	byID := make(map[string]*model.Course)
	for _, teacherID := range teacherIDs {
		for _, course := range coursesByTeacher[teacherID] {
			if course != nil {
				byID[course.ID] = course
			}
		}
	}
	courses := make([]*model.Course, 0, len(byID))
	for _, course := range byID {
		courses = append(courses, course)
	}
	sort.SliceStable(courses, func(i, j int) bool {
		if !courses[i].CreatedAt.Equal(courses[j].CreatedAt) {
			return courses[i].CreatedAt.After(courses[j].CreatedAt)
		}
		return courses[i].ID < courses[j].ID
	})
	if len(courses) > limit {
		courses = courses[:limit]
	}
	briefs := make([]dto.CourseBrief, 0, len(courses))
	for _, course := range courses {
		briefs = append(briefs, dto.CourseBrief{ID: course.ID, Name: course.Name})
	}
	return briefs
}

func searchSuggestionTypePriority(suggestionType string) int {
	switch suggestionType {
	case consts.SuggestionTargetTypeCourse:
		return 0
	case consts.SuggestionTargetTypeTeacher:
		return 1
	case consts.SuggestionTargetTypeCategory:
		return 2
	case consts.SuggestionTargetTypeDepartment:
		return 3
	default:
		return 4
	}
}

func sortAndPageSearchSuggestions(candidates []*rankedSearchSuggestion, param *dto.PageParam) []*dto.SearchSuggestionsVO {
	unique := make(map[string]*rankedSearchSuggestion, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.VO == nil {
			continue
		}
		key := candidate.VO.Type + "\x00" + candidate.VO.SearchValue
		current := unique[key]
		if current == nil || candidate.Rank < current.Rank {
			unique[key] = candidate
		}
	}

	ranked := make([]*rankedSearchSuggestion, 0, len(unique))
	for _, candidate := range unique {
		ranked = append(ranked, candidate)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Rank != ranked[j].Rank {
			return ranked[i].Rank < ranked[j].Rank
		}
		if ranked[i].TypePriority != ranked[j].TypePriority {
			return ranked[i].TypePriority < ranked[j].TypePriority
		}
		return ranked[i].VO.SearchValue < ranked[j].VO.SearchValue
	})

	pageNumber, pageSize := param.UnWrap()
	start := (pageNumber - 1) * pageSize
	if start >= int64(len(ranked)) {
		return []*dto.SearchSuggestionsVO{}
	}
	end := start + pageSize
	if end > int64(len(ranked)) {
		end = int64(len(ranked))
	}
	result := make([]*dto.SearchSuggestionsVO, 0, end-start)
	for _, candidate := range ranked[start:end] {
		result = append(result, candidate.VO)
	}
	return result
}
