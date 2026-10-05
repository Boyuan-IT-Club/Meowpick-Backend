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

// PreviewProposalApproval godoc
// @Summary 预览共同审批、重复候选与字段冲突
// @Description 管理员先调用此接口，使用与approve完全相同的finalCourse/final/proposalIds/confirmedNewTeachers。新增课程只有名称、代码、完整教师身份集合、校区集合、分类、开课院系六项均一致才自动归组；已有教师按ID、新教师按去首尾空格后的姓名比较，职称/content/匿名不比较，集合顺序无关。同名待审提案列入suspicious，可借助GET /api/proposal/suggest?status=pending&keyword=课程名&type=create_course模糊搜索并手动选入proposalIds。手动选择会纳入所选提案的自动组。existingCourses是正式课程，不能作为共同通过成员；管理员发现课程已有时手动拒绝，可批量同理由拒绝。teacherCandidates提示同名正式教师，复用时把finalCourse教师改为已有ID；仍新建时把姓名加入confirmedNewTeachers并再次预览。final仅用于修改提案，finalCourse仅用于新增提案；传错类型字段返回108000015，不静默忽略。修改提案conflicts.state为unchanged/realized/conflict；冲突字段须明确传final并再次预览。canApprove=false表示无实际变更、正式课程重复或未确认的冲突。确认审批必须原样携带previewToken和本次预览参数；变化返回108000031，重新预览。业务错误均HTTP200、非零code、data=null，候选详情从本成功接口获取。
// @Tags proposal
// @Accept json
// @Produce json
// @Param proposalId path string true "主提案ID"
// @Param body body dto.ToggleProposalReq true "审批预览参数，不需要previewToken"
// @Success 200 {object} Response[dto.ApprovalPreviewResp]
// @Router /api/proposal/{proposalId}/preview [post]
func PreviewProposalApproval(c *gin.Context) {
	var req dto.ToggleProposalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	req.ProposalID = c.Param("proposalId")
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().ProposalService.PreviewApproval(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetTeacher godoc
// @Summary 获取教师资料与历史贡献者
// @Description 登录用户获取正式教师当前资料及contributors数组。按用户去重，昵称取当前值；匿名作者不返回ID或昵称。教师资料修改通过后所有引用该教师的课程展示随之更新。
// @Tags teacher
// @Produce json
// @Param teacherId path string true "教师ID"
// @Success 200 {object} Response[dto.GetTeacherResp]
// @Router /api/teacher/{teacherId} [get]
func GetTeacher(c *gin.Context) {
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().ProposalService.GetTeacher(c, c.Param("teacherId"))
	PostProcess(c, nil, resp, err)
}

// GetCourseHistory godoc
// @Summary 分页查询课程资料历史
// @Description 登录可查看未撤回的一次实际资料变更及原值、建议、最终值、公开贡献者。同批多个提案只产生一条历史，匿名作者不返回身份。旧数据没有来源时不补造创建记录；有旧来源但无最终快照时legacy=true且省略final，不能用当前资料冒充历史。
// @Tags course
// @Produce json
// @Param courseId path string true "正式资料ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页1-100" default(10)
// @Success 200 {object} Response[dto.EntityHistoryResp]
// @Router /api/course/{courseId}/history [get]
func GetCourseHistory(c *gin.Context) {
	var req dto.EntityHistoryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	req.TargetID = c.Param("courseId")
	req.TargetType = "course"
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().ProposalService.EntityHistory(c, &req)
	PostProcess(c, &req, resp, err)
}

// GetTeacherHistory godoc
// @Summary 分页查询教师资料历史
// @Description 登录可查看未撤回的一次实际资料变更及原值、建议、最终值、公开贡献者。同批多个提案只产生一条历史，匿名作者不返回身份。旧数据没有来源时不补造创建记录；有旧来源但无最终快照时legacy=true且省略final，不能用当前资料冒充历史。
// @Tags teacher
// @Produce json
// @Param teacherId path string true "正式资料ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页1-100" default(10)
// @Success 200 {object} Response[dto.EntityHistoryResp]
// @Router /api/teacher/{teacherId}/history [get]
func GetTeacherHistory(c *gin.Context) {
	var req dto.EntityHistoryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	req.TargetID = c.Param("teacherId")
	req.TargetType = "teacher"
	c.Set(consts.CtxUserID, token.GetUserID(c))
	resp, err := provider.Get().ProposalService.EntityHistory(c, &req)
	PostProcess(c, &req, resp, err)
}
