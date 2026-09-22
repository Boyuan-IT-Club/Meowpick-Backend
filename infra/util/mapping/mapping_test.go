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

package mapping

import (
	"slices"
	"testing"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
)

func TestValidateMappingsAllowsOneCanonicalHistoricalAlias(t *testing.T) {
	err := validateMappings([]*model.Mapping{
		{Type: model.MappingTypeDepartment, Name: "同名院系", Code: 1, Canonical: true},
		{Type: model.MappingTypeDepartment, Name: "同名院系", Code: 2, Canonical: false},
	})
	if err != nil {
		t.Fatalf("expected valid alias mapping, got %v", err)
	}
}

func TestValidateMappingsRejectsMissingOrDuplicateCanonical(t *testing.T) {
	tests := []struct {
		name     string
		mappings []*model.Mapping
	}{
		{
			name: "missing",
			mappings: []*model.Mapping{
				{Type: model.MappingTypeCategory, Name: "同名分类", Code: 1},
			},
		},
		{
			name: "duplicate",
			mappings: []*model.Mapping{
				{Type: model.MappingTypeCategory, Name: "同名分类", Code: 1, Canonical: true},
				{Type: model.MappingTypeCategory, Name: "同名分类", Code: 2, Canonical: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateMappings(tt.mappings); err == nil {
				t.Fatal("expected canonical validation error")
			}
		})
	}
}

// TestData_GetCampusNameByID 测试校区名称查找
func TestData_GetCampusNameByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int32
		expected string
	}{
		{"存在的ID-滴水湖软件楼", 1, "滴水湖软件楼"},
		{"存在的ID-临港校区", 2, "临港校区"},
		{"存在的ID-普陀校区", 3, "普陀校区"},
		{"存在的ID-闵行校区", 4, "闵行校区"},
		{"不存在的ID", 999, "未知校区"},
		{"负数ID", -1, "未知校区"},
		{"零值ID", 0, "未知校区"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Data.GetCampusNameByID(tt.id)
			if result != tt.expected {
				t.Errorf("GetCampusNameByID(%d) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

// TestData_GetCampusIDByName 测试校区ID查找
func TestData_GetCampusIDByName(t *testing.T) {
	tests := []struct {
		name     string
		campus   string
		expected int32
	}{
		{"存在的名称-滴水湖软件楼", "滴水湖软件楼", 1},
		{"存在的名称-临港校区", "临港校区", 2},
		{"存在的名称-普陀校区", "普陀校区", 3},
		{"存在的名称-闵行校区", "闵行校区", 4},
		{"不存在的名称", "不存在的校区", 0},
		{"空字符串", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Data.GetCampusIDByName(tt.campus)
			if result != tt.expected {
				t.Errorf("GetCampusIDByName(%q) = %d, want %d", tt.campus, result, tt.expected)
			}
		})
	}
}

// TestData_GetProposalStatusNameByID 测试提案状态查找
func TestData_GetProposalStatusNameByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int32
		expected string
	}{
		{"待审核状态", 1, "pending"},
		{"已通过状态", 2, "approved"},
		{"已拒绝状态", 3, "rejected"},
		{"不存在的状态", 999, "未知提案状态"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Data.GetProposalStatusNameByID(tt.id)
			if result != tt.expected {
				t.Errorf("GetProposalStatusNameByID(%d) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

// TestData_GetLikeTargetTypeNameByID 测试点赞目标类型查找
func TestData_GetLikeTargetTypeNameByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int32
		expected string
	}{
		{"提案类型", 1, "proposal"},
		{"评论类型", 2, "comment"},
		{"不存在的类型", 999, "未知点赞目标类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Data.GetLikeTargetTypeNameByID(tt.id)
			if result != tt.expected {
				t.Errorf("GetLikeTargetTypeNameByID(%d) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

// TestData_GetChangeLogTargetTypeNameByID 测试变更记录类型查找
func TestData_GetChangeLogTargetTypeNameByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int32
		expected string
	}{
		{"课程类型", 1, "course"},
		{"提案类型", 2, "proposal"},
		{"教师类型", 3, "teacher"},
		{"用户类型", 4, "user"},
		{"不存在的类型", 999, "未知变更记录类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Data.GetChangeLogTargetTypeNameByID(tt.id)
			if result != tt.expected {
				t.Errorf("GetChangeLogTargetTypeNameByID(%d) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

// Reference mapping search tests use a small snapshot, independent of production data.
func TestReferenceMappingSearch(t *testing.T) {
	d := newStaticData()
	d.CategoryNameByID = map[int32]string{1: "通识课程", 2: "专业课程", 3: "English"}
	d.CategoryIDByName = reverseMap(d.CategoryNameByID)
	d.DepartmentNameByID = map[int32]string{1: "计算机学院", 2: "教育学院", 3: "教育研究所"}
	d.DepartmentIDByName = reverseMap(d.DepartmentNameByID)
	for _, tt := range []struct {
		name    string
		search  func(string) []int32
		keyword string
		want    []int32
	}{
		{"category prefix", d.GetCategoryIDsByKeyword, "通识", []int32{1}},
		{"category substring", d.GetCategoryIDsByKeyword, "课程", []int32{1, 2}},
		{"case insensitive", d.GetCategoryIDsByKeyword, "english", []int32{3}},
		{"category missing", d.GetCategoryIDsByKeyword, "不存在", []int32{}},
		{"department prefix", d.GetDepartmentIDsByKeyword, "教育", []int32{2, 3}},
		{"department substring", d.GetDepartmentIDsByKeyword, "学院", []int32{1, 2}},
		{"department missing", d.GetDepartmentIDsByKeyword, "不存在", []int32{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.search(tt.keyword)
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("search(%q) = %v, want %v", tt.keyword, got, tt.want)
			}
		})
	}
	if got := d.GetBestCategoryIDByKeyword("通识"); got != 1 {
		t.Fatalf("best category = %d, want 1", got)
	}
	if got := d.GetBestCategoryIDByKeyword("不存在"); got != 0 {
		t.Fatalf("missing category = %d, want 0", got)
	}
	for id, name := range d.CategoryNameByID {
		if d.GetCategoryNameByID(id) != name || d.GetCategoryIDByName(name) != id {
			t.Fatalf("category round trip failed for %d", id)
		}
	}
	for id, name := range d.DepartmentNameByID {
		if d.GetDepartmentNameByID(id) != name || d.GetDepartmentIDByName(name) != id {
			t.Fatalf("department round trip failed for %d", id)
		}
	}
}

func TestReferenceMappingsStartEmpty(t *testing.T) {
	d := newStaticData()
	if len(d.DepartmentNameByID) != 0 || len(d.DepartmentIDByName) != 0 || len(d.CategoryNameByID) != 0 || len(d.CategoryIDByName) != 0 {
		t.Fatal("department and category mappings must be loaded from MongoDB")
	}
	if d.GetDepartmentIDByName("软件学院院部") != 0 || d.GetCategoryIDByName("通识课程") != 0 {
		t.Fatal("unexpected legacy mapping fallback")
	}
	if d.GetDepartmentNameByID(1) != "未知开课院系" || d.GetCategoryNameByID(1) != "未知分类" {
		t.Fatal("missing mappings must return unknown labels")
	}
	if len(d.CampusNameByID) == 0 || len(d.ProposalStatusNameByID) == 0 {
		t.Fatal("campus and fixed enum defaults must remain available")
	}
}
