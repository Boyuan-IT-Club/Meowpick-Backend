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

package handler

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// CreateComment godoc
// @Summary 发布课程评论
// @Description 登录用户对指定正式课程发布评论。courseId 和 content 必填，tags 可省略或为空数组；成功响应包含新评论ID、发布时间以及当前点赞状态。该接口本身不修改课程标签字段，课程详情的 tagCount 会从未删除评论标签实时聚合
// @Tags comment
// @Accept json
// @Produce json
// @Param body body dto.CreateCommentReq true "课程ID、评论正文和可选标签"
// @Success 200 {object} Response[dto.CreateCommentResp]
// @Router /api/comment/add [post]
func CreateComment(c *gin.Context) {
	var err error
	var req dto.CreateCommentReq
	var resp *dto.CreateCommentResp

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CommentService.CreateComment(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListCourseComments godoc
// @Summary 分页获取课程评论
// @Description 登录后分页查询指定课程的未删除评论，按创建时间倒序返回，并为每条评论附带当前用户是否点赞及最新点赞总数。新调用使用 courseId；服务端仍兼容旧 query 参数 id，但旧参数不展示在 Swagger 中
// @Tags comment
// @Produce json
// @Param courseId query string true "课程ID"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.ListCourseCommentsResp]
// @Router /api/comment/query [get]
func ListCourseComments(c *gin.Context) {
	var err error
	var req dto.ListCourseCommentsReq
	var resp *dto.ListCourseCommentsResp

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	if req.ID == "" {
		req.ID = req.LegacyID
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CommentService.GetCourseComments(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetTotalCourseCommentsCount godoc
// @Summary 获取吐槽总数
// @Description 登录后获取系统中所有未删除课程评论总数。服务端优先读取短期缓存，缓存不可用或未命中时查询数据库并回填；已随课程撤回而软删除的评论不计入
// @Tags comment
// @Produce json
// @Success 200 {object} Response[dto.GetTotalCourseCommentsCountResp]
// @Router /api/search/total [get]
func GetTotalCourseCommentsCount(c *gin.Context) {
	var resp *dto.GetTotalCourseCommentsCountResp
	var err error

	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CommentService.GetTotalCommentsCount(c)
	PostProcess(c, nil, resp, err)
}

// GetMyComments godoc
// @Summary 获取我的评论历史
// @Description 登录后分页获取当前用户发布的未删除评论，按创建时间倒序返回。每条记录包含当前点赞状态和对应未删除课程的名称、分类、开课院系及教师姓名职称信息
// @Tags comment
// @Accept json
// @Produce json
// @Param body body dto.GetMyCommentsReq true "分页参数；page 小于1按1处理，pageSize 范围1-100且非法值按10处理"
// @Success 200 {object} Response[dto.GetMyCommentsResp]
// @Router /api/comment/history [post]
func GetMyComments(c *gin.Context) {
	var err error
	var req dto.GetMyCommentsReq
	var resp *dto.GetMyCommentsResp

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CommentService.GetMyComments(c, &req)
	PostProcess(c, &req, resp, err)
}
