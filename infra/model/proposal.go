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

import (
	"time"
)

type Proposal struct {
	ID        string    `bson:"_id,omitempty"      json:"id"`
	UserID    string    `bson:"userId"             json:"userId"`               // 提出Proposal的用户ID
	Title     string    `bson:"title"              json:"title"`                // 标题
	Content   string    `bson:"content"            json:"content"`              // 描述的内容
	Deleted   bool      `bson:"deleted"            json:"deleted"`              // 删除标记
	DeletedAt time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"` //删除时间
	Status    int32     `bson:"status"             json:"status"`               // 提案的状态，0: 待审核，1: 通过，2: 拒绝
	Course    *Course   `bson:"course"             json:"course"`               // 课程信息，包含教师的ID（未创建不需要ID）
	CreatedAt time.Time `bson:"createdAt"          json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"          json:"updatedAt"`
}
