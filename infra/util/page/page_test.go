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
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
)

// TestFindPageOption 测试分页选项计算
func TestFindPageOption(t *testing.T) {
	tests := []struct {
		name      string
		pageParam *dto.PageParam
		wantSkip  int64
		wantLimit int64
	}{
		{
			name:      "正常情况: page=1, size=10",
			pageParam: &dto.PageParam{Page: 1, PageSize: 10},
			wantSkip:  10, // page * size = 1 * 10
			wantLimit: 10,
		},
		{
			name:      "第一页: page=0",
			pageParam: &dto.PageParam{Page: 0, PageSize: 10},
			wantSkip:  0,
			wantLimit: 10,
		},
		{
			name:      "PageSize 为 0（应该用默认值 10）",
			pageParam: &dto.PageParam{Page: 1, PageSize: 0},
			wantSkip:  10, // 1 * 10（默认）
			wantLimit: 10,
		},
		{
			name:      "PageSize 超限 200（应该用默认值 10）",
			pageParam: &dto.PageParam{Page: 1, PageSize: 200},
			wantSkip:  10, // 1 * 10（默认）
			wantLimit: 10,
		},
		{
			name:      "页码为负数（应该修正为 0）",
			pageParam: &dto.PageParam{Page: -5, PageSize: 10},
			wantSkip:  0, // 0 * 10
			wantLimit: 10,
		},
		{
			name:      "PageSize 为负数（应该修正为 10）",
			pageParam: &dto.PageParam{Page: 1, PageSize: -10},
			wantSkip:  10, // 1 * 10
			wantLimit: 10,
		},
		{
			name:      "PageSize 为最大值 100",
			pageParam: &dto.PageParam{Page: 2, PageSize: 100},
			wantSkip:  200, // 2 * 100
			wantLimit: 100,
		},
		{
			name:      "第二页: page=2, size=20",
			pageParam: &dto.PageParam{Page: 2, PageSize: 20},
			wantSkip:  40, // 2 * 20
			wantLimit: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := FindPageOption(tt.pageParam)

			// 检查 Skip
			if opts.Skip == nil {
				t.Error("Skip 不应该为 nil")
			} else if *opts.Skip != tt.wantSkip {
				t.Errorf("Skip = %v, want %v", *opts.Skip, tt.wantSkip)
			}

			// 检查 Limit
			if opts.Limit == nil {
				t.Error("Limit 不应该为 nil")
			} else if *opts.Limit != tt.wantLimit {
				t.Errorf("Limit = %v, want %v", *opts.Limit, tt.wantLimit)
			}
		})
	}
}

// TestDSort 测试排序构造
func TestDSort(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		direction int
		wantKey   string
		wantValue int
	}{
		{"升序: createdAt", "createdAt", 1, "createdAt", 1},
		{"降序: createdAt", "createdAt", -1, "createdAt", -1},
		{"升序: title", "title", 1, "title", 1},
		{"降序: updatedAt", "updatedAt", -1, "updatedAt", -1},
		{"升序: _id", "_id", 1, "_id", 1},
		{"降序: userId", "userId", -1, "userId", -1},
		{"升序: status", "status", 1, "status", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DSort(tt.field, tt.direction)

			if len(result) != 1 {
				t.Errorf("期望返回 1 个元素, 得到 %d", len(result))
			}

			if result[0].Key != tt.wantKey {
				t.Errorf("Key = %v, want %v", result[0].Key, tt.wantKey)
			}
			if result[0].Value != tt.wantValue {
				t.Errorf("Value = %v, want %v", result[0].Value, tt.wantValue)
			}
		})
	}
}
