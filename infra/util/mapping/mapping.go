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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	typemapping "github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"go.mongodb.org/mongo-driver/mongo"
)

// StaticData retains static process-local maps for immutable enum-like values.
// Campus, department, and category maps are only fallbacks before runtime
// dependencies are initialized (primarily unit tests and migration seeds).
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

	mappingRepo  *repo.MappingRepo
	mappingCache *cache.MappingCache
	mutex        sync.RWMutex
}

var Data = newStaticData()

func newStaticData() *StaticData {
	d := &StaticData{
		CampusNameByID:              cloneMap(typemapping.CampusesMap),
		DepartmentNameByID:          cloneMap(typemapping.DepartmentsMap),
		CategoryNameByID:            cloneMap(typemapping.CategoriesMap),
		ProposalStatusNameByID:      cloneMap(typemapping.ProposalStatusMap),
		LikeTargetTypeNameByID:      cloneMap(typemapping.LikeTargetTypeMap),
		ChangeLogTargetTypeNameByID: cloneMap(typemapping.ChangeLogTargetTypeMap),
	}
	d.CampusIDByName = reverseMap(d.CampusNameByID)
	d.DepartmentIDByName = reverseMap(d.DepartmentNameByID)
	d.CategoryIDByName = reverseMap(d.CategoryNameByID)
	d.ProposalStatusIDByName = reverseMap(d.ProposalStatusNameByID)
	d.LikeTargetTypeIDByName = reverseMap(d.LikeTargetTypeNameByID)
	d.ChangeLogTargetTypeIDByName = reverseMap(d.ChangeLogTargetTypeNameByID)
	return d
}

func cloneMap(source map[int32]string) map[int32]string {
	result := make(map[int32]string, len(source))
	for code, name := range source {
		result[code] = name
	}
	return result
}

func reverseMap(source map[int32]string) map[string]int32 {
	result := make(map[string]int32, len(source))
	for code, name := range source {
		result[name] = code
	}
	return result
}

type MappingDependencies struct {
	MappingRepo  *repo.MappingRepo
	MappingCache *cache.MappingCache
}

// InitWithDependencies loads MongoDB's complete mapping set, warms Redis using
// temporary hashes, and only then exposes the new in-process fuzzy-search snapshot.
func (d *StaticData) InitWithDependencies(ctx context.Context, deps *MappingDependencies) error {
	if deps == nil || deps.MappingRepo == nil || deps.MappingCache == nil {
		return errors.New("mapping dependencies are incomplete")
	}
	d.mutex.Lock()
	d.mappingRepo = deps.MappingRepo
	d.mappingCache = deps.MappingCache
	d.mutex.Unlock()
	return d.Refresh(ctx)
}

// Refresh rebuilds Redis and the local fuzzy-search snapshot from MongoDB. This
// is also called after a transaction creates new reference mappings.
func (d *StaticData) Refresh(ctx context.Context) error {
	d.mutex.RLock()
	mappingRepo := d.mappingRepo
	mappingCache := d.mappingCache
	d.mutex.RUnlock()
	if mappingRepo == nil || mappingCache == nil {
		return errors.New("mapping dependencies are not initialized")
	}
	if err := mappingRepo.SyncCounters(ctx); err != nil {
		return fmt.Errorf("sync mapping counters: %w", err)
	}
	mappings, err := mappingRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load mappings: %w", err)
	}
	if err := validateMappings(mappings); err != nil {
		return err
	}
	if err := mappingCache.Warm(ctx, mappings); err != nil {
		return fmt.Errorf("warm mapping cache: %w", err)
	}

	campusByID, departmentByID, categoryByID := map[int32]string{}, map[int32]string{}, map[int32]string{}
	campusByName, departmentByName, categoryByName := map[string]int32{}, map[string]int32{}, map[string]int32{}
	for _, mapping := range mappings {
		switch mapping.Type {
		case model.MappingTypeCampus:
			campusByID[mapping.Code] = mapping.Name
			if mapping.Canonical {
				campusByName[mapping.Name] = mapping.Code
			}
		case model.MappingTypeDepartment:
			departmentByID[mapping.Code] = mapping.Name
			if mapping.Canonical {
				departmentByName[mapping.Name] = mapping.Code
			}
		case model.MappingTypeCategory:
			categoryByID[mapping.Code] = mapping.Name
			if mapping.Canonical {
				categoryByName[mapping.Name] = mapping.Code
			}
		}
	}

	d.mutex.Lock()
	d.CampusNameByID, d.CampusIDByName = campusByID, campusByName
	d.DepartmentNameByID, d.DepartmentIDByName = departmentByID, departmentByName
	d.CategoryNameByID, d.CategoryIDByName = categoryByID, categoryByName
	d.mutex.Unlock()
	logs.Infof("[Mapping] warmed %d MongoDB mappings into Redis", len(mappings))
	return nil
}

