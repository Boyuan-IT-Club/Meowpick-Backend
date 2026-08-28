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
	"strings"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ IMappingRepo = (*MappingRepo)(nil)

const (
	MappingCollectionName        = "mapping"
	mappingCounterCollectionName = "mapping_counter"
	mappingNameIndexName         = "mapping_type_name_canonical_unique"
	mappingCodeIndexName         = "mapping_type_code_unique"
	mappingCreateMaxAttempts     = 8
)

type IMappingRepo interface {
	FindByNameAndType(ctx context.Context, name string, mType model.MappingType) (*model.Mapping, error)
	FindByCodeAndType(ctx context.Context, code int32, mType model.MappingType) (*model.Mapping, error)
	FindByCodes(ctx context.Context, mType model.MappingType, codes []int32) ([]*model.Mapping, error)
	FindAll(ctx context.Context) ([]*model.Mapping, error)
	FindAllByType(ctx context.Context, mType model.MappingType) ([]*model.Mapping, error)
	SyncCounters(ctx context.Context) error
	CreateOrGet(ctx context.Context, mType model.MappingType, name string) (*model.Mapping, bool, error)
}

// MappingRepo deliberately bypasses monc's per-document string cache. Mapping data
// is cached as bidirectional Redis hashes by MappingCache, while MongoDB remains the
// sole source of truth.
type MappingRepo struct {
	conn     *monc.Model
	counters *mongo.Collection
}

func NewMappingRepo(cfg *config.Config) (*MappingRepo, error) {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, MappingCollectionName, cfg.Cache)
	r := &MappingRepo{
		conn:     conn,
		counters: conn.Database().Collection(mappingCounterCollectionName),
	}
	if err := r.ensureIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure mapping indexes: %w", err)
	}
	return r, nil
}

func (r *MappingRepo) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "type", Value: 1}, {Key: "name", Value: 1}},
			Options: options.Index().SetName(mappingNameIndexName).SetUnique(true).
				SetPartialFilterExpression(bson.M{"canonical": true}),
		},
		{
			Keys:    bson.D{{Key: "type", Value: 1}, {Key: "code", Value: 1}},
			Options: options.Index().SetName(mappingCodeIndexName).SetUnique(true),
		},
	}
	collection, err := r.conn.Clone()
	if err != nil {
		return err
	}
	_, err = collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *MappingRepo) FindByNameAndType(ctx context.Context, name string, mType model.MappingType) (*model.Mapping, error) {
	var mapping model.Mapping
	err := r.conn.FindOneNoCache(ctx, &mapping, bson.M{"name": strings.TrimSpace(name), "type": mType},
		options.FindOne().SetSort(bson.D{{Key: "canonical", Value: -1}, {Key: "code", Value: 1}}))
	if errors.Is(err, monc.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (r *MappingRepo) FindByCodeAndType(ctx context.Context, code int32, mType model.MappingType) (*model.Mapping, error) {
	var mapping model.Mapping
	err := r.conn.FindOneNoCache(ctx, &mapping, bson.M{"code": code, "type": mType})
	if errors.Is(err, monc.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (r *MappingRepo) FindByCodes(ctx context.Context, mType model.MappingType, codes []int32) ([]*model.Mapping, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	var mappings []*model.Mapping
	if err := r.conn.Find(ctx, &mappings, bson.M{"type": mType, "code": bson.M{"$in": codes}}); err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) FindAll(ctx context.Context) ([]*model.Mapping, error) {
	var mappings []*model.Mapping
	if err := r.conn.Find(ctx, &mappings, bson.M{}); err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) FindAllByType(ctx context.Context, mType model.MappingType) ([]*model.Mapping, error) {
	var mappings []*model.Mapping
	if err := r.conn.Find(ctx, &mappings, bson.M{"type": mType}); err != nil {
		return nil, err
	}
	return mappings, nil
}

// SyncCounters advances every sequence to at least the greatest code already in
// MongoDB. It never decreases a sequence, so restarts cannot reuse an old code.
func (r *MappingRepo) SyncCounters(ctx context.Context) error {
	for _, mType := range []model.MappingType{
		model.MappingTypeDepartment,
		model.MappingTypeCategory,
		model.MappingTypeCampus,
	} {
		var latest model.Mapping
		err := r.conn.FindOneNoCache(ctx, &latest, bson.M{"type": mType}, options.FindOne().SetSort(bson.D{{Key: "code", Value: -1}}))
		if errors.Is(err, monc.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		_, err = r.counters.UpdateOne(ctx, bson.M{"_id": mType}, bson.M{"$max": bson.M{"seq": latest.Code}}, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateOrGet allocates a code atomically. Concurrent requests for the same name
// converge on the unique (type,name) record; code collisions are retried.
func (r *MappingRepo) CreateOrGet(ctx context.Context, mType model.MappingType, name string) (*model.Mapping, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false, errors.New("mapping name cannot be empty")
	}

	existing, err := r.FindByNameAndType(ctx, name, mType)
	if err != nil || existing != nil {
		return existing, false, err
	}

	for attempt := 0; attempt < mappingCreateMaxAttempts; attempt++ {
		var counter struct {
			Seq int32 `bson:"seq"`
		}
		err = r.counters.FindOneAndUpdate(
			ctx,
			bson.M{"_id": mType},
			bson.M{"$inc": bson.M{"seq": 1}},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		).Decode(&counter)
		if err != nil {
			return nil, false, err
		}

		mapping := &model.Mapping{ID: primitive.NewObjectID(), Type: mType, Name: name, Code: counter.Seq, Canonical: true}
		if _, err = r.conn.InsertOneNoCache(ctx, mapping); err == nil {
			return mapping, true, nil
		}
		if !mongo.IsDuplicateKeyError(err) {
			return nil, false, err
		}
		if sameName, findErr := r.FindByNameAndType(ctx, name, mType); findErr != nil {
			return nil, false, findErr
		} else if sameName != nil {
			return sameName, false, nil
		}
	}

	return nil, false, fmt.Errorf("allocate mapping code after %d attempts", mappingCreateMaxAttempts)
}
