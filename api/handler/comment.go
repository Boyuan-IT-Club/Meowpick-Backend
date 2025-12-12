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

// CreateComment 发布评论
// @router /api/comment/add [POST]
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

// ListCourseComments 分页获取课程评论
// @router /api/comment/query [GET]
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

// GetTotalCourseCommentsCount 获得小程序收录吐槽总数
// @router /api/search/total [GET]
func GetTotalCourseCommentsCount(c *gin.Context) {
	var resp *dto.GetTotalCourseCommentsCountResp
	var err error

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err = provider.Get().CommentService.GetTotalCommentsCount(c)
	PostProcess(c, nil, resp, err)
}

// GetMyComments .
// @router /api/comment/history [POST]
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
