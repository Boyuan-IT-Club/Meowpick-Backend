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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// CreateComment godoc
// @Summary 发布课程评论
// @Description 用户对指定课程发布评论
// @Tags comment
// @Accept json
// @Produce json
// @Param body body dto.CreateCommentReq true "CreateCommentReq"
// @Success 200 {object} dto.CreateCommentResp
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
// @Description 根据课程ID分页查询评论列表
// @Tags comment
// @Produce json
// @Param courseId query int true "课程ID"
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Success 200 {object} dto.ListCourseCommentsResp
// @Router /api/comment/query [get]
func ListCourseComments(c *gin.Context) {
	var err error
	var req dto.ListCourseCommentsReq
	var resp *dto.ListCourseCommentsResp

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().CommentService.GetCourseComments(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetTotalCourseCommentsCount godoc
// @Summary 获取吐槽总数
// @Description 获取系统中所有课程评论的总数量
// @Tags comment
// @Produce json
// @Success 200 {object} dto.GetTotalCourseCommentsCountResp
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
// @Description 分页获取当前用户发布过的评论
// @Tags comment
// @Accept json
// @Produce json
// @Param body body dto.GetMyCommentsReq true "GetMyCommentsReq"
// @Success 200 {object} dto.GetMyCommentsResp
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
