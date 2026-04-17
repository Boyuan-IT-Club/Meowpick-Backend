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
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"
	"github.com/Boyuan-IT-Club/go-kit/logs"
)

// StaticData 存放所有静态映射数据
type StaticData struct {
	CampusNameByID              map[int32]string
	DepartmentNameByID          map[int32]string
	CategoryNameByID            map[int32]string
	ProposalStatusNameByID      map[int32]string
	LikeTargetTypeNameByID      map[int32]string
	ChangeLogTargetTypeNameByID map[int32]string
	CampusIDByName              map[string]int32
	DepartmentIDByName          map[string]int32
	CategoryIDByName            map[string]int32
	ProposalStatusIDByName      map[string]int32
	LikeTargetTypeIDByName      map[string]int32
	ChangeLogTargetTypeIDByName map[string]int32
	// 数据库和缓存依赖
	mappingRepo  *repo.MappingRepo
	mappingCache *cache.MappingCache
	mutex        sync.RWMutex // 用于并发安全
}

var Data = &StaticData{
	CampusNameByID:              make(map[int32]string),
	DepartmentNameByID:          make(map[int32]string),
	CategoryNameByID:            make(map[int32]string),
	ProposalStatusNameByID:      make(map[int32]string),
	LikeTargetTypeNameByID:      make(map[int32]string),
	ChangeLogTargetTypeNameByID: make(map[int32]string),
	CampusIDByName:              make(map[string]int32),
	DepartmentIDByName:          make(map[string]int32),
	CategoryIDByName:            make(map[string]int32),
	ProposalStatusIDByName:      make(map[string]int32),
	LikeTargetTypeIDByName:      make(map[string]int32),
	ChangeLogTargetTypeIDByName: make(map[string]int32),
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
	for k, v := range mapping.ChangeLogTargetTypeMap {
		Data.ChangeLogTargetTypeNameByID[k] = v
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
	for id, name := range Data.ChangeLogTargetTypeNameByID {
		Data.ChangeLogTargetTypeIDByName[name] = id
	}
}

// InitWithDependencies 初始化映射工具类的数据库和缓存依赖
type MappingDependencies struct {
	MappingRepo  *repo.MappingRepo
	MappingCache *cache.MappingCache
}

func (d *StaticData) InitWithDependencies(deps *MappingDependencies) {
	if deps == nil {
		logs.Errorf("[Mapping] dependencies is nil")
		return
	}

	d.mappingRepo = deps.MappingRepo
	d.mappingCache = deps.MappingCache

	// 加载数据库中的动态映射数据到内存
	ctx := context.Background()
	if err := d.loadDynamicMappings(ctx); err != nil {
		logs.Errorf("[Mapping] Failed to load dynamic mappings: %v", err)
	}
}

// loadDynamicMappings 从数据库加载动态映射数据到内存
func (d *StaticData) loadDynamicMappings(ctx context.Context) error {
	logs.Info("[Mapping] Starting to load dynamic mappings from database")

	// 加载校区映射
	campusMappings, err := d.mappingRepo.FindAllByType(ctx, model.MappingTypeCampus)
	if err != nil {
		return err
	}
	for _, m := range campusMappings {
		d.CampusNameByID[m.Code] = m.Name
		d.CampusIDByName[m.Name] = m.Code
	}
	logs.Infof("[Mapping] Loaded %d campus mappings", len(campusMappings))

	// 加载院系映射
	deptMappings, err := d.mappingRepo.FindAllByType(ctx, model.MappingTypeDepartment)
	if err != nil {
		return err
	}
	for _, m := range deptMappings {
		d.DepartmentNameByID[m.Code] = m.Name
		d.DepartmentIDByName[m.Name] = m.Code
	}
	logs.Infof("[Mapping] Loaded %d department mappings", len(deptMappings))

	// 加载课程类别映射
	categoryMappings, err := d.mappingRepo.FindAllByType(ctx, model.MappingTypeCategory)
	if err != nil {
		return err
	}
	for _, m := range categoryMappings {
		d.CategoryNameByID[m.Code] = m.Name
		d.CategoryIDByName[m.Name] = m.Code
	}
	logs.Infof("[Mapping] Loaded %d category mappings", len(categoryMappings))

	logs.Info("[Mapping] Dynamic mappings loaded successfully")
	return nil
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

func (d *StaticData) GetChangeLogTargetTypeNameByID(id int32) string {
	if name, ok := d.ChangeLogTargetTypeNameByID[id]; ok {
		return name
	}
	return "未知变更记录类型"
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
	return d.autoRegisterMapping(name, model.MappingTypeDepartment,
		func(id int32, name string) {
			d.DepartmentNameByID[id] = name
			d.DepartmentIDByName[name] = id
		})
}

// AutoRegisterCategory 自动注册不存在的类别，返回其ID
func (d *StaticData) AutoRegisterCategory(name string) int32 {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.autoRegisterMapping(name, model.MappingTypeCategory,
		func(id int32, name string) {
			d.CategoryNameByID[id] = name
			d.CategoryIDByName[name] = id
		})
}

// AutoRegisterCampus 自动注册不存在的校区，返回其ID
func (d *StaticData) AutoRegisterCampus(name string) int32 {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.autoRegisterMapping(name, model.MappingTypeCampus,
		func(id int32, name string) {
			d.CampusNameByID[id] = name
			d.CampusIDByName[name] = id
		})
}

// autoRegisterMapping 通用的自动注册方法
func (d *StaticData) autoRegisterMapping(name string, mappingType model.MappingType, updateFunc func(int32, string)) int32 {
	if name == "" {
		return 0
	}

	ctx := context.Background()

	// 1. 先检查内存中是否已存在
	switch mappingType {
	case model.MappingTypeDepartment:
		if id, ok := d.DepartmentIDByName[name]; ok {
			return id
		}
	case model.MappingTypeCategory:
		if id, ok := d.CategoryIDByName[name]; ok {
			return id
		}
	case model.MappingTypeCampus:
		if id, ok := d.CampusIDByName[name]; ok {
			return id
		}
	}

	// 2. 检查缓存中是否存在
	if d.mappingCache != nil {
		if id, hit, err := d.mappingCache.GetCodeByKey(ctx, mappingType, name); err == nil && hit {
			updateFunc(id, name) // 同步到内存
			return id
		}
	}

	// 3. 查询数据库
	if d.mappingRepo != nil {
		if existing, err := d.mappingRepo.FindByNameAndType(ctx, name, mappingType); err == nil && existing != nil {
			// 存在于数据库，同步到内存和缓存
			updateFunc(existing.Code, name)
			if d.mappingCache != nil {
				d.mappingCache.SetCodeByKey(ctx, mappingType, name, existing.Code, cache.DefaultTTL)
				d.mappingCache.SetNameByKey(ctx, mappingType, existing.Code, name, cache.DefaultTTL)
			}
			return existing.Code
		}
	}

	// 4. 需要创建新的映射
	if d.mappingRepo == nil {
		logs.Errorf("[Mapping] mappingRepo is nil, cannot auto register mapping for type %d, name %s", mappingType, name)
		return 0
	}

	maxCode, err := d.mappingRepo.FindMaxCodeByType(ctx, mappingType)
	if err != nil {
		logs.Errorf("[Mapping] Failed to find max code for type %d: %v", mappingType, err)
		return 0
	}

	newID := maxCode + 1
	newMapping := &model.Mapping{
		Type: mappingType,
		Name: name,
		Code: newID,
	}

	// 保存到数据库
	if err := d.mappingRepo.Insert(ctx, newMapping); err != nil {
		logs.Errorf("[Mapping] Failed to insert new mapping: %v", err)
		return 0
	}

	// 更新内存
	updateFunc(newID, name)

	// 更新缓存
	if d.mappingCache != nil {
		d.mappingCache.SetCodeByKey(ctx, mappingType, name, newID, cache.DefaultTTL)
		d.mappingCache.SetNameByKey(ctx, mappingType, newID, name, cache.DefaultTTL)
		// 清除列表缓存，因为有新数据加入
		d.mappingCache.Invalidate(ctx, mappingType)
	}

	logs.Infof("[Mapping] Auto registered new mapping: type=%d, name=%s, code=%d", mappingType, name, newID)
	return newID
}

// --- 搜索方法（正则+包含匹配）---
func (d *StaticData) GetChangeLogTargetTypeIDByName(name string) int32 {
	if id, ok := d.ChangeLogTargetTypeIDByName[name]; ok {
		return id
	}
	return 0
}

// --- 搜索方法 ---

// GetBestCategoryIDByKeyword 根据关键词获取最匹配的单个分类 ID
func (d *StaticData) GetBestCategoryIDByKeyword(keyword string) int32 {
	ids := d.GetCategoryIDsByKeyword(keyword)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// GetBestDepartmentIDByKeyword 根据关键词获取最匹配的单个部门 ID
func (d *StaticData) GetBestDepartmentIDByKeyword(keyword string) int32 {
	ids := d.GetDepartmentIDsByKeyword(keyword)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// GetCategoryIDsByKeyword 根据类别关键词快速查找匹配的分类 ID
func (d *StaticData) GetCategoryIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.CategoryNameByID)
}

// GetDepartmentIDsByKeyword 根据部门关键词快速查找匹配的院系 ID
func (d *StaticData) GetDepartmentIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.DepartmentNameByID)
}

// 正则 + 包含模糊搜索，按匹配度排序
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
