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

package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	_             IMappingRepo = (*MappingRepo)(nil)
	ErrNilMapping              = errors.New("mapping is nil")
)

const (
	MappingCollectionName = "mapping"
	MappingCacheKeyPrefix = "cache:mapping:name2id:"
)

type IMappingRepo interface {
	GetIDByName(ctx context.Context, mType model.MappingType, name string) (int32, error)
	Insert(ctx context.Context, mapping *model.Mapping) error
	FindByNameAndType(ctx context.Context, name string, mType model.MappingType) (*model.Mapping, error)
	FindMaxCodeByType(ctx context.Context, mType model.MappingType) (int32, error)
	FindAllByType(ctx context.Context, mType model.MappingType) ([]*model.Mapping, error)
}

type MappingRepo struct {
	conn *monc.Model
}

func NewMappingRepo(cfg *config.Config) *MappingRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, MappingCollectionName, cfg.Cache)
	return &MappingRepo{conn: conn}
}

// GetIDByName 根据映射类型和名称获取ID，具备Redis缓存能力
func (r *MappingRepo) GetIDByName(ctx context.Context, mType model.MappingType, name string) (int32, error) {
	// 构造缓存Key: cache:mapping:name2id:1:计算机学院
	cacheKey := fmt.Sprintf("%s%d:%s", MappingCacheKeyPrefix, mType, name)

	var m model.Mapping
	err := r.conn.FindOne(ctx, cacheKey, &m, bson.M{"type": mType, "name": name})
	if err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return m.Code, nil
}

// FindByNameAndType 根据名称和类型查找映射
func (r *MappingRepo) FindByNameAndType(ctx context.Context, name string, mType model.MappingType) (*model.Mapping, error) {
	var m model.Mapping
	err := r.conn.FindOneNoCache(ctx, &m, bson.M{"name": name, "type": mType})
	if err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// FindMaxCodeByType 查找特定类型的最大Code，用于自增
func (r *MappingRepo) FindMaxCodeByType(ctx context.Context, mType model.MappingType) (int32, error) {
	var results []model.Mapping
	// 按照 Code 降序排列，取第一个即为最大值
	opts := options.Find().SetSort(bson.D{{Key: "code", Value: -1}}).SetLimit(1)
	err := r.conn.Find(ctx, &results, bson.M{"type": mType}, opts)
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, nil
	}

	return results[0].Code, nil
}

// FindAllByType 查找特定类型的所有映射
func (r *MappingRepo) FindAllByType(ctx context.Context, mType model.MappingType) ([]*model.Mapping, error) {
	var results []*model.Mapping
	err := r.conn.Find(ctx, &results, bson.M{"type": mType})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Insert 插入映射数据（通常由管理员在后台或初始化时操作）
func (r *MappingRepo) Insert(ctx context.Context, mapping *model.Mapping) error {
	if mapping == nil {
		return ErrNilMapping
	}

	cacheKey := fmt.Sprintf("%s%d:%s", MappingCacheKeyPrefix, mapping.Type, mapping.Name)
	_, err := r.conn.InsertOne(ctx, cacheKey, mapping)
	return err
}
