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
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/lib"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/gin-gonic/gin"
)

// Response 统一响应格式，仅用于 Swagger 文档生成
type Response[T any] struct {
	Code int    `json:"code" example:"0"`      // 业务代码, 0表示成功
	Msg  string `json:"msg" example:"success"` // 提示信息
	Data T      `json:"data"`                  // 实际业务数据
}

// PostProcess 处理http响应, resp要求指针或接口类型
// 在日志中记录本次调用详情, 同时向响应头中注入符合b3规范的链路信息, 主要是trace_id
// 最佳实践: 在Handler中调用业务处理, 处理结束后调用PostProcess
func PostProcess(c *gin.Context, req, resp any, err error) {
	logs.CtxInfof(c, "[PostProcess] [%s] req=%s, resp=%s, err=%v", c.FullPath(), lib.JSONF(req), lib.JSONF(resp), err)

	// 无错, 正常响应
	if err == nil {
		response := makeResponse(resp)
		c.JSON(http.StatusOK, response)
		return
	}

	var se errorx.StatusError
	if errors.As(err, &se) {
		c.JSON(http.StatusOK, gin.H{
			"code": se.Code(),
			"msg":  se.Msg(),
			"data": nil,
		})
	} else {
		// 其他非 errorx 错误，500
		logs.CtxErrorf(c, "[PostProcess] internal error, err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  err.Error(),
			"data": nil,
		})
	}
}

// makeResponse 通过反射构造嵌套格式的响应体
// 会展示零值（包括 false/0/"")，并会展开顶层的 struct 或 *struct 字段到 data 下。
// 不会覆盖已存在的 data key。
func makeResponse(resp any) map[string]any {
	v := reflect.ValueOf(resp)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	v = v.Elem()

	response := map[string]any{
		"code": v.FieldByName("Code").Int(),
		"msg":  v.FieldByName("Msg").String(),
	}

	data := make(map[string]any)
	flattenStruct(v, data)

	if len(data) > 0 {
		response["data"] = data
	}

	return response
}

// flattenStruct 递归展开 struct 的字段到 data 中
// 对于嵌入的 struct/*struct（无 json tag），递归展开其字段
// 对于带 json tag 的 struct/*struct，作为一个整体存入 data
func flattenStruct(v reflect.Value, data map[string]any) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)

		if field.Name == "Code" || field.Name == "Msg" {
			continue
		}

		fv := v.Field(i)
		ft := field.Type
		jsonTag := field.Tag.Get("json")
		hasExplicitTag := jsonTag != "" && jsonTag != "-"

		isStruct := (fv.Kind() == reflect.Struct) ||
			(fv.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct)

		if !hasExplicitTag && isStruct {
			var inner reflect.Value
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					inner = reflect.Zero(ft.Elem())
				} else {
					inner = fv.Elem()
				}
			} else {
				inner = fv
			}
			// 无 json tag 的嵌入 struct，递归展开其字段
			flattenStruct(inner, data)
			continue
		}

		// 有显式 json tag，或者非 struct 类型
		if !hasExplicitTag {
			continue
		}

		key := strings.Split(jsonTag, ",")[0]
		if key == "" || key == "-" {
			continue
		}

		if _, exists := data[key]; !exists {
			data[key] = fv.Interface()
		}
	}
}
