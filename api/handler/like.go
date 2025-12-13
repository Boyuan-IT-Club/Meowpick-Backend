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

// ToggleLike 点赞或取消点赞某个目标（课程、评论等）
// @router /api/action/like/{id} [POST]
func ToggleLike(c *gin.Context) {
	var req dto.ToggleLikeReq
	var resp *dto.ToggleLikeResp
	var err error

	req.TargetID = c.Param(consts.CtxLikeID)

	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err = provider.Get().LikeService.ToggleLike(c, &req)
	PostProcess(c, req, resp, err)
}
