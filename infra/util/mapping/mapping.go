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
	"sync"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"
)

// StaticData 存放所有静态映射数据
type StaticData struct {
	CampusNameByID         map[int32]string
	DepartmentNameByID     map[int32]string
	CategoryNameByID       map[int32]string
	ProposalStatusNameByID map[int32]string
	LikeTargetTypeNameByID map[int32]string
	CampusIDByName         map[string]int32
	DepartmentIDByName     map[string]int32
	CategoryIDByName       map[string]int32
	ProposalStatusIDByName map[string]int32
	LikeTargetTypeIDByName map[string]int32
	mutex                  sync.RWMutex // 用于并发安全
}

var Data = &StaticData{
	CampusNameByID:         make(map[int32]string),
	DepartmentNameByID:     make(map[int32]string),
	CategoryNameByID:       make(map[int32]string),
	ProposalStatusNameByID: make(map[int32]string),
	LikeTargetTypeNameByID: make(map[int32]string),
	CampusIDByName:         make(map[string]int32),
	DepartmentIDByName:     make(map[string]int32),
	CategoryIDByName:       make(map[string]int32),
	ProposalStatusIDByName: make(map[string]int32),
	LikeTargetTypeIDByName: make(map[string]int32),
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
	for k, v := range mapping.ProposalStatusMap {
		Data.ProposalStatusNameByID[k] = v
	}
	for k, v := range mapping.LikeTargetTypeMap {
		Data.LikeTargetTypeNameByID[k] = v
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
	for id, name := range Data.ProposalStatusNameByID {
		Data.ProposalStatusIDByName[name] = id
	}
	for id, name := range Data.LikeTargetTypeNameByID {
		Data.LikeTargetTypeIDByName[name] = id
	}
}

func (d *StaticData) GetCampusNameByID(id int32) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if name, ok := d.CampusNameByID[id]; ok {
		return name
	}
	return "未知校区"
}

func (d *StaticData) GetDepartmentNameByID(id int32) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if name, ok := d.DepartmentNameByID[id]; ok {
		return name
	}
	return "未知开课院系"
}

func (d *StaticData) GetCategoryNameByID(id int32) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if name, ok := d.CategoryNameByID[id]; ok {
		return name
	}
	return "未知分类"
}

func (d *StaticData) GetProposalStatusNameByID(id int32) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if name, ok := d.ProposalStatusNameByID[id]; ok {
		return name
	}
	return "未知提案状态"
}

func (d *StaticData) GetLikeTargetTypeNameByID(id int32) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if name, ok := d.LikeTargetTypeNameByID[id]; ok {
		return name
	}
	return "未知点赞目标类型"
}

func (d *StaticData) GetCampusIDByName(name string) int32 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if id, ok := d.CampusIDByName[name]; ok {
		return id
	}
	return 0
}

func (d *StaticData) GetDepartmentIDByName(name string) int32 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if id, ok := d.DepartmentIDByName[name]; ok {
		return id
	}
	return 0
}

func (d *StaticData) GetCategoryIDByName(name string) int32 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if id, ok := d.CategoryIDByName[name]; ok {
		return id
	}
	return 0
}

func (d *StaticData) GetProposalStatusIDByName(name string) int32 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if id, ok := d.ProposalStatusIDByName[name]; ok {
		return id
	}
	return 0
}

func (d *StaticData) GetLikeTargetTypeIDByName(name string) int32 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if id, ok := d.LikeTargetTypeIDByName[name]; ok {
		return id
	}
	return 0
}

// AutoRegisterDepartment 自动注册不存在的院系，返回其ID
func (d *StaticData) AutoRegisterDepartment(name string) int32 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 检查是否已存在
	if id, ok := d.DepartmentIDByName[name]; ok {
		return id
	}

	// 找到下一个可用的最大ID
	maxID := int32(0)
	for _, id := range d.DepartmentIDByName { // 修复：遍历值而不是键
		if id > maxID {
			maxID = id
		}
	}
	newID := maxID + 1

	// 注册新的院系
	d.DepartmentNameByID[newID] = name
	d.DepartmentIDByName[name] = newID

	return newID
}

// AutoRegisterCategory 自动注册不存在的类别，返回其ID
func (d *StaticData) AutoRegisterCategory(name string) int32 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 检查是否已存在
	if id, ok := d.CategoryIDByName[name]; ok {
		return id
	}

	// 找到下一个可用的最大ID
	maxID := int32(0)
	for _, id := range d.CategoryIDByName { // 修复：遍历值而不是键
		if id > maxID {
			maxID = id
		}
	}
	newID := maxID + 1

	// 注册新的类别
	d.CategoryNameByID[newID] = name
	d.CategoryIDByName[name] = newID

	return newID
}

// AutoRegisterCampus 自动注册不存在的校区，返回其ID
func (d *StaticData) AutoRegisterCampus(name string) int32 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 检查是否已存在
	if id, ok := d.CampusIDByName[name]; ok {
		return id
	}

	// 找到下一个可用的最大ID
	maxID := int32(0)
	for _, id := range d.CampusIDByName { // 修复：遍历值而不是键
		if id > maxID {
			maxID = id
		}
	}
	newID := maxID + 1

	// 注册新的校区
	d.CampusNameByID[newID] = name
	d.CampusIDByName[name] = newID

	return newID
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
