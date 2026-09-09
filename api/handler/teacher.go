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

// CreateTeacher godoc
// @Summary 新建教师
// @Description 管理员独立创建正式教师。此接口的姓名、职称和所属院系均必填；与提案审批自动创建教师不同，独立创建教师不接受空院系。院系名称不存在时会创建并持久化对应映射
// @Tags teacher
// @Accept json
// @Produce json
// @Param body body dto.CreateTeacherReq true "正式教师信息；name、title、department 均必填"
// @Success 200 {object} Response[dto.CreateTeacherResp]
// @Router /api/teacher/add [post]
func CreateTeacher(c *gin.Context) {
	var req *dto.CreateTeacherReq
	var resp *dto.CreateTeacherResp
	var err error

	if err = c.ShouldBind(&req); err != nil {
		PostProcess(c, req, resp, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().TeacherService.CreateTeacher(c, req)
	PostProcess(c, req, resp, err)
}

// GetTeacherSuggestions godoc
// @Summary 获取教师搜索建议（已弃用）
// @Description 已弃用，当前前端未调用且不再扩展搜索规则；提案教师建议请使用 /api/proposal/field-suggestions?field=teacherName，全局搜索建议请使用 /api/search/suggest
// @Deprecated
// @Tags teacher
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码，小于1按1处理" default(1)
// @Param pageSize query int false "每页数量，范围1-100，超出范围按10处理" default(10)
// @Success 200 {object} Response[dto.GetTeacherSuggestionsResp]
// @Router /api/teacher/suggest [get]
func GetTeacherSuggestions(c *gin.Context) {
	var req *dto.GetTeacherSuggestionsReq
	var resp *dto.GetTeacherSuggestionsResp
	var err error

	if err = c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, req, resp, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().TeacherService.GetTeacherSuggestions(c, req)
	PostProcess(c, req, resp, err)
}
