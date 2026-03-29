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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IChangeLogService = (*ChangeLogService)(nil)

type IChangeLogService interface {
	CreateChangeLog(ctx context.Context, req *dto.CreateChangeLogReq) (*dto.CreateChangeLogResp, error)
}

type ChangeLogService struct {
	ChangeLogRepo      *repo.ChangeLogRepo
	ChangeLogAssembler *assembler.ChangeLogAssembler
}

var ChangeLogServiceSet = wire.NewSet(
	wire.Struct(new(ChangeLogService), "*"),
	wire.Bind(new(IChangeLogService), new(*ChangeLogService)),
)

// CreateChangeLog 添加一个新的变更日志
func (s *ChangeLogService) CreateChangeLog(ctx context.Context, req *dto.CreateChangeLogReq) (*dto.CreateChangeLogResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 使用Assembler转换变更日志
	now := time.Now()
	changeLogVO := &dto.ChangeLogVO{
		ID:           primitive.NewObjectID().Hex(),
		TargetID:     req.TargetID,
		TargetType:   req.TargetType,
		Action:       req.Action,
		Content:      req.Content,
		UpdateSource: req.UpdateSource,
		ProposalID:   req.ProposalID,
		UserID:       userId,
		UpdatedAt:    now,
	}

	changeLog, err := s.ChangeLogAssembler.ToChangeLogDB(ctx, changeLogVO)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrChangeLogCvtFailed,
			errorx.KV("src", "changelog vo"), errorx.KV("dst", "database changelog"),
		)
	}

	// 保存变更日志到数据库
	if err = s.ChangeLogRepo.Insert(ctx, changeLog); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrChangeLogInsertFailed, errorx.KV("targetID", req.TargetID))
	}

	return &dto.CreateChangeLogResp{
		Resp:        dto.Success(),
		ChangeLogID: changeLog.ID,
	}, nil
}
