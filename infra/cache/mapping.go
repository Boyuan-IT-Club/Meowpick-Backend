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

package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const mappingRedisHashTag = "{reference-mappings}"

type MappingCache struct {
	cache *redis.Redis
}

func NewMappingCache(cfg *config.Config) *MappingCache {
	return &MappingCache{cache: redis.MustNewRedis(*cfg.Redis)}
}

func mappingNameToCodeKey(mappingType model.MappingType) string {
	return fmt.Sprintf("mapping:%s:%d:name_to_code", mappingRedisHashTag, mappingType)
}

func mappingCodeToNameKey(mappingType model.MappingType) string {
	return fmt.Sprintf("mapping:%s:%d:code_to_name", mappingRedisHashTag, mappingType)
}

func (c *MappingCache) GetCodeByKey(ctx context.Context, mappingType model.MappingType, name string) (int32, bool, error) {
	value, err := c.cache.HgetCtx(ctx, mappingNameToCodeKey(mappingType), name)
	if err != nil || value == "" {
		return 0, false, err
	}
	code, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, false, err
	}
	return int32(code), true, nil
}

func (c *MappingCache) GetNameByCode(ctx context.Context, mappingType model.MappingType, code int32) (string, bool, error) {
	value, err := c.cache.HgetCtx(ctx, mappingCodeToNameKey(mappingType), strconv.FormatInt(int64(code), 10))
	if err != nil || value == "" {
		return "", false, err
	}
	return value, true, nil
}

// GetNamesByCodes uses one HMGET so assembling a list of courses does not issue
// one Redis round trip for every field of every course.
func (c *MappingCache) GetNamesByCodes(ctx context.Context, mappingType model.MappingType, codes []int32) ([]string, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	fields := make([]string, len(codes))
	for i, code := range codes {
		fields[i] = strconv.FormatInt(int64(code), 10)
	}
	return c.cache.HmgetCtx(ctx, mappingCodeToNameKey(mappingType), fields...)
}

func (c *MappingCache) GetAllCodeToName(ctx context.Context, mappingType model.MappingType) (map[string]string, error) {
	return c.cache.HgetallCtx(ctx, mappingCodeToNameKey(mappingType))
}

// SetMapping writes both directions in one Lua operation. All mapping hash keys
// share a Redis hash tag, so this also works in Redis Cluster.
func (c *MappingCache) SetMapping(ctx context.Context, mapping *model.Mapping) error {
	const script = `
redis.call('HSET', KEYS[2], ARGV[2], ARGV[1])
if ARGV[3] == '1' then
  redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
end
return 1
`
	canonical := "0"
	if mapping.Canonical {
		canonical = "1"
	}
	_, err := c.cache.EvalCtx(
		ctx,
		script,
		[]string{mappingNameToCodeKey(mapping.Type), mappingCodeToNameKey(mapping.Type)},
		mapping.Name,
		strconv.FormatInt(int64(mapping.Code), 10),
		canonical,
	)
	return err
}

// Warm builds complete temporary hashes and switches all six live hashes in one
// atomic Lua execution. Readers therefore see either the old complete snapshot or
// the new complete snapshot, never a half-populated startup cache.
func (c *MappingCache) Warm(ctx context.Context, mappings []*model.Mapping) error {
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	types := []model.MappingType{
		model.MappingTypeDepartment,
		model.MappingTypeCategory,
		model.MappingTypeCampus,
	}
	tempKeys := make([]string, 0, len(types)*2)
	liveKeys := make([]string, 0, len(types)*2)
	for _, mappingType := range types {
		nameKey := mappingNameToCodeKey(mappingType)
		codeKey := mappingCodeToNameKey(mappingType)
		tempKeys = append(tempKeys, nameKey+":tmp:"+suffix, codeKey+":tmp:"+suffix)
		liveKeys = append(liveKeys, nameKey, codeKey)
	}

	canonicalByName := make(map[string]*model.Mapping, len(mappings))
	for _, mapping := range mappings {
		index := (int(mapping.Type) - 1) * 2
		if index < 0 || index+1 >= len(tempKeys) {
			return fmt.Errorf("unsupported mapping type %d", mapping.Type)
		}
		if err := c.cache.HsetCtx(ctx, tempKeys[index+1], strconv.FormatInt(int64(mapping.Code), 10), mapping.Name); err != nil {
			c.deleteKeys(context.Background(), tempKeys)
			return err
		}
		nameKey := fmt.Sprintf("%d:%s", mapping.Type, mapping.Name)
		current := canonicalByName[nameKey]
		if current == nil || (mapping.Canonical && !current.Canonical) || (mapping.Canonical == current.Canonical && mapping.Code < current.Code) {
			canonicalByName[nameKey] = mapping
		}
	}
	for _, mapping := range canonicalByName {
		index := (int(mapping.Type) - 1) * 2
		if err := c.cache.HsetCtx(ctx, tempKeys[index], mapping.Name, strconv.FormatInt(int64(mapping.Code), 10)); err != nil {
			c.deleteKeys(context.Background(), tempKeys)
			return err
		}
	}

	const switchScript = `
local half = #KEYS / 2
for i = 1, half do
  local temp = KEYS[i]
  local live = KEYS[i + half]
  redis.call('DEL', live)
  if redis.call('EXISTS', temp) == 1 then
    redis.call('RENAME', temp, live)
  end
end
return half
`
	keys := append(append([]string{}, tempKeys...), liveKeys...)
	if _, err := c.cache.EvalCtx(ctx, switchScript, keys); err != nil {
		c.deleteKeys(context.Background(), tempKeys)
		return err
	}
	return nil
}

func (c *MappingCache) deleteKeys(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}
	_, _ = c.cache.DelCtx(ctx, keys...)
}
