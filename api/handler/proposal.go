package handler

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/gin-gonic/gin"
)

// CreateProposal 新建一个提案
// @router /api/proposal/add [POST]
func CreateProposal(c *gin.Context) {
	var req dto.CreateProposalReq
	var resp *dto.CreateProposalResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		PostProcess(c, &req, nil, err)
		return
	}
	c.Set(consts.CtxUserID, token.GetUserID(c))

	resp, err = provider.Get().ProposalService.CreateProposal(c, &req)
	PostProcess(c, req, resp, err)
}

// ListProposals 分页列出所有提案
// @router /api/proposal/list [GET]
func ListProposals(c *gin.Context) {
	// TODO: not implemented
}

// GetProposal 获取提案详情
// @router /api/proposal/:id [GET]
func GetProposal(c *gin.Context) {
	// TODO: not implemented
}

// ApproveProposal 审批提案
// @router /api/proposal/:id/approve [POST]
func ApproveProposal(c *gin.Context) {
	// TODO: not implemented
}

// UpdateProposal 修改提案
// @router /api/proposal/:id/update [POST]
func UpdateProposal(c *gin.Context) {
	// TODO: not implemented
}

// DeleteProposal 删除提案
// @router /api/proposal/:id/delete [POST]
func DeleteProposal(c *gin.Context) {
	// TODO: not implemented
}

// GetProposalSuggestions 获取提案搜索建议
// @router /api/proposal/suggest [GET]
func GetProposalSuggestions(c *gin.Context) {
	// TODO: not implemented
}
