package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/google/wire"
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
	courseTotal, err := s.CourseMapper.CountCourses(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	targetPage := req.Page
	targetSize := req.PageSize

	offset := (targetPage - 1) * targetSize

	allSuggestions := make([]*cmd.SearchSuggestionsVO, 0, targetSize)

	if offset < courseTotal { //请求之始为课程

		courseModels, err2 := s.CourseMapper.GetCourseSuggestions(ctx, req.Keyword, req.PageParam)
		if err2 != nil {
			return nil, err2
		}

		for _, model := range courseModels {
			allSuggestions = append(allSuggestions, &cmd.SearchSuggestionsVO{
				Type: "course",
				Name: model.Name,
			})
		}

		if int64(len(allSuggestions)) < targetSize {
			teachersNeeded := targetSize - int64(len(allSuggestions))
			param := &cmd.PageParam{
				Page:     1,
				PageSize: teachersNeeded,
			}

			teacherModels, _ := s.TeacherMapper.GetTeacherSuggestions(ctx, req.Keyword, param)

			for _, model := range teacherModels {
				allSuggestions = append(allSuggestions, &cmd.SearchSuggestionsVO{
					Type: "teacher",
					Name: model.Name,
				})
			}
		}

	} else {
		//请求的数据完全在老师列表内
		teacherOffset := offset - courseTotal
		teacherPageNum := teacherOffset/targetSize + 1
		param := &cmd.PageParam{
			Page:     teacherPageNum,
			PageSize: targetSize,
		}

		teacherModels, _ := s.TeacherMapper.GetTeacherSuggestions(ctx, req.Keyword, param)

		for _, model := range teacherModels {
			allSuggestions = append(allSuggestions, &cmd.SearchSuggestionsVO{
				Type: "teacher",
				Name: model.Name,
			})
		}
	}

	// 组装并返回响应
	response := &cmd.GetSearchSuggestResp{
		Resp:        cmd.Success(),
		Suggestions: allSuggestions,
	}

	return response, nil
}
