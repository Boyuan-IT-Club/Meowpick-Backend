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

package dto

import "testing"

// TestResp_Success 测试 Success 函数
func TestResp_Success(t *testing.T) {
	resp := Success()

	if resp == nil {
		t.Fatal("Success() 返回 nil")
	}
	if resp.Code != 0 {
		t.Errorf("期望 Code=0, 得到 %d", resp.Code)
	}
	if resp.Msg != "success" {
		t.Errorf("期望 Msg='success', 得到 %s", resp.Msg)
	}
}

// TestPageParam_UnWrap 测试 PageParam 的 UnWrap 方法
func TestPageParam_UnWrap(t *testing.T) {
	tests := []struct {
		name     string
		input    *PageParam
		wantPage int64
		wantSize int64
	}{
		{
			name:     "正常情况",
			input:    &PageParam{Page: 1, PageSize: 10},
			wantPage: 1,
			wantSize: 10,
		},
		{
			name:     "页码为负数",
			input:    &PageParam{Page: -5, PageSize: 10},
			wantPage: 0,
			wantSize: 10,
		},
		{
			name:     "PageSize 超过最大值",
			input:    &PageParam{Page: 1, PageSize: 200},
			wantPage: 1,
			wantSize: 10,
		},
		{
			name:     "PageSize 为 0",
			input:    &PageParam{Page: 1, PageSize: 0},
			wantPage: 1,
			wantSize: 10,
		},
		{
			name:     "PageSize 为负数",
			input:    &PageParam{Page: 1, PageSize: -10},
			wantPage: 1,
			wantSize: 10,
		},
		{
			name:     "页码为 0",
			input:    &PageParam{Page: 0, PageSize: 10},
			wantPage: 0,
			wantSize: 10,
		},
		{
			name:     "PageSize 为最大值 100",
			input:    &PageParam{Page: 1, PageSize: 100},
			wantPage: 1,
			wantSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, size := tt.input.UnWrap()
			if page != tt.wantPage {
				t.Errorf("page = %v, want %v", page, tt.wantPage)
			}
			if size != tt.wantSize {
				t.Errorf("size = %v, want %v", size, tt.wantSize)
			}
		})
	}
}
