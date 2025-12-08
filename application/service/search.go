package service

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
	"golang.org/x/sync/errgroup"
)

var _ ISearchService = (*SearchService)(nil)

type ISearchService interface {
	GetSearchSuggestions(ctx context.Context, req *dto.GetSearchSuggestReq) (*dto.GetSearchSuggestResp, error)
	ListCoursesByType(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error)
}

type SearchService struct {
	CourseMapper  *course.MongoRepo
	TeacherMapper *teacher.MongoRepo
	StaticData    *mapping.StaticData
	CourseDTO     *assembler.CourseDTO
}

var SearchServiceSet = wire.NewSet(
	wire.Struct(new(SearchService), "*"),
	wire.Bind(new(ISearchService), new(*SearchService)),
)

func (s *SearchService) GetSearchSuggestions(ctx context.Context, req *dto.GetSearchSuggestReq) (*dto.GetSearchSuggestResp, error) { // 定义四个任务，每个任务返回其结果和可能的错误
	tasks := []func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error){
		// Courses
		func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error) {
			courseModels, err := s.CourseMapper.GetCourseSuggestions(ctx, req.Keyword, req.PageParam)
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
			teacherModels, err := s.TeacherMapper.GetTeacherSuggestions(ctx, req.Keyword, req.PageParam)
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
			ids := s.StaticData.GetCategoryIDsByKeyword(req.Keyword)
			var out []*dto.SearchSuggestionsVO
			for _, id := range ids {
				name := s.StaticData.GetCategoryNameByID(id)
				out = append(out, &dto.SearchSuggestionsVO{
					Type: "category",
					Name: name,
				})
			}
			return out, nil
		},
		// Departments
		func(ctx context.Context) ([]*dto.SearchSuggestionsVO, error) {
			ids := s.StaticData.GetDepartmentIDsByKeyword(req.Keyword)
			var out []*dto.SearchSuggestionsVO
			for _, id := range ids {
				name := s.StaticData.GetDepartmentNameByID(id)
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

	return &dto.GetSearchSuggestResp{
		Resp:        dto.Success(),
		Suggestions: suggestions,
	}, nil
}

func (s *SearchService) ListCoursesByType(ctx context.Context, req *dto.ListCoursesReq) (*dto.ListCoursesResp, error) {
	// 将关键词转化为最匹配的种类id
	var typeId int32
	if req.Type == consts.Category {
		typeId = s.StaticData.GetBestCategoryIDByKeyword(req.Keyword)
	} else if req.Type == consts.Department {
		typeId = s.StaticData.GetBestDepartmentIDByKeyword(req.Keyword)
	}

	// 如果转化结果为0，说明关键词搜不到，直接返回
	if typeId == 0 {
		return &dto.ListCoursesResp{
			Resp:             dto.Success(),
			PaginatedCourses: nil,
		}, errorx.ErrFindSuccessButNoResult
	}

	var dbCourses []*course.Course
	var total int64
	var err error
	// 根据type决定查询数据库的方式
	if req.Type == consts.Category {
		dbCourses, total, err = s.CourseMapper.FindCoursesByCategoryID(ctx, typeId, req.PageParam)
	} else if req.Type == consts.Department {
		dbCourses, total, err = s.CourseMapper.FindCoursesByDepartmentID(ctx, typeId, req.PageParam)
	}
	// 检查结果
	if err != nil {
		log.Error("FindCoursesByCategory err", err)
		return nil, errorx.ErrFindFailed
	}
	if total == 0 {
		return nil, errorx.ErrFindSuccessButNoResult
	}
	// dto转化
	paginatedCourses, err := s.CourseDTO.ToPaginatedCourses(ctx, dbCourses, total, req.PageParam)
	if err != nil {
		log.CtxError(ctx, "CourseDB To CourseVO error: %v", err)
		return nil, errorx.ErrCourseDB2VO
	}

	return &dto.ListCoursesResp{
		Resp:             dto.Success(),
		PaginatedCourses: paginatedCourses,
	}, nil
}
