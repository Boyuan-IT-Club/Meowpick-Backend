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

package service

import (
	"context"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"

	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IChangeLogService = (*ChangeLogService)(nil)

// IChangeLogService 变更记录业务接口
type IChangeLogService interface {
	ListChangeLogs(ctx context.Context, req *dto.ListChangeLogReq) (*dto.ListChangeLogResp, error)
	CreateChangeLog(ctx context.Context, req *dto.CreateChangeLogReq) (*dto.CreateChangeLogResp, error)
}

// ChangeLogService 变更记录业务实现
type ChangeLogService struct {
	ChangeLogRepo *repo.ChangeLogRepo
}

// ChangeLogServiceSet wire注册
var ChangeLogServiceSet = wire.NewSet(
	wire.Struct(new(ChangeLogService), "*"),
	wire.Bind(new(IChangeLogService), new(*ChangeLogService)),
)

// ListChangeLogs 分页查询变更记录（简化：无类型转换）
func (s *ChangeLogService) ListChangeLogs(ctx context.Context, req *dto.ListChangeLogReq) (*dto.ListChangeLogResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询变更记录
	changeLogs, total, err := s.ChangeLogRepo.FindByTarget(ctx, req.TargetType, req.TargetID, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogRepo] [FindByTarget] error: %v, targetType: %s, targetID: %s", err, req.TargetType, req.TargetID)
		return nil, errorx.WrapByCode(err, errno.ErrChangeLogFindFailed,
			errorx.KV("targetType", req.TargetType),
			errorx.KV("targetId", req.TargetID),
		)
	}

	// 转换为VO（全程string，无需类型处理）
	vos := make([]*dto.ChangeLogVO, 0, len(changeLogs))
	for _, cl := range changeLogs {
		vo := &dto.ChangeLogVO{
			ID:           cl.ID,
			TargetID:     cl.TargetID,
			TargetType:   cl.TargetType,
			Action:       cl.Action,
			Content:      cl.Content,
			UserID:       cl.UserID,
			UpdateSource: cl.UpdateSource,
			ProposalID:   cl.ProposalID,
			CreatedAt:    cl.CreatedAt,
		}
		vos = append(vos, vo)
	}

	// 构造resp
	return &dto.ListChangeLogResp{
		Resp:       dto.Success(),
		Total:      total,
		ChangeLogs: vos,
	}, nil
}

// CreateChangeLog 新增变更记录
func (s *ChangeLogService) CreateChangeLog(ctx context.Context, req *dto.CreateChangeLogReq) (*dto.CreateChangeLogResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 转换为Model
	now := time.Now()
	changelog := &model.ChangeLog{
		ID:           primitive.NewObjectID().Hex(),
		TargetID:     req.TargetID,
		TargetType:   req.TargetType,
		Action:       req.Action,
		Content:      req.Content,
		UserID:       userId,
		UpdateSource: req.UpdateSource,
		ProposalID:   req.ProposalID,
		CreatedAt:    now,
	}

	// 保存到数据库
	if err := s.ChangeLogRepo.Insert(ctx, changelog); err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogRepo] [Insert] error: %v, targetType: %s, targetID: %s", err, req.TargetType, req.TargetID)
		return nil, errorx.WrapByCode(err, errno.ErrChangeLogInsertFailed,
			errorx.KV("targetType", req.TargetType),
			errorx.KV("targetId", req.TargetID),
		)
	}

	// 构造resp
	return &dto.CreateChangeLogResp{
		Resp:        dto.Success(),
		ChangeLogID: changelog.ID,
	}, nil
}
