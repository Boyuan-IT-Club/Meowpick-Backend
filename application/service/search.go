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
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"golang.org/x/sync/errgroup"
)

var _ ISearchService = (*SearchService)(nil)

type ISearchService interface {
	GetSearchSuggestions(ctx context.Context, req *dto.GetSearchSuggestionsReq) (*dto.GetSearchSuggestionsResp, error)
}

type SearchService struct {
	CourseRepo  *repo.CourseRepo
	TeacherRepo *repo.TeacherRepo
}

var SearchServiceSet = wire.NewSet(
	wire.Struct(new(SearchService), "*"),
	wire.Bind(new(ISearchService), new(*SearchService)),
)

// GetSearchSuggestions 并行获取搜索建议
func (s *SearchService) GetSearchSuggestions(ctx context.Context, req *dto.GetSearchSuggestionsReq) (*dto.GetSearchSuggestionsResp, error) {
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	if req.PageParam == nil {
		req.PageParam = &dto.PageParam{Page: 1, PageSize: 10}
	}
	candidateLimit := searchSuggestionCandidateLimit(req.PageParam)

	// 各类型先独立召回，随后统一按相关度排序和分页，避免固定类型顺序挤占结果。
	tasks := []func(ctx context.Context) ([]*rankedSearchSuggestion, error){
		func(ctx context.Context) ([]*rankedSearchSuggestion, error) {
			courses, err := s.CourseRepo.FindSearchSuggestionCandidates(ctx, req.Keyword, candidateLimit)
			if err != nil {
				logs.CtxErrorf(ctx, "[CourseRepo] [FindSearchSuggestionCandidates] error: %v", err)
				return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
					errorx.KV("keyword", req.Keyword))
			}
			result := make([]*rankedSearchSuggestion, 0, len(courses))
			for _, course := range courses {
				result = append(result, &rankedSearchSuggestion{
					VO: &dto.SearchSuggestionsVO{
						Type:        consts.SuggestionTargetTypeCourse,
						Name:        course.Name,
						SearchValue: course.Name,
					},
					Rank:         textSuggestionRank(req.Keyword, course.Name),
					TypePriority: searchSuggestionTypePriority(consts.SuggestionTargetTypeCourse),
				})
			}
			return result, nil
		},
		func(ctx context.Context) ([]*rankedSearchSuggestion, error) {
			teacherIDs, err := s.CourseRepo.FindActiveTeacherIDs(ctx)
			if err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
					errorx.KV("keyword", req.Keyword))
			}
			teachers, err := s.TeacherRepo.FindSuggestionCandidates(ctx, req.Keyword, teacherIDs)
			if err != nil {
				logs.CtxErrorf(ctx, "[TeacherRepo] [FindSuggestionCandidates] error: %v", err)
				return nil, errorx.WrapByCode(err, errno.ErrTeacherGetSuggestionsFailed,
					errorx.KV("keyword", req.Keyword))
			}
			groups := groupTeacherSuggestionCandidates(teachers, req.Keyword)
			result := make([]*rankedSearchSuggestion, 0, len(groups))
			for _, group := range groups {
				teacher := group.Teacher
				result = append(result, &rankedSearchSuggestion{
					VO: &dto.SearchSuggestionsVO{
						Type:        consts.SuggestionTargetTypeTeacher,
						Name:        teacher.Name,
						Title:       teacher.Title,
						SearchValue: teacher.Name + teacher.Title,
					},
					Rank:         group.Rank,
					TypePriority: searchSuggestionTypePriority(consts.SuggestionTargetTypeTeacher),
				})
			}
			return result, nil
		},
		func(ctx context.Context) ([]*rankedSearchSuggestion, error) {
			activeIDs, err := s.CourseRepo.FindActiveCategoryIDs(ctx)
			if err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
					errorx.KV("keyword", req.Keyword))
			}
			ids := referencedMappingIDs(mapping.Data.GetCategoryIDsByKeyword(req.Keyword), activeIDs)
			result := make([]*rankedSearchSuggestion, 0, len(ids))
			for _, id := range ids {
				name := mapping.Data.GetCategoryNameByID(id)
				result = append(result, &rankedSearchSuggestion{
					VO: &dto.SearchSuggestionsVO{
						Type:        consts.SuggestionTargetTypeCategory,
						Name:        name,
						SearchValue: name,
					},
					Rank:         textSuggestionRank(req.Keyword, name),
					TypePriority: searchSuggestionTypePriority(consts.SuggestionTargetTypeCategory),
				})
			}
			return result, nil
		},
		func(ctx context.Context) ([]*rankedSearchSuggestion, error) {
			activeIDs, err := s.CourseRepo.FindActiveDepartmentIDs(ctx)
			if err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrCourseGetSuggestionsFailed,
					errorx.KV("keyword", req.Keyword))
			}
			ids := referencedMappingIDs(mapping.Data.GetDepartmentIDsByKeyword(req.Keyword), activeIDs)
			result := make([]*rankedSearchSuggestion, 0, len(ids))
			for _, id := range ids {
				name := mapping.Data.GetDepartmentNameByID(id)
				result = append(result, &rankedSearchSuggestion{
					VO: &dto.SearchSuggestionsVO{
						Type:        consts.SuggestionTargetTypeDepartment,
						Name:        name,
						SearchValue: name,
					},
					Rank:         textSuggestionRank(req.Keyword, name),
					TypePriority: searchSuggestionTypePriority(consts.SuggestionTargetTypeDepartment),
				})
			}
			return result, nil
		},
	}
	results := make([][]*rankedSearchSuggestion, len(tasks))

	g, ctx := errgroup.WithContext(ctx)
	for i, task := range tasks {
		i, task := i, task
		g.Go(func() error {
			suggestions, err := task(ctx)
			if err != nil {
				return err
			}
			results[i] = suggestions
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var candidates []*rankedSearchSuggestion
	for _, result := range results {
		candidates = append(candidates, result...)
	}

	return &dto.GetSearchSuggestionsResp{
		Resp:        dto.Success(),
		Suggestions: sortAndPageSearchSuggestions(candidates, req.PageParam),
	}, nil
}

func referencedMappingIDs(matches, active []int32) []int32 {
	activeSet := make(map[int32]struct{}, len(active))
	for _, id := range active {
		activeSet[id] = struct{}{}
	}
	result := make([]int32, 0, len(matches))
	for _, id := range matches {
		if _, ok := activeSet[id]; ok {
			result = append(result, id)
		}
	}
	return result
}
