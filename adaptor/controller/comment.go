package controller

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"net/http"
)

type CommentController struct {
	CommentService service.ICommentService
}

var CommentControllerSet = wire.NewSet(
	wire.Struct(new(CommentController), "*"),
)

func CreateComment(c *gin.Context) {
	var req cmd.CreateCommentCmd

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误或缺少必填项"})
		return
	}

	// TODO: 这里的 userID 将在未来由“认证中间件”提供
	userID := "66a0d533722904b3952243d4"

	p := provider.Get()
	standardCtx := c.Request.Context()
	createdComment, err := p.CommentService.CreateComment(standardCtx, &req, userID)
	if err != nil {
		log.CtxError(standardCtx, "Service CreateComment failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误，请稍后重试"})
		return
	}

	c.JSON(http.StatusCreated, createdComment)
}
