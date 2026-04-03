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
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IChangeLogService = (*ChangeLogService)(nil)

type IChangeLogService interface {
	CreateChangeLog(ctx context.Context, req *dto.CreateChangeLogReq) (*dto.CreateChangeLogResp, error)
	ListAdminLogs(ctx context.Context, req *dto.ListAdminLogsReq) (*dto.ListAdminLogsResp, error)
}

type ChangeLogService struct {
	ChangeLogRepo      *repo.ChangeLogRepo
	ChangeLogAssembler *assembler.ChangeLogAssembler
	UserRepo           *repo.UserRepo
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

// ListAdminLogs 查询管理员日志列表
func (s *ChangeLogService) ListAdminLogs(ctx context.Context, req *dto.ListAdminLogsReq) (*dto.ListAdminLogsResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 检查是否是管理员
	isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [IsAdminByID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed)
	}
	if !isAdmin {
		return nil, errorx.New(errno.ErrUserNotAdmin)
	}

	// 设置默认分页参数
	if req.PageParam == nil {
		req.PageParam = &dto.PageParam{
			Page:     1,
			PageSize: 20,
		}
	}

	// 查询所有日志（不做筛选）
	logList, total, err := s.ChangeLogRepo.FindMany(ctx, req.PageParam)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrChangeLogInsertFailed)
	}

	// 转换为VO
	logVOs := make([]*dto.AdminLogVO, 0, len(logList))
	for _, log := range logList {
		// 查询管理员姓名
		adminName := ""
		user, err := s.UserRepo.FindByID(ctx, log.UserID)
		if err == nil && user != nil {
			adminName = user.Username
			if adminName == "" {
				adminName = user.OpenID
			}
		}

		logVOs = append(logVOs, &dto.AdminLogVO{
			ID:         log.ID,
			AdminID:    log.UserID,
			AdminName:  adminName,
			Action:     log.Action,
			ActionName: s.getActionName(log.Action),
			Content:    log.Content,
			TargetType: log.TargetType,
			TargetID:   log.TargetID,
			IP:         log.IP,
			UserAgent:  log.UserAgent,
			CreatedAt:  log.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &dto.ListAdminLogsResp{
		Resp:  dto.Success(),
		Total: total,
		Logs:  logVOs,
	}, nil
}

// getActionName 获取操作名称
func (s *ChangeLogService) getActionName(action int32) string {
	switch action {
	case 1:
		return "授予管理员权限"
	case 2:
		return "删除提案"
	case 3:
		return "更新提案"
	case 4:
		return "审核提案"
	default:
		return "未知操作"
	}
}
