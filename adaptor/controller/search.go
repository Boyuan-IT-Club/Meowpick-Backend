package controller

import (
	common "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// GetSearchHistory 获得最近15条搜索历史
// @router /api/search/recent [GET]
func GetSearchHistory(c *gin.Context) {
	var err error
	var resp *cmd.GetSearchHistoriesResp

	c.Set(consts.ContextUserID, token.GetUserId(c))
	resp, err = provider.Get().SearchHistoryService.GetSearchHistoryByUserId(c)
	common.PostProcess(c, nil, resp, err)
}

// GetSearchSuggestions 输入框有文本更新时 显示搜索建议
// @router /api/search/suggest
func GetSearchSuggestions(c *gin.Context) {
	var err error
	var req cmd.GetSearchSuggestReq
	var resp *cmd.GetSearchSuggestResp
	if err = c.ShouldBindQuery(&req); err != nil {
		common.PostProcess(c, req, nil, err)
		return
	}
	resp, err = provider.Get().SearchService.GetSearchSuggestions(c, &req)
	common.PostProcess(c, &req, resp, err)
}

// ListCourses 用户点击🔍时模糊搜索课程，返回课程VO列表
// @router /api/search/course
func ListCourses(c *gin.Context) {
	var req cmd.ListCoursesReq
	var resp *cmd.ListCoursesResp
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		// 如果这里出错，err 就被赋值了。我们直接 return，
		// defer 会自动捕获这个 err 并处理错误响应。
		return
	}

	c.Set(consts.ContextUserID, token.GetUserId(c))

	if req.Keyword != "" {
		keyword := req.Keyword
		// 使用 gin.Context 的副本，安全传入 goroutine
		cCopy := c.Copy()
		go func() {
			if err := provider.Get().SearchHistoryService.LogSearch(cCopy, keyword); err != nil {
				log.CtxError(cCopy, "记录搜索历史失败: %v", err)
			}
		}()
	}

	resp, err = provider.Get().CourseService.ListCourses(c, &req)
	common.PostProcess(c, &req, resp, err)
}

// ListTeachers 用户点击🔍时模糊搜索，返回教师VO列表
// @router /api/search/teacher
func ListTeachers(c *gin.Context) {
	// TODO 实现接口
}
