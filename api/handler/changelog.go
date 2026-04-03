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
	"strconv"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
)

// ListAdminLogs 查询管理员日志列表
// @Summary 查询管理员日志列表
// @Description 分页查询管理员操作日志
// @Tags Admin
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} dto.ListAdminLogsResp
// @Router /api/admin/logs [get]
func ListAdminLogs(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	req := &dto.ListAdminLogsReq{
		PageParam: &dto.PageParam{
			Page: int64(page),
			Size: int64(size),
		},
	}

	resp, err := provider.GetProvider().ChangeLogService.ListAdminLogs(c.Request.Context(), req)
	Response(c, resp, err)
}