func validateMappings(mappings []*model.Mapping) error {
	canonicalCount := make(map[string]int, len(mappings))
	seenCodes := make(map[string]string, len(mappings))
	for _, mapping := range mappings {
		if mapping == nil || mapping.Code <= 0 || strings.TrimSpace(mapping.Name) == "" {
			return fmt.Errorf("invalid mapping record: %+v", mapping)
		}
		if mapping.Type < model.MappingTypeDepartment || mapping.Type > model.MappingTypeCampus {
			return fmt.Errorf("unsupported mapping type %d", mapping.Type)
		}
		nameKey := fmt.Sprintf("%d:%s", mapping.Type, mapping.Name)
		codeKey := fmt.Sprintf("%d:%d", mapping.Type, mapping.Code)
		if oldName, ok := seenCodes[codeKey]; ok && oldName != mapping.Name {
			return fmt.Errorf("mapping conflict: type=%d code=%d has names %q and %q", mapping.Type, mapping.Code, oldName, mapping.Name)
		}
		if mapping.Canonical {
			canonicalCount[nameKey]++
		}
		seenCodes[codeKey] = mapping.Name
	}
	for nameKey, count := range canonicalCount {
		if count != 1 {
			return fmt.Errorf("mapping conflict: %s has %d canonical codes", nameKey, count)
		}
	}
	uniqueNames := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		uniqueNames[fmt.Sprintf("%d:%s", mapping.Type, mapping.Name)] = struct{}{}
	}
	for nameKey := range uniqueNames {
		if canonicalCount[nameKey] != 1 {
			return fmt.Errorf("mapping conflict: %s has no single canonical code", nameKey)
		}
	}
	return nil
}

func (d *StaticData) GetCampusNameByID(id int32) string {
	return d.getNameByCode(model.MappingTypeCampus, id, "未知校区")
}

func (d *StaticData) GetDepartmentNameByID(id int32) string {
	return d.getNameByCode(model.MappingTypeDepartment, id, "未知开课院系")
}

func (d *StaticData) GetCategoryNameByID(id int32) string {
	return d.getNameByCode(model.MappingTypeCategory, id, "未知分类")
}

func (d *StaticData) getNameByCode(mappingType model.MappingType, code int32, unknown string) string {
	if code <= 0 {
		return unknown
	}
	d.mutex.RLock()
	mappingRepo, mappingCache := d.mappingRepo, d.mappingCache
	d.mutex.RUnlock()
	if mappingRepo == nil || mappingCache == nil {
		if name, ok := d.snapshotName(mappingType, code); ok {
			return name
		}
		return unknown
	}
	ctx := context.Background()
	if name, hit, err := mappingCache.GetNameByCode(ctx, mappingType, code); err == nil && hit {
		return name
	}
	mapping, err := mappingRepo.FindByCodeAndType(ctx, code, mappingType)
	if err != nil || mapping == nil {
		return unknown
	}
	if err := mappingCache.SetMapping(ctx, mapping); err != nil {
		logs.Errorf("[Mapping] refill code cache: %v", err)
	}
	d.updateSnapshot(mapping)
	return mapping.Name
}

