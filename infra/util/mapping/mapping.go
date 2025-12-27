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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func (d *StaticData) GetCampusNameByID(id int32) string {
	key := fmt.Sprintf("%d", id)
	if name, ok := d.Campuses[key]; ok {
		return name
	}
	return "未知校区"
}

func (d *StaticData) GetDepartmentNameByID(id int32) string {
	key := fmt.Sprintf("%d", id)
	if name, ok := d.Departments[key]; ok {
		return name
	}
	return "未知院系"
}

func (d *StaticData) GetCategoryNameByID(id int32) string {
	key := fmt.Sprintf("%d", id)
	if name, ok := d.Categories[key]; ok {
		return name
	}
	return "未知课程"
}

func (d *StaticData) GetCampusIDByName(name string) int32 {
	for idStr, campusName := range d.Campuses {
		if campusName == name {
			if id, err := strconv.ParseInt(idStr, 10, 32); err == nil {
				return int32(id)
			}
		}
	}
	return 0
}

func (d *StaticData) GetDepartmentIDByName(name string) int32 {
	for idStr, departmentName := range d.Departments {
		if departmentName == name {
			if id, err := strconv.ParseInt(idStr, 10, 32); err == nil {
				return int32(id)
			}
		}
	}
	return 0
}

func (d *StaticData) GetCategoryIDByName(name string) int32 {
	for idStr, categoryName := range d.Categories {
		if categoryName == name {
			if id, err := strconv.ParseInt(idStr, 10, 32); err == nil {
				return int32(id)
			}
		}
	}
	return 0
}

// --- 搜索方法（正则+包含匹配）---

// GetBestCategoryIDByKeyword 根据关键词获取最匹配的单个分类ID
func (d *StaticData) GetBestCategoryIDByKeyword(keyword string) int32 {
	ids := d.GetCategoryIDsByKeyword(keyword)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// GetBestDepartmentIDByKeyword 根据关键词获取最匹配的单个部门ID
func (d *StaticData) GetBestDepartmentIDByKeyword(keyword string) int32 {
	ids := d.GetDepartmentIDsByKeyword(keyword)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// GetCategoryIDsByKeyword 根据类别关键词快速查找匹配的分类ID
func (d *StaticData) GetCategoryIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.Categories)
}

// GetDepartmentIDsByKeyword 根据部门关键词快速查找匹配的院系ID
func (d *StaticData) GetDepartmentIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.Departments)
}

// 正则+包含模糊搜索，按匹配度排序
func fuzzySearch(keyword string, dataMap map[string]string) []int32 {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(keyword))
	if err != nil {
		return nil
	}
	type result struct {
		id    int32
		score int // 匹配度分数
	}
	var results []result
	for idStr, name := range dataMap {
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			continue
		}
		nameLower := strings.ToLower(name)
		keywordLower := strings.ToLower(keyword)
		if strings.HasPrefix(nameLower, keywordLower) {
			results = append(results, result{int32(id), 100})
		} else if re.MatchString(name) {
			// 匹配位置越靠前分数越高
			idx := strings.Index(nameLower, keywordLower)
			score := 80 - idx
			results = append(results, result{int32(id), score})
		}
	}
	// 按分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	ids := make([]int32, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.id)
	}
	return ids
}
