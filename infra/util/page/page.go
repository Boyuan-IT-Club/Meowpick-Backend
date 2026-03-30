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

package page

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FindPageOption(param dto.IPageParam) *options.FindOptions {
	// page.go ：修复了一个潜在的空指针风险，确保统计数量时不会崩溃
	ops := options.Find()
	if param == nil {
		ops.SetLimit(100) // 默认上限 100
		return ops
	}
	page, size := param.UnWrap()
	ops.SetSkip((page - 1) * size)
	ops.SetLimit(size)

	return ops
}

func DSort(s string, i int) bson.D {
	return bson.D{{s, i}}
}
