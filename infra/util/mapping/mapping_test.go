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

import "testing"

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

// TestData_GetCategoryIDsByKeyword 测试分类模糊搜索
func TestData_GetCategoryIDsByKeyword(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		minLen  int // 最少返回几个
	}{
		{"前缀匹配-通识", "通识", 1},  // 至少返回1个
		{"前缀匹配-体育", "体育", 1},  // 至少返回1个
		{"包含匹配-课程", "课程", 10}, // 至少返回10个（因为很多含"课程"）
		{"大小写不敏感", "通识", 1},
		{"空关键词", "", 0},
		{"不存在的关键词", "不存在的分类xxx", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Data.GetCategoryIDsByKeyword(tt.keyword)
			if len(results) < tt.minLen {
				t.Errorf("GetCategoryIDsByKeyword(%q) 返回 %d 个结果, 期望至少 %d",
					tt.keyword, len(results), tt.minLen)
			}
		})
	}
}

// TestData_GetBestCategoryIDByKeyword 测试最佳匹配
func TestData_GetBestCategoryIDByKeyword(t *testing.T) {
	// 前缀匹配应该返回第一个
	id := Data.GetBestCategoryIDByKeyword("通识")
	if id == 0 {
		t.Error("GetBestCategoryIDByKeyword 应该返回匹配的 ID")
	}

	// 不匹配应该返回 0
	id = Data.GetBestCategoryIDByKeyword("不存在的分类")
	if id != 0 {
		t.Errorf("不匹配时应该返回 0, 得到 %d", id)
	}
}

// TestData_GetDepartmentIDsByKeyword 测试院系模糊搜索
func TestData_GetDepartmentIDsByKeyword(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		minLen  int
	}{
		{"前缀匹配-计算机", "计算机", 1},
		{"前缀匹配-教育", "教育", 10}, // 很多含"教育"的院系
		{"包含匹配-学院", "学院", 20}, // 很多含"学院"的院系
		{"空关键词", "", 0},
		{"不存在的关键词", "不存在的院系", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Data.GetDepartmentIDsByKeyword(tt.keyword)
			if len(results) < tt.minLen {
				t.Errorf("GetDepartmentIDsByKeyword(%q) 返回 %d 个结果, 期望至少 %d",
					tt.keyword, len(results), tt.minLen)
			}
		})
	}
}

// TestData_双向映射一致性 测试 ID↔Name 双向映射是否一致
func TestData_双向映射一致性(t *testing.T) {
	// 遍历所有校区，验证双向映射一致
	for id, name := range Data.CampusNameByID {
		reverseID := Data.CampusIDByName[name]
		if reverseID != id {
			t.Errorf("校区双向映射不一致: ID %d → %q → ID %d", id, name, reverseID)
		}
	}

	// 验证院系双向映射（院系有重名 这个暂时没做测试）
	/*for id, name := range Data.DepartmentNameByID {
		reverseID := Data.DepartmentIDByName[name]
		if reverseID != id {
			t.Errorf("院系双向映射不一致: ID %d → %q → ID %d", id, name, reverseID)
		}
	}*/

	// 验证分类双向映射（只测试前10个）
	count := 0
	for id, name := range Data.CategoryNameByID {
		reverseID := Data.CategoryIDByName[name]
		if reverseID != id {
			t.Errorf("分类双向映射不一致: ID %d → %q → ID %d", id, name, reverseID)
		}
		count++
		if count >= 10 {
			break
		}
	}
}

// TestData_init初始化完整性 测试 init 函数是否正确初始化所有映射
func TestData_init初始化完整性(t *testing.T) {
	// 验证校区映射不为空
	if len(Data.CampusNameByID) == 0 {
		t.Error("CampusNameByID 映射为空，init 可能未执行")
	}
	if len(Data.CampusIDByName) == 0 {
		t.Error("CampusIDByName 映射为空，init 可能未执行")
	}

	// 验证院系映射不为空
	if len(Data.DepartmentNameByID) == 0 {
		t.Error("DepartmentNameByID 映射为空，init 可能未执行")
	}

	// 验证分类映射不为空
	if len(Data.CategoryNameByID) == 0 {
		t.Error("CategoryNameByID 映射为空，init 可能未执行")
	}

	// 验证提案状态映射不为空
	if len(Data.ProposalStatusNameByID) == 0 {
		t.Error("ProposalStatusNameByID 映射为空，init 可能未执行")
	}
}
