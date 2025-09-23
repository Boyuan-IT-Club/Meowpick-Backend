package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/google/wire"
)

type ISearchService interface {
	GetSearchSuggestions(ctx context.Context, req *cmd.GetSearchSuggestReq) (*cmd.GetSearchSuggestResp, error)
	ListCoursesByType(ctx context.Context, req *cmd.ListCoursesReq) (*cmd.ListCoursesResp, error)
}

type SearchService struct {
	CourseMapper  *course.MongoMapper
	TeacherMapper *teacher.MongoMapper
	StaticData    *consts.StaticData
	CourseDTO     *dto.CourseDTO
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

func (s *SearchService) ListCoursesByType(ctx context.Context, req *cmd.ListCoursesReq) (*cmd.ListCoursesResp, error) {
	// 将关键词转化为最匹配的种类id
	var typeId int32
	if req.Type == consts.Category {
		typeId = s.StaticData.GetBestCategoryIDByKeyword(req.Keyword)
	} else if req.Type == consts.Department {
		typeId = s.StaticData.GetBestDepartmentIDByKeyword(req.Keyword)
	}

	// 如果转化结果为0，说明关键词搜不到，直接返回
	if typeId == 0 {
		return &cmd.ListCoursesResp{
			Resp:             cmd.Success(),
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

	return &cmd.ListCoursesResp{
		Resp:             cmd.Success(),
		PaginatedCourses: paginatedCourses,
	}, nil
}
