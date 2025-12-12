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

// SignIn 用户登录接口
// @router /api/sign_in [POST]
func SignIn(c *gin.Context) {
	var err error
	var req dto.SignInReq
	var resp *dto.SignInResp

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	tokenStr, err := token.ExtractToken(c.Request.Header)
	if err != nil {
		PostProcess(c, &req, nil, err)
		return
	}

	c.Set(consts.CtxToken, tokenStr)
	resp, err = provider.Get().AuthService.SignIn(c, &req)
	PostProcess(c, &req, resp, err)
}
