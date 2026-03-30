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

package model

// MappingType 映射类型
type MappingType int32

const (
	MappingTypeDepartment MappingType = 1 // 院系
	MappingTypeCategory   MappingType = 2 // 课程类别
	MappingTypeCampus     MappingType = 3 // 校区
)

// Mapping 映射模型，用于存储院系、类别、校区、状态等映射数据
type Mapping struct {
	ID   int32       `bson:"_id"    json:"id"`   // 映射ID
	Type MappingType `bson:"type"   json:"type"` // 映射类型
	Name string      `bson:"name"   json:"name"` // 显示名称
	Code int32       `bson:"code"   json:"code"` // 对应的值或编码
}
