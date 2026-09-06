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

type ToggleLikeReq struct {
	TargetID   string `json:"-" swaggerignore:"true"`                                                        // 从 URL path 获取
	TargetType string `json:"targetType" binding:"required,oneof=proposal comment" enums:"proposal,comment"` // 点赞对象类型：proposal/comment
}

type ToggleLikeResp struct {
	*LikeVO
	*Resp
}

type LikeVO struct {
	Like    bool  `json:"like"`    // 当前登录用户在操作后或查询时是否已点赞
	LikeCnt int64 `json:"likeCnt"` // 目标当前点赞总数
}
