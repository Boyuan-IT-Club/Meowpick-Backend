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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// Like .
// @router /api/action/like/{id} [POST]
func Like(c *gin.Context) {
	var req dto.CreateLikeReq
	var resp *dto.LikeResp
	var err error

	// 解析目标id和用户id
	req.TargetID = c.Param("id") // 前端采用路由匹配传参，直接解析即可

	c.Set(consts.ContextUserID, token.GetUserId(c))
	// 未来可能需要添加targetType解析
	resp, err = provider.Get().LikeService.Like(c, &req)
	PostProcess(c, nil, resp, err)
}
