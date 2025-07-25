package adaptor

import (
	"context"
	"errors"
	"reflect"

	"github.com/cloudwego/hertz/pkg/app"
	hertz "github.com/cloudwego/hertz/pkg/protocol/consts"

	// 注意：请根据您的项目实际路径修改以下 import
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
)

// PostProcess 处理http响应, resp要求指针或接口类型
// 在日志中记录本次调用详情
// 最佳实践:
// - 在controller中调用业务处理, 处理结束后调用PostProcess
func PostProcess(ctx context.Context, c *app.RequestContext, req, resp any, err error) {
	// 已移除链路追踪(propagation)相关的代码
	log.CtxInfo(ctx, "[%s] req=%s, resp=%s, err=%v", c.Path(), util.JSONF(req), util.JSONF(resp), err)

	// 无错, 正常响应
	if err == nil {
		response := makeResponse(resp)
		c.JSON(hertz.StatusOK, response)
		return // 增加 return 明确结束流程
	}

	var ex *exception.Errorx
	if errors.As(err, &ex) { // 自定义业务异常
		// 对于自定义业务异常，我们返回 200 OK，并将错误信息放在响应体中
		c.JSON(hertz.StatusOK, ex)
	} else { // 其他未捕获的常规错误, 状态码500
		log.CtxError(ctx, "internal server error, err=%s", err.Error())
		// 对于服务器内部错误，返回 500 状态码和通用错误信息
		c.String(hertz.StatusInternalServerError, "Internal Server Error")
	}
}

// makeResponse 通过反射构造嵌套格式的响应体
// 原始逻辑：假设响应DTO包含Code, Msg以及其他作为data的字段
func makeResponse(resp any) map[string]any {
	v := reflect.ValueOf(resp)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		// 如果不是预期的类型，返回一个默认的成功结构，并将原始resp作为data
		return map[string]any{
			"code": 0,
			"msg":  "success",
			"data": resp,
		}
	}
	// 构建返回数据
	v = v.Elem()
	response := map[string]any{
		"code": v.FieldByName("Code").Interface(), // 使用Interface()更通用
		"msg":  v.FieldByName("Msg").String(),
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