func (d *StaticData) GetCampusIDByName(name string) int32 {
	return d.getCodeByName(model.MappingTypeCampus, name)
}

func (d *StaticData) GetDepartmentIDByName(name string) int32 {
	return d.getCodeByName(model.MappingTypeDepartment, name)
}

func (d *StaticData) GetCategoryIDByName(name string) int32 {
	return d.getCodeByName(model.MappingTypeCategory, name)
}

func (d *StaticData) getCodeByName(mappingType model.MappingType, name string) int32 {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	d.mutex.RLock()
	mappingRepo, mappingCache := d.mappingRepo, d.mappingCache
	d.mutex.RUnlock()
	if mappingRepo == nil || mappingCache == nil {
		if code, ok := d.snapshotCode(mappingType, name); ok {
			return code
		}
		return 0
	}
	ctx := context.Background()
	if code, hit, err := mappingCache.GetCodeByKey(ctx, mappingType, name); err == nil && hit {
		return code
	}
	mapping, err := mappingRepo.FindByNameAndType(ctx, name, mappingType)
	if err != nil || mapping == nil {
		return 0
	}
	if err := mappingCache.SetMapping(ctx, mapping); err != nil {
		logs.Errorf("[Mapping] refill name cache: %v", err)
	}
	d.updateSnapshot(mapping)
	return mapping.Code
}

// GetNamesByCodes returns values in the same order as codes and uses HMGET for
// the common path. Only misses are fetched from MongoDB in one query.
func (d *StaticData) GetNamesByCodes(ctx context.Context, mappingType model.MappingType, codes []int32, unknown string) []string {
	result := make([]string, len(codes))
	if len(codes) == 0 {
		return result
	}
	d.mutex.RLock()
	mappingRepo, mappingCache := d.mappingRepo, d.mappingCache
	d.mutex.RUnlock()
	if mappingRepo == nil || mappingCache == nil {
		for i, code := range codes {
			if name, ok := d.snapshotName(mappingType, code); ok {
				result[i] = name
			} else {
				result[i] = unknown
			}
		}
		return result
	}

	values, err := mappingCache.GetNamesByCodes(ctx, mappingType, codes)
	missing := make([]int32, 0)
	seenMissing := make(map[int32]struct{})
	for i, code := range codes {
		if err == nil && i < len(values) && values[i] != "" {
			result[i] = values[i]
			continue
		}
		if _, seen := seenMissing[code]; !seen {
			missing = append(missing, code)
			seenMissing[code] = struct{}{}
		}
	}
	if len(missing) > 0 {
		mappings, findErr := mappingRepo.FindByCodes(ctx, mappingType, missing)
		if findErr == nil {
			byCode := make(map[int32]string, len(mappings))
			for _, mapping := range mappings {
				byCode[mapping.Code] = mapping.Name
				d.updateSnapshot(mapping)
				if cacheErr := mappingCache.SetMapping(ctx, mapping); cacheErr != nil {
					logs.Errorf("[Mapping] refill batch cache: %v", cacheErr)
				}
			}
			for i, code := range codes {
				if result[i] == "" {
					result[i] = byCode[code]
				}
			}
		}
	}
	for i := range result {
		if result[i] == "" {
			result[i] = unknown
		}
	}
	return result
}

// ResolveOrCreateDepartment and ResolveOrCreateCategory persist to MongoDB first.
// Redis is updated only outside a MongoDB transaction; callers refresh after commit.
func (d *StaticData) ResolveOrCreateDepartment(ctx context.Context, name string) (int32, error) {
	return d.resolveOrCreate(ctx, model.MappingTypeDepartment, name)
}

func (d *StaticData) ResolveOrCreateCategory(ctx context.Context, name string) (int32, error) {
	return d.resolveOrCreate(ctx, model.MappingTypeCategory, name)
}

