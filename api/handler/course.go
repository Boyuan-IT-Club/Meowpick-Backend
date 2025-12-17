package handler

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/gin-gonic/gin"
)

// GetCourse godoc
// @Summary 获取课程信息
// @Description 获取课程信息
// @Tags course
// @Produce json
// @Param courseId path string true "课程ID"
// @Success 200 {object} dto.GetCourseResp
// @Router /api/course/{courseId} [get]
func GetCourse(c *gin.Context) {
	var req dto.GetCourseReq
	var resp *dto.GetCourseResp
	var err error

	req.CourseID = c.Param(consts.CtxCourseID)
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetCourse(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseDepartments godoc
// @Summary 获取课程开课院系
// @Description 根据课程名字获取课程开课院系
// @Tags course
// @Produce json
// @Param keyword query string true "课程名称关键词"
// @Success 200 {object} dto.GetCourseDepartmentsResp
// @Router /api/course/departs [get]
func GetCourseDepartments(c *gin.Context) {
	var req dto.GetCourseDepartmentsReq
	var resp *dto.GetCourseDepartmentsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetDepartments(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseCategories godoc
// @Summary 获取课程分类
// @Description 根据课程名字获取课程分类
// @Tags course
// @Produce json
// @Param keyword query string true "课程名称关键词"
// @Success 200 {object} dto.GetCourseCategoriesResp
// @Router /api/course/categories [get]
func GetCourseCategories(c *gin.Context) {
	var req dto.GetCourseCategoriesReq
	var resp *dto.GetCourseCategoriesResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetCategories(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetCourseCampuses godoc
// @Summary 获取课程开课校区
// @Description 根据课程名字获取课程开课校区
// @Tags course
// @Produce json
// @Param keyword query string true "课程名称关键词"
// @Success 200 {object} dto.GetCourseCampusesResp
// @Router /api/course/campuses [get]
func GetCourseCampuses(c *gin.Context) {
	var req dto.GetCourseCampusesReq
	var resp *dto.GetCourseCampusesResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CourseService.GetCampuses(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListCourses godoc
// @Summary 搜索课程列表
// @Description 搜索课程列表
// @Tags courses
// @Accept json
// @Produce json
// @Param body body dto.ListCoursesReq true "ListCoursesReq"
// @Success 200 {object} dto.ListCoursesResp
// @Router /api/search [post]
func ListCourses(c *gin.Context) {
	var req dto.ListCoursesReq
	var resp *dto.ListCoursesResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	if req.Keyword != "" {
		go func() {
			cCopy := c.Copy()
			if errCopy := provider.Get().SearchHistoryService.LogSearch(cCopy, req.Keyword); errCopy != nil {
				logs.CtxErrorf(cCopy, "[SearchHistoryService] [LogSearch] error: %v", errCopy)
			}
		}()
	}

	resp, err = provider.Get().CourseService.ListCourses(c, &req)
	PostProcess(c, &req, resp, err)
}
