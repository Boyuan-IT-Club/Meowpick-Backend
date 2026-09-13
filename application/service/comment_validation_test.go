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

package service

import (
	"strings"
	"testing"
)

func TestNormalizeCommentInput(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		tags        []string
		wantContent string
		wantTags    []string
		wantErr     bool
	}{
		{name: "valid", content: "  很有收获  ", tags: []string{"推荐", "硬核"}, wantContent: "很有收获", wantTags: []string{"推荐", "硬核"}},
		{name: "empty tags", content: "不错", tags: nil, wantContent: "不错", wantTags: []string{}},
		{name: "blank content", content: " \t\n", wantErr: true},
		{name: "content too long", content: strings.Repeat("课", 141), wantErr: true},
		{name: "four tags", content: "不错", tags: []string{"容易", "推荐", "幽默", "严格"}, wantContent: "不错", wantTags: []string{"容易", "推荐", "幽默", "严格"}},
		{name: "too many tags", content: "不错", tags: []string{"容易", "推荐", "幽默", "严格", "硬核"}, wantErr: true},
		{name: "unknown tag", content: "不错", tags: []string{"自定义"}, wantErr: true},
		{name: "duplicate tag", content: "不错", tags: []string{"推荐", "推荐"}, wantErr: true},
		{name: "tag is not trimmed into allowlist", content: "不错", tags: []string{" 推荐 "}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, tags, err := normalizeCommentInput(tt.content, tt.tags)
			if tt.wantErr {
				if err == nil {
					t.Fatal("normalizeCommentInput() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeCommentInput() error = %v", err)
			}
			if content != tt.wantContent {
				t.Fatalf("content = %q, want %q", content, tt.wantContent)
			}
			if len(tags) != len(tt.wantTags) {
				t.Fatalf("tags = %#v, want %#v", tags, tt.wantTags)
			}
			for i := range tags {
				if tags[i] != tt.wantTags[i] {
					t.Fatalf("tags = %#v, want %#v", tags, tt.wantTags)
				}
			}
		})
	}
}
