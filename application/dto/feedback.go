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

package dto

import "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"

type CreateFeedbackReq struct {
	Text     string `json:"text" binding:"required"` // 去掉首尾空格后1-2000 Unicode字符；纯文字，不审核，不支持附件
	Category string `json:"category"`                // bug/feature/other，省略为other
}
type FeedbackMessageReq struct {
	Text string `json:"text" binding:"required"` // 1-2000 Unicode字符，作者续问或管理员回复
}
type FeedbackReadReq struct {
	Sequence int64 `json:"sequence"` // 客户端已展示的最大消息序号；只前进，不超过当前对话最大序号
}
type ListFeedbackReq struct {
	Status   string `form:"status"`   // pending/answered/closed
	Category string `form:"category"` // bug/feature/other
	Keyword  string `form:"keyword"`  // 管理员按整段对话正文进行普通文本模糊搜索
	*PageParam
}
type FeedbackVO struct {
	*model.Feedback
	UnreadCount int64 `json:"unreadCount"` // 当前查看者未读的对方消息数
}
type FeedbackResp struct {
	*Resp
	Feedback *FeedbackVO              `json:"feedback"`
	Messages []*model.FeedbackMessage `json:"messages"` // 本次发送的消息或当前页消息，按sequence倒序
	Total    int64                    `json:"total"`    // 消息总数
}
type ListFeedbackResp struct {
	*Resp
	Feedbacks []*FeedbackVO `json:"feedbacks"`
	Total     int64         `json:"total"`
}
type FeedbackUnreadResp struct {
	*Resp
	UnreadCount int64 `json:"unreadCount"` // 未读对方消息总数
}