func (d *StaticData) ResolveCampus(ctx context.Context, name string) (int32, error) {
	name = strings.TrimSpace(name)
	d.mutex.RLock()
	mappingRepo := d.mappingRepo
	d.mutex.RUnlock()
	if mappingRepo == nil {
		code := d.GetCampusIDByName(name)
		if code == 0 {
			return 0, fmt.Errorf("unknown campus %q", name)
		}
		return code, nil
	}
	mapping, err := mappingRepo.FindByNameAndType(ctx, name, model.MappingTypeCampus)
	if err != nil {
		return 0, err
	}
	if mapping == nil {
		return 0, fmt.Errorf("unknown campus %q", name)
	}
	return mapping.Code, nil
}

func (d *StaticData) resolveOrCreate(ctx context.Context, mappingType model.MappingType, name string) (int32, error) {
	d.mutex.RLock()
	mappingRepo, mappingCache := d.mappingRepo, d.mappingCache
	d.mutex.RUnlock()
	if mappingRepo == nil {
		return 0, errors.New("mapping repository is not initialized")
	}
	mapping, _, err := mappingRepo.CreateOrGet(ctx, mappingType, name)
	if err != nil {
		return 0, err
	}
	if _, inTransaction := ctx.(mongo.SessionContext); !inTransaction {
		d.updateSnapshot(mapping)
		if mappingCache != nil {
			if err := mappingCache.SetMapping(ctx, mapping); err != nil {
				return mapping.Code, fmt.Errorf("mapping persisted but Redis update failed: %w", err)
			}
		}
	}
	return mapping.Code, nil
}

// Deprecated compatibility wrappers. Transactional proposal paths use the
// context-aware methods above so errors can never be silently converted to code 0.
func (d *StaticData) AutoRegisterDepartment(name string) int32 {
	code, err := d.ResolveOrCreateDepartment(context.Background(), name)
	if err != nil {
		logs.Errorf("[Mapping] register department %q: %v", name, err)
	}
	return code
}

func (d *StaticData) AutoRegisterCategory(name string) int32 {
	code, err := d.ResolveOrCreateCategory(context.Background(), name)
	if err != nil {
		logs.Errorf("[Mapping] register category %q: %v", name, err)
	}
	return code
}

// AutoRegisterCampus intentionally never creates a campus. Campus registration
// is an explicit migration/administrative operation.
func (d *StaticData) AutoRegisterCampus(name string) int32 {
	code, err := d.ResolveCampus(context.Background(), name)
	if err != nil {
		logs.Errorf("[Mapping] resolve campus %q: %v", name, err)
	}
	return code
}

func (d *StaticData) snapshotName(mappingType model.MappingType, code int32) (string, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	switch mappingType {
	case model.MappingTypeCampus:
		name, ok := d.CampusNameByID[code]
		return name, ok
	case model.MappingTypeDepartment:
		name, ok := d.DepartmentNameByID[code]
		return name, ok
	case model.MappingTypeCategory:
		name, ok := d.CategoryNameByID[code]
		return name, ok
	default:
		return "", false
	}
}

func (d *StaticData) snapshotCode(mappingType model.MappingType, name string) (int32, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	switch mappingType {
	case model.MappingTypeCampus:
		code, ok := d.CampusIDByName[name]
		return code, ok
	case model.MappingTypeDepartment:
		code, ok := d.DepartmentIDByName[name]
		return code, ok
	case model.MappingTypeCategory:
		code, ok := d.CategoryIDByName[name]
		return code, ok
	default:
		return 0, false
	}
}

