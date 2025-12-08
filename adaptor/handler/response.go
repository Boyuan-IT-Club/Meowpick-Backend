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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/lib"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/gin-gonic/gin"
)

// PostProcess 处理http响应, resp要求指针或接口类型
// 在日志中记录本次调用详情, 同时向响应头中注入符合b3规范的链路信息, 主要是trace_id
// 最佳实践:
// - 在controller中调用业务处理, 处理结束后调用PostProcess
func PostProcess(c *gin.Context, req, resp any, err error) {
	log.CtxInfo(c, "[%s] req=%s, response=%s, err=%v", c.FullPath(), lib.JSONF(req), lib.JSONF(resp), err)

	// 无错, 正常响应
	if err == nil {
		response := makeResponse(resp)
		c.JSON(http.StatusOK, response)
		return
	}

	if errors.Is(err, errorx.ErrFindSuccessButNoResult) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success but no result",
			"data": nil,
		})
		return
	}

	var ex errorx.Errorx
	if errors.As(err, &ex) { // errorx错误
		StatusCode := http.StatusOK
		c.JSON(StatusCode, &errorx.Errorx{
			Code: ex.Code,
			Msg:  ex.Msg,
		})
	} else { // 常规错误, 状态码500
		log.CtxError(c, "internal error, err=%s", err.Error())
		code := http.StatusInternalServerError
		c.String(code, err.Error())
	}
}

// makeResponse 通过反射构造嵌套格式的响应体
// 注意：此版本会展示零值（包括 false/0/"")，并会展开顶层的 struct 或 *struct 字段到 data 下。
// 同时会跳过 Code/Msg 的重复展开，且不会覆盖已存在的 data key。
func makeResponse(resp any) map[string]any {
	v := reflect.ValueOf(resp)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	// 构建返回数据
	v = v.Elem()

	// 构建基础响应（假设 Code/Msg 存在并可取）
	response := map[string]any{
		"code": v.FieldByName("Code").Int(),
		"msg":  v.FieldByName("Msg").String(),
	}

	data := make(map[string]any)

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		// 跳过顶层的 Code/Msg 字段（已经放到 response 里）
		if field.Name == "Code" || field.Name == "Msg" {
			continue
		}

		fv := v.Field(i)    // reflect.Value
		ftype := field.Type // reflect.Type
		// 先处理 struct 或 *struct（展开内层字段到 data）
		if (fv.Kind() == reflect.Ptr && ftype.Elem().Kind() == reflect.Struct) ||
			(fv.Kind() == reflect.Struct && ftype.Kind() == reflect.Struct) {

			var inner reflect.Value
			var innerType reflect.Type

			if fv.Kind() == reflect.Ptr {
				// 指针指向 struct：若为 nil 则使用零值 struct，以便也能展开零值字段
				if fv.IsNil() {
					innerType = ftype.Elem()
					inner = reflect.Zero(innerType)
				} else {
					inner = fv.Elem()
					innerType = inner.Type()
				}
			} else { // 直接 struct 值
				inner = fv
				innerType = inner.Type()
			}

			for j := 0; j < inner.NumField(); j++ {
				f := innerType.Field(j)

				// 跳过可能来自 Resp 的 Code/Msg 字段
				if f.Name == "Code" || f.Name == "Msg" {
					continue
				}

				tag := f.Tag.Get("json")
				if tag == "" || tag == "-" {
					continue
				}
				key := strings.Split(tag, ",")[0]
				if key == "" {
					continue
				}

				// 即使是零值也要展示，所以直接取 Interface()
				val := inner.Field(j).Interface()

				// 不覆盖已存在 key（先到先得）
				if _, exists := data[key]; !exists {
					data[key] = val
				}
			}
			continue
		}

		// 普通非 struct 字段 —— 即使是零值也展示
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		key := strings.Split(jsonTag, ",")[0]
		if key == "" {
			continue
		}

		data[key] = fv.Interface()
	}

	if len(data) > 0 {
		response["data"] = data
	}

	return response
}
