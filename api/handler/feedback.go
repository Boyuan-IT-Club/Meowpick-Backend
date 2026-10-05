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

// CreateFeedback godoc
// @Summary 提交私密反馈
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Accept json
// @Param body body dto.CreateFeedbackReq true "请求参数"
// @Success 200 {object} Response[dto.FeedbackResp]
// @Router /api/feedback [post]
func CreateFeedback(c *gin.Context) {
	var req dto.CreateFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Create(c, &req)
	PostProcess(c, &req, resp, err)
}

// ListMyFeedback godoc
// @Summary 我的反馈列表
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页1-100" default(10)
// @Param status query string false "pending/answered/closed"
// @Param category query string false "bug/feature/other"
// @Param keyword query string false "管理员正文普通文本模糊搜索"
// @Success 200 {object} Response[dto.ListFeedbackResp]
// @Router /api/feedback/mine [get]
func ListMyFeedback(c *gin.Context) {
	var req dto.ListFeedbackReq
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.List(c, &req, false)
	PostProcess(c, &req, resp, err)
}

// ListAdminFeedback godoc
// @Summary 管理员反馈列表
// @Description 仅管理员可操作，查看全体用户的私密反馈。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页1-100" default(10)
// @Param status query string false "pending/answered/closed"
// @Param category query string false "bug/feature/other"
// @Param keyword query string false "管理员正文普通文本模糊搜索"
// @Success 200 {object} Response[dto.ListFeedbackResp]
// @Router /api/admin/feedback [get]
func ListAdminFeedback(c *gin.Context) {
	var req dto.ListFeedbackReq
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.List(c, &req, true)
	PostProcess(c, &req, resp, err)
}

// GetFeedback godoc
// @Summary 我的反馈对话
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页1-100" default(10)
// @Success 200 {object} Response[dto.FeedbackResp]
// @Router /api/feedback/{feedbackId} [get]
func GetFeedback(c *gin.Context) {
	var req dto.PageParam
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Detail(c, c.Param("feedbackId"), &req, false)
	PostProcess(c, &req, resp, err)
}

// GetAdminFeedback godoc
// @Summary 管理员反馈对话
// @Description 仅管理员可操作，查看全体用户的私密反馈。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页1-100" default(10)
// @Success 200 {object} Response[dto.FeedbackResp]
// @Router /api/admin/feedback/{feedbackId} [get]
func GetAdminFeedback(c *gin.Context) {
	var req dto.PageParam
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Detail(c, c.Param("feedbackId"), &req, true)
	PostProcess(c, &req, resp, err)
}

// SendFeedbackMessage godoc
// @Summary 作者继续回复
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Accept json
// @Param body body dto.FeedbackMessageReq true "请求参数"
// @Success 200 {object} Response[dto.FeedbackResp]
// @Router /api/feedback/{feedbackId}/messages [post]
func SendFeedbackMessage(c *gin.Context) {
	var req dto.FeedbackMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Send(c, c.Param("feedbackId"), &req, false)
	PostProcess(c, &req, resp, err)
}

// ReplyFeedback godoc
// @Summary 管理员回复反馈
// @Description 仅管理员可操作，查看全体用户的私密反馈。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Accept json
// @Param body body dto.FeedbackMessageReq true "请求参数"
// @Success 200 {object} Response[dto.FeedbackResp]
// @Router /api/admin/feedback/{feedbackId}/reply [post]
func ReplyFeedback(c *gin.Context) {
	var req dto.FeedbackMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Send(c, c.Param("feedbackId"), &req, true)
	PostProcess(c, &req, resp, err)
}

// ReadFeedback godoc
// @Summary 标记作者已读位置
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Accept json
// @Param body body dto.FeedbackReadReq true "请求参数"
// @Success 200 {object} Response[dto.Resp]
// @Router /api/feedback/{feedbackId}/read [post]
func ReadFeedback(c *gin.Context) {
	var req dto.FeedbackReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Read(c, c.Param("feedbackId"), &req, false)
	PostProcess(c, &req, resp, err)
}

// ReadAdminFeedback godoc
// @Summary 标记管理员已读位置
// @Description 仅管理员可操作，查看全体用户的私密反馈。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Accept json
// @Param body body dto.FeedbackReadReq true "请求参数"
// @Success 200 {object} Response[dto.Resp]
// @Router /api/admin/feedback/{feedbackId}/read [post]
func ReadAdminFeedback(c *gin.Context) {
	var req dto.FeedbackReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Read(c, c.Param("feedbackId"), &req, true)
	PostProcess(c, &req, resp, err)
}

// CloseFeedback godoc
// @Summary 作者关闭反馈
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Success 200 {object} Response[dto.Resp]
// @Router /api/feedback/{feedbackId}/close [post]
func CloseFeedback(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Close(c, c.Param("feedbackId"), false)
	PostProcess(c, nil, resp, err)
}

// CloseAdminFeedback godoc
// @Summary 管理员关闭反馈
// @Description 仅管理员可操作，查看全体用户的私密反馈。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Param feedbackId path string true "反馈ID"
// @Success 200 {object} Response[dto.Resp]
// @Router /api/admin/feedback/{feedbackId}/close [post]
func CloseAdminFeedback(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Close(c, c.Param("feedbackId"), true)
	PostProcess(c, nil, resp, err)
}

// FeedbackUnread godoc
// @Summary 我的未读反馈回复数
// @Description 仅登录作者可查看和操作自己的反馈，他人反馈与不存在返回同一错误112000001。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Success 200 {object} Response[dto.FeedbackUnreadResp]
// @Router /api/feedback/unread [get]
func FeedbackUnread(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Unread(c, false)
	PostProcess(c, nil, resp, err)
}

// AdminFeedbackUnread godoc
// @Summary 管理员未读反馈消息数
// @Description 仅管理员可操作，查看全体用户的私密反馈。纯文字，每条1-2000 Unicode字符，无外部内容审核；每位用户每分钟最多5条消息、每天UTC+8最多新建10条，超限错误112000003/112000004。状态pending为等待管理员，管理员回复变answered，作者回复回pending，双方可关闭，作者在closed中继续回复重新打开。GET详情不自动标记已读，展示消息后POST read携带已展示的最大sequence，已读位置只前进；管理员各自独立未读，共用待处理列表。无删除消息、附件或微信推送功能。
// @Tags feedback
// @Produce json
// @Success 200 {object} Response[dto.FeedbackUnreadResp]
// @Router /api/admin/feedback/unread [get]
func AdminFeedbackUnread(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().FeedbackService.Unread(c, true)
	PostProcess(c, nil, resp, err)
}