func (d *StaticData) updateSnapshot(mapping *model.Mapping) {
	if mapping == nil {
		return
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	switch mapping.Type {
	case model.MappingTypeCampus:
		d.CampusNameByID[mapping.Code] = mapping.Name
		if mapping.Canonical {
			d.CampusIDByName[mapping.Name] = mapping.Code
		}
	case model.MappingTypeDepartment:
		d.DepartmentNameByID[mapping.Code] = mapping.Name
		if mapping.Canonical {
			d.DepartmentIDByName[mapping.Name] = mapping.Code
		}
	case model.MappingTypeCategory:
		d.CategoryNameByID[mapping.Code] = mapping.Name
		if mapping.Canonical {
			d.CategoryIDByName[mapping.Name] = mapping.Code
		}
	}
}

func (d *StaticData) GetProposalStatusNameByID(id int32) string {
	if name, ok := d.ProposalStatusNameByID[id]; ok {
		return name
	}
	return "未知提案状态"
}

func (d *StaticData) GetLikeTargetTypeNameByID(id int32) string {
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

func (d *StaticData) GetProposalStatusIDByName(name string) int32 {
	return d.ProposalStatusIDByName[name]
}

func (d *StaticData) GetLikeTargetTypeIDByName(name string) int32 {
	return d.LikeTargetTypeIDByName[name]
}

func (d *StaticData) GetChangeLogTargetTypeIDByName(name string) int32 {
	return d.ChangeLogTargetTypeIDByName[name]
}

func (d *StaticData) GetBestCategoryIDByKeyword(keyword string) int32 {
	ids := d.GetCategoryIDsByKeyword(keyword)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func (d *StaticData) GetBestDepartmentIDByKeyword(keyword string) int32 {
	ids := d.GetDepartmentIDsByKeyword(keyword)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func (d *StaticData) GetCategoryIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.mappingMapForSearch(model.MappingTypeCategory))
}

func (d *StaticData) GetDepartmentIDsByKeyword(keyword string) []int32 {
	return fuzzySearch(keyword, d.mappingMapForSearch(model.MappingTypeDepartment))
}

func (d *StaticData) AllCampuses() map[int32]string {
	return d.mappingMapForSearch(model.MappingTypeCampus)
}

func (d *StaticData) mappingMapForSearch(mappingType model.MappingType) map[int32]string {
	d.mutex.RLock()
	mappingRepo, mappingCache := d.mappingRepo, d.mappingCache
	d.mutex.RUnlock()
	if mappingRepo != nil && mappingCache != nil {
		ctx := context.Background()
		if cached, err := mappingCache.GetAllCodeToName(ctx, mappingType); err == nil && len(cached) > 0 {
			result := make(map[int32]string, len(cached))
			for field, name := range cached {
				code, parseErr := strconv.ParseInt(field, 10, 32)
				if parseErr == nil {
					result[int32(code)] = name
				}
			}
			if len(result) > 0 {
				return result
			}
		}
		if mappings, err := mappingRepo.FindAllByType(ctx, mappingType); err == nil {
			result := make(map[int32]string, len(mappings))
			for _, item := range mappings {
				result[item.Code] = item.Name
				if cacheErr := mappingCache.SetMapping(ctx, item); cacheErr != nil {
					logs.Errorf("[Mapping] refill search cache: %v", cacheErr)
				}
			}
			return result
		}
	}
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	switch mappingType {
	case model.MappingTypeCampus:
		return cloneMap(d.CampusNameByID)
	case model.MappingTypeDepartment:
		return cloneMap(d.DepartmentNameByID)
	case model.MappingTypeCategory:
		return cloneMap(d.CategoryNameByID)
	default:
		return map[int32]string{}
	}
}

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
		score int
	}
	results := make([]result, 0)
	for id, name := range dataMap {
		nameLower, keywordLower := strings.ToLower(name), strings.ToLower(keyword)
		if strings.HasPrefix(nameLower, keywordLower) {
			results = append(results, result{id: id, score: 100})
		} else if re.MatchString(name) {
			results = append(results, result{id: id, score: 80 - strings.Index(nameLower, keywordLower)})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].id < results[j].id
		}
		return results[i].score > results[j].score
	})
	ids := make([]int32, len(results))
	for i, result := range results {
		ids[i] = result.id
	}
	return ids
}

// RedisField exposes the decimal representation used by mapping hashes for
// operational diagnostics and migration validation.
func RedisField(code int32) string {
	return strconv.FormatInt(int64(code), 10)
}
