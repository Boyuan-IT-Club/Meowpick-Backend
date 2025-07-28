package util

import (
	"bytes"
	"fmt"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/cloudwego/hertz/pkg/common/json"
	"github.com/gin-gonic/gin"
	"io"
)

func ObtainParameter(c *gin.Context, key string) string {
	// 从请求中获取参数，优先从JSON body中获取，其次从form参数中获取
	contentType := c.ContentType()

	// 如果是JSON请求，尝试从body中获取
	if contentType == "application/json" {
		// 读取body内容
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Error("读取请求体失败: %v", err)
			return ""
		}
		// 恢复body，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// 使用hertz的json解析器
		var jsonBody map[string]interface{}
		if err := json.Unmarshal(body, &jsonBody); err != nil {
			log.Error("解析JSON失败: %v, body: %s", err, string(body))
			return ""
		}

		if val, ok := jsonBody[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
			// 如果不是字符串类型，转换为字符串
			return fmt.Sprintf("%v", val)
		}
	}

	// 从form参数中获取
	return c.Request.FormValue(key)
}

// ObtainHeader 从请求头中获取指定参数
func ObtainHeader(c *gin.Context, key string) string {
	return c.GetHeader(key)
}
