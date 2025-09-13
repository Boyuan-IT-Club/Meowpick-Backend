package service

import (
	"context"
	"github.com/google/wire"
	"sync" // <-- 导入 sync 包用于并发控制

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
)

type ISearchService interface {
	GetSearchSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) (*cmd.GetSearchSuggestResp, error)
}

type SearchService struct {
	CourseMapper  *course.MongoMapper
	TeacherMapper *teacher.MongoMapper
}

var SearchServiceSet = wire.NewSet(
	wire.Struct(new(SearchService), "*"),
	wire.Bind(new(ISearchService), new(*SearchService)),
)

func (s *SearchService) GetSearchSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) (*cmd.GetSearchSuggestResp, error) {
	var wg sync.WaitGroup // 创建一个 WaitGroup 来等待所有查询完成

	var courseSuggestions []*cmd.SearchSuggestionsVO
	var teacherSuggestions []*cmd.SearchSuggestionsVO

	// 启动一个 goroutine 去查询课程建议
	wg.Add(1) // 任务计数器+1
	go func() {
		defer wg.Done()
		courseModels, _ := s.CourseMapper.GetCourseSuggestions(ctx, req)
		suggestions := make([]*cmd.SearchSuggestionsVO, 0, len(courseModels))
		for _, model := range courseModels {
			suggestions = append(suggestions, &cmd.SearchSuggestionsVO{
				Type: "课程",
				Name: model.Name,
			})
		}
		courseSuggestions = suggestions
	}()

	// 启动另一个 goroutine 去查询老师建议
	wg.Add(1) // 任务计数器+1
	go func() {
		defer wg.Done()
		teacherModels, _ := s.TeacherMapper.GetTeacherSuggestions(ctx, req)
		suggestions := make([]*cmd.SearchSuggestionsVO, 0, len(teacherModels))
		for _, model := range teacherModels {
			suggestions = append(suggestions, &cmd.SearchSuggestionsVO{
				Type: "教师",
				Name: model.Name,
			})
		}
		teacherSuggestions = suggestions
	}()

	// 等待上面所有 goroutine 执行完毕
	wg.Wait()

	// 聚合所有结果
	allSuggestions := make([]*cmd.SearchSuggestionsVO, 0, len(courseSuggestions)+len(teacherSuggestions))
	allSuggestions = append(allSuggestions, courseSuggestions...)
	allSuggestions = append(allSuggestions, teacherSuggestions...)

	response := &cmd.GetSearchSuggestResp{
		Resp:        cmd.Success(),
		Suggestions: allSuggestions,
	}

	return response, nil
}
