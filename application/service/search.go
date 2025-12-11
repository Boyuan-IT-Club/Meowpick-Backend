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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/google/wire"
	"golang.org/x/sync/errgroup"
)

var _ ISearchService = (*SearchService)(nil)

type ISearchService interface {
	GetSearchSuggestions(ctx context.Context, req *dto.GetSearchSuggestionsReq) (*dto.GetSearchSuggestionsResp, error)
}

type SearchService struct {
	CourseRepo      *repo.CourseRepo
	TeacherRepo     *repo.TeacherRepo
	CourseAssembler *assembler.CourseAssembler
}

var SearchServiceSet = wire.NewSet(
	wire.Struct(new(SearchService), "*"),
	wire.Bind(new(ISearchService), new(*SearchService)),
)

func (s *SearchService) GetSearchSuggestions(ctx context.Context, req *dto.GetSearchSuggestionsReq) (*dto.GetSearchSuggestionsResp, error) { // 定义四个任务，每个任务返回其结果和可能的错误
	tasks := []func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error){
		// Courses
		func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error) {
			courseModels, err := s.CourseRepo.GetSuggestions(ctx, req.Keyword, req.PageParam)
			if err != nil {
				// 返回错误，errgroup 会捕获它
				return nil, err
			}
			var out []*dto.SearchSuggestionsVO
			for _, model := range courseModels {
				out = append(out, &dto.SearchSuggestionsVO{
					Type: "course",
					Name: model.Name,
				})
			}
			return out, nil
		},
		// Teachers
		func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error) {
			teacherModels, err := s.TeacherRepo.GetSuggestions(ctx, req.Keyword, req.PageParam)
			if err != nil {
				// 返回错误
				return nil, err
			}
			var out []*dto.SearchSuggestionsVO
			for _, model := range teacherModels {
				out = append(out, &dto.SearchSuggestionsVO{
					Type: "teacher",
					Name: model.Name,
				})
			}
			return out, nil
		},
		// Categories
		func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error) {
			ids := mapping.Data.GetCategoryIDsByKeyword(req.Keyword)
			var out []*dto.SearchSuggestionsVO
			for _, id := range ids {
				name := mapping.Data.GetCategoryNameByID(id)
				out = append(out, &dto.SearchSuggestionsVO{
					Type: "category",
					Name: name,
				})
			}
			return out, nil
		},
		// Departments
		func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error) {
			ids := mapping.Data.GetDepartmentIDsByKeyword(req.Keyword)
			var out []*dto.SearchSuggestionsVO
			for _, id := range ids {
				name := mapping.Data.GetDepartmentNameByID(id)
				out = append(out, &dto.SearchSuggestionsVO{
					Type: "department",
					Name: name,
				})
			}
			return out, nil
		},
	}

	n := len(tasks)
	results := make([][]*dto.SearchSuggestionsVO, n)

	// 创建一个 errgroup.Group
	g, ctx := errgroup.WithContext(ctx)

	// 启动 goroutine，使用 g.Go
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

	// 等待所有 goroutine 完成
	// g.Wait() 会阻塞直到所有任务都完成。 如果任何一个任务返回了非 nil 的 error，g.Wait() 会返回这个 error， 并且自动取消其他正在运行的任务。
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 合并结果（保持顺序）
	var suggestions []*dto.SearchSuggestionsVO
	for i := 0; i < n; i++ {
		if results[i] != nil {
			suggestions = append(suggestions, results[i]...)
		}
		if int64(len(suggestions)) >= req.PageSize {
			suggestions = suggestions[:req.PageSize]
			break
		}
	}

	return &dto.GetSearchSuggestionsResp{
		Resp:        dto.Success(),
		Suggestions: suggestions,
	}, nil
}
