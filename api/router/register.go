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

package router

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/handler"
	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	router := gin.Default()

	// CommentApi
	commentGroup := router.Group("/api/comment")
	{
		commentGroup.POST("/add", handler.CreateComment)       // 发布评论
		commentGroup.GET("/query", handler.ListCourseComments) // 分页获取课程下的评论
		commentGroup.POST("/history", handler.GetMyComments)   // 获得我的吐槽
	}

	// SearchApi
	searchGroup := router.Group("/api/search")
	{
		searchGroup.GET("/recent", handler.GetSearchHistories)         // 搜索历史
		searchGroup.POST("", handler.ListCourses)                      // 模糊搜索展示课程列表
		searchGroup.GET("/total", handler.GetTotalCourseCommentsCount) // 小程序初始化界面的总吐槽数
		searchGroup.GET("/suggest", handler.GetSearchSuggestions)      // 用户输入搜索内容期间获得搜索建议
	}

	// AuthApi
	authGroup := router.Group("/api/auth")
	authGroup.POST("/sign_in", handler.SignIn) // 初始化时的登录、授权

	// LikeApi
	likeGroup := router.Group("/api/like")
	likeGroup.POST("/:likeId", handler.ToggleLike) // 为评论点赞

	// CourseApi
	courseGroup := router.Group("/api/course")
	{
		courseGroup.GET("/:courseId", handler.GetCourse)              // 精确搜索某个课程
		courseGroup.GET("/departments", handler.GetCourseDepartments) // 获得某课程的“所属部门”信息
		courseGroup.GET("/categories", handler.GetCourseCategories)   // 获得某课程的“课程类型”信息
		courseGroup.GET("/campuses", handler.GetCourseCampuses)       // 获得某课程的“开设校区”信息
	}

	// TeacherApi
	//teacherGroup := router.Group("/api/teacher")
	//{
	//}

	// ProposalApi
	proposalGroup := router.Group("/api/proposal")
	{
		proposalGroup.POST("/add", handler.CreateProposal)
		proposalGroup.GET("/list", handler.ListProposals)
		proposalGroup.GET("/:proposalId", handler.GetProposal)
		proposalGroup.POST("/:proposalId/update", handler.UpdateProposal)
		proposalGroup.POST("/:proposalId/delete", handler.DeleteProposal)
		proposalGroup.POST("/suggest", handler.GetProposalSuggestions)
	}
	return router
}
