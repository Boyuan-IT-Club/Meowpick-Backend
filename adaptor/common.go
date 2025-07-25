package common

import (
	"context"
	"errors"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	hertz "github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/xh-polaris/psych-pkg/errorx"
	"github.com/xh-polaris/psych-pkg/util"
	"github.com/xh-polaris/psych-pkg/util/logx"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
	"reflect"
)

// httpx 是hertz框架的http处理package

// PostProcess 处理http响应, resp要求指针或接口类型
// 在日志中记录本次调用详情, 同时向响应头中注入符合b3规范的链路信息, 主要是trace_id
// 最佳实践:
// - 在controller中调用业务处理, 处理结束后调用PostProcess
func PostProcess(ctx context.Context, c *app.RequestContext, req, resp any, err error) {
	b3.New().Inject(ctx, &headerProvider{headers: &c.Response.Header})
	logx.CtxInfo(ctx, "[%s] req=%s, resp=%s, err=%v", c.Path(), util.JSONF(req), util.JSONF(resp), err)

	// 无错, 正常响应
	if err == nil {
		response := makeResponse(resp)
		c.JSON(hertz.StatusOK, response)
	}

	var ex errorx.Errorx
	if errors.As(err, &ex) { // errorx错误
		StatusCode := hertz.StatusOK
		c.JSON(StatusCode, &errorx.Errorx{
			Code: ex.Code,
			Msg:  ex.Msg,
		})
	} else { // 常规错误, 状态码500
		logx.CtxError(ctx, "internal error, err=%s", err.Error())
		code := hertz.StatusInternalServerError
		c.String(code, err.Error())
	}
}

// makeResponse 通过反射构造嵌套格式的响应体
func makeResponse(resp any) map[string]any {
	v := reflect.ValueOf(resp)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	// 构建返回数据
	v = v.Elem()
	response := map[string]any{
		"code": v.FieldByName("Code").Int(),
		"msg":  v.FieldByName("Msg").String(),
	}
	data := make(map[string]any)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && field.Name != "Code" && field.Name != "Msg" {
			if fieldValue := v.Field(i).Interface(); !reflect.ValueOf(fieldValue).IsZero() {
				data[jsonTag] = fieldValue
			}
		}
	}
	if len(data) > 0 {
		response["data"] = data
	}
	return response
}

var _ propagation.TextMapCarrier = &headerProvider{}

type headerProvider struct {
	headers *protocol.ResponseHeader
}

// Get a value from metadata by key
func (m *headerProvider) Get(key string) string {
	return m.headers.Get(key)
}

// Set a value to metadata by k/v
func (m *headerProvider) Set(key, value string) {
	m.headers.Set(key, value)
}

// Keys Iteratively get all keys of metadata
func (m *headerProvider) Keys() []string {
	out := make([]string, 0)

	m.headers.VisitAll(func(key, value []byte) {
		out = append(out, string(key))
	})

	return out
}
