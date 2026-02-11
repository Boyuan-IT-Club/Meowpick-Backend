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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

var _ IChangeLogRepo = (*ChangeLogRepo)(nil)

const (
	ChangeLogCollectionName = "change_log"
)

// IChangeLogRepo 变更记录数据访问接口（入参改为string）
type IChangeLogRepo interface {
	Insert(ctx context.Context, changelog *model.ChangeLog) error
	FindByTarget(ctx context.Context, targetType string, targetID string, param *dto.PageParam) ([]*model.ChangeLog, int64, error) // targetType从int32改string
	FindByID(ctx context.Context, changeLogID string) (*model.ChangeLog, error)
}

// ChangeLogRepo 变更记录数据访问实现
type ChangeLogRepo struct {
	conn *monc.Model
}

// NewChangeLogRepo 初始化
func NewChangeLogRepo(cfg *config.Config) *ChangeLogRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ChangeLogCollectionName, cfg.Cache)
	return &ChangeLogRepo{conn: conn}
}

// Insert 新增变更记录
func (r *ChangeLogRepo) Insert(ctx context.Context, changelog *model.ChangeLog) error {
	_, err := r.conn.InsertOneNoCache(ctx, changelog)
	return err
}

// FindByTarget 分页查询变更记录
func (r *ChangeLogRepo) FindByTarget(ctx context.Context, targetType string, targetID string, param *dto.PageParam) ([]*model.ChangeLog, int64, error) {
	changeLogs := []*model.ChangeLog{}
	// 查询条件
	filter := bson.M{
		consts.TargetType: targetType,
		consts.TargetID:   targetID,
	}

	// 统计总数
	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	if err = r.conn.Find(
		ctx,
		&changeLogs,
		filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
		return nil, 0, err
	}

	return changeLogs, total, nil
}

// FindByID 根据ID查询变更记录
func (r *ChangeLogRepo) FindByID(ctx context.Context, changeLogID string) (*model.ChangeLog, error) {
	changelog := model.ChangeLog{}
	if err := r.conn.FindOneNoCache(ctx, &changelog, bson.M{consts.ID: changeLogID}, nil); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &changelog, nil
}
