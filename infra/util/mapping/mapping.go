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
	"regexp"
	"sort"
	"strings"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"
)

// StaticData 存放所有静态映射数据
type StaticData struct {
	CampusNameByID     map[int32]string
	DepartmentNameByID map[int32]string
	CategoryNameByID   map[int32]string
	CampusIDByName     map[string]int32
	DepartmentIDByName map[string]int32
	CategoryIDByName   map[string]int32
}

var Data = &StaticData{
	CampusNameByID:     make(map[int32]string),
	DepartmentNameByID: make(map[int32]string),
	CategoryNameByID:   make(map[int32]string),
	CampusIDByName:     make(map[string]int32),
	DepartmentIDByName: make(map[string]int32),
	CategoryIDByName:   make(map[string]int32),
}

func init() {
	for k, v := range mapping.CampusesMap {
		Data.CampusNameByID[k] = v
	}
	for k, v := range mapping.DepartmentsMap {
		Data.DepartmentNameByID[k] = v
	}
	for k, v := range mapping.CategoriesMap {
		Data.CategoryNameByID[k] = v
	}

	for id, name := range Data.CampusNameByID {
		Data.CampusIDByName[name] = id
	}
	for id, name := range Data.DepartmentNameByID {
		Data.DepartmentIDByName[name] = id
	}
	for id, name := range Data.CategoryNameByID {
		Data.CategoryIDByName[name] = id
	}
}

func (d *StaticData) GetCampusNameByID(id int32) string {
	if name, ok := d.CampusNameByID[id]; ok {
		return name
	}
	return "未知校区"
}

func (d *StaticData) GetDepartmentNameByID(id int32) string {
	if name, ok := d.DepartmentNameByID[id]; ok {
		return name
	}
	return "未知开课院系"
}

func (d *StaticData) GetCategoryNameByID(id int32) string {
	if name, ok := d.CategoryNameByID[id]; ok {
		return name
	}
	return "未知分类"
}

func (d *StaticData) GetCampusIDByName(name string) int32 {
	if id, ok := d.CampusIDByName[name]; ok {
		return id
	}
	return 0
}

func (d *StaticData) GetDepartmentIDByName(name string) int32 {
	if id, ok := d.DepartmentIDByName[name]; ok {
		return id
	}
	return 0
}

func (d *StaticData) GetCategoryIDByName(name string) int32 {
	if id, ok := d.CategoryIDByName[name]; ok {
		return id
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
	return fuzzySearch(keyword, d.CategoryNameByID)
}

// GetDepartmentIDsByKeyword 根据部门关键词快速查找匹配的院系ID
func (d *StaticData) GetDepartmentIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.DepartmentNameByID)
}

// 正则+包含模糊搜索，按匹配度排序
func fuzzySearch(keyword string, dataMap map[int32]string) []int32 {
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
	for id, name := range dataMap {
		nameLower := strings.ToLower(name)
		keywordLower := strings.ToLower(keyword)
		if strings.HasPrefix(nameLower, keywordLower) {
			results = append(results, result{id, 100})
		} else if re.MatchString(name) {
			// 匹配位置越靠前分数越高
			idx := strings.Index(nameLower, keywordLower)
			score := 80 - idx
			results = append(results, result{id, score})
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
