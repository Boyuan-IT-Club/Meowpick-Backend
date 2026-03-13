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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

var _ IChangeLogRepo = (*ChangeLogRepo)(nil)

const (
	ChangeLogCollectionName = "changelog"
)

type IChangeLogRepo interface {
	Insert(ctx context.Context, changelog *model.ChangeLog) error
}

type ChangeLogRepo struct {
	conn *monc.Model
}

func NewChangeLogRepo(cfg *config.Config) *ChangeLogRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ChangeLogCollectionName, cfg.Cache)
	return &ChangeLogRepo{conn: conn}
}

// Insert 插入变更日志
func (r *ChangeLogRepo) Insert(ctx context.Context, changelog *model.ChangeLog) error {
	_, err := r.conn.InsertOneNoCache(ctx, changelog)
	return err
}
