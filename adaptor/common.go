package adaptor

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"

	// 注意：请根据您的项目实际路径修改以下 import
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
)

// PostProcess 处理http响应, 专门为 Gin 框架适配。
// 在 Controller 中调用业务处理后，使用此函数来统一格式化输出。
func PostProcess(c *gin.Context, req, resp any, err error) {
	// 从 Gin 的 Context 中获取标准的 context.Context，用于日志记录
	ctx := c.Request.Context()
	log.CtxInfo(ctx, "[%s] req=%s, resp=%s, err=%v", c.FullPath(), util.JSONF(req), util.JSONF(resp), err)

	// 无错, 正常响应
	if err == nil {
		response := makeResponse(resp)
		// 已将 hertz.StatusOK 替换为 http.StatusOK
		c.JSON(http.StatusOK, response)
		return
	}

	var ex *exception.Errorx
	if errors.As(err, &ex) { // 自定义业务异常
		// 对于业务逻辑中的已知错误，返回 200 状态码，错误信息在 JSON 体中体现
		c.JSON(http.StatusOK, ex)
	} else { // 其他未捕获的常规错误, 状态码 500
		log.CtxError(ctx, "internal server error, err=%s", err.Error())
		// 为安全起见，不向客户端暴露详细的服务器错误信息
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

// makeResponse 通过反射构造嵌套格式的响应体
func makeResponse(resp any) map[string]any {
	v := reflect.ValueOf(resp)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		// 如果 resp 不是预期的类型，返回 nil，让上层处理
		return nil
	}
	// 构建返回数据
	v = v.Elem()
	response := make(map[string]any)

	// 确保字段存在且类型正确
	codeField := v.FieldByName("Code")
	if codeField.IsValid() && codeField.CanInt() {
		response["code"] = codeField.Int()
	}

	msgField := v.FieldByName("Msg")
	if msgField.IsValid() && msgField.Kind() == reflect.String {
		response["msg"] = msgField.String()
	}

	data := make(map[string]any)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && field.Name != "Code" && field.Name != "Msg" {
			fieldValue := v.Field(i)
			if !fieldValue.IsZero() {
				data[jsonTag] = fieldValue.Interface()
			}
		}
	}
	if len(data) > 0 {
		response["data"] = data
	}
	return response
}
