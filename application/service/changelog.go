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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
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
	ListProposalLogsGrouped(ctx context.Context, req *dto.ListProposalLogsGroupedReq) (*dto.ListProposalLogsGroupedResp, error)
	ListProposalLogsTimeline(ctx context.Context, req *dto.ListProposalLogsTimelineReq) (*dto.ListProposalLogsTimelineResp, error)
}

type ChangeLogService struct {
	ChangeLogRepo      *repo.ChangeLogRepo
	ChangeLogAssembler *assembler.ChangeLogAssembler
	UserRepo           *repo.UserRepo
	ProposalRepo       *repo.ProposalRepo
	CourseAssembler    *assembler.CourseAssembler
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

// ListProposalLogsGrouped 按提案聚合的日志列表
func (s *ChangeLogService) ListProposalLogsGrouped(ctx context.Context, req *dto.ListProposalLogsGroupedReq) (*dto.ListProposalLogsGroupedResp, error) {
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

	// 查询所有提案
	proposals, total, err := s.ProposalRepo.FindMany(ctx, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindMany] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
	}

	// 提取所有提案ID
	proposalIds := make([]string, len(proposals))
	for i, p := range proposals {
		proposalIds[i] = p.ID
	}

	// 批量查询提案相关的管理员操作日志
	adminLogs, err := s.ChangeLogRepo.FindByProposalIDs(ctx, proposalIds)
	if err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogRepo] [FindByProposalIDs] error: %v", err)
		return nil, err
	}

	// 构建提案ID到管理员操作的映射
	adminActionMap := make(map[string]*model.ChangeLog)
	for _, log := range adminLogs {
		if log.ProposalID != "" {
			// 只保留最新的管理员操作
			if existing, ok := adminActionMap[log.ProposalID]; !ok || log.UpdatedAt.After(existing.UpdatedAt) {
				adminActionMap[log.ProposalID] = log
			}
		}
	}

	// 组装返回数据
	proposalVOs := make([]*dto.ProposalLogVO, 0, len(proposals))

	// 收集所有需要查询的用户ID
	userIDSet := make(map[string]bool)
	for _, proposal := range proposals {
		userIDSet[proposal.UserID] = true
	}
	for _, log := range adminActionMap {
		userIDSet[log.UserID] = true
	}

	// 批量查询用户信息
	userIDs := make([]string, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	users, err := s.UserRepo.FindByIDs(ctx, userIDs)
	if err != nil {
		logs.CtxWarnf(ctx, "[UserRepo] [FindByIDs] error: %v", err)
	}

	// 构建用户ID到用户信息的映射
	userMap := make(map[string]*model.User)
	for _, user := range users {
		userMap[user.ID] = user
	}

	for _, proposal := range proposals {
		// 从映射中获取创建者信息
		creatorName := ""
		if creator, ok := userMap[proposal.UserID]; ok {
			creatorName = creator.Username
			if creatorName == "" {
				creatorName = creator.OpenID
			}
		}

		// 转换课程信息
		var courseVO *dto.CourseVO
		if proposal.Course != nil {
			courseVO, err = s.CourseAssembler.ToCourseVO(ctx, proposal.Course)
			if err != nil {
				logs.CtxWarnf(ctx, "[CourseAssembler] [ToCourseVO] error: %v", err)
			}
		}

		proposalVO := &dto.ProposalLogVO{
			ProposalID: proposal.ID,
			Title:      proposal.Title,
			Content:    proposal.Content,
			Status:     s.getProposalStatusName(proposal.Status),
			Course:     courseVO,
			Creator: &dto.CreatorVO{
				CreatorID:   proposal.UserID,
				CreatorName: creatorName,
				CreateTime:  proposal.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		}

		// 添加管理员操作信息
		if adminLog, ok := adminActionMap[proposal.ID]; ok {
			adminName := ""
			if admin, ok := userMap[adminLog.UserID]; ok {
				adminName = admin.Username
				if adminName == "" {
					adminName = admin.OpenID
				}
			}

			proposalVO.AdminAction = &dto.AdminActionVO{
				AdminID:    adminLog.UserID,
				AdminName:  adminName,
				Action:     s.getActionTypeName(adminLog.Action),
				ActionTime: adminLog.UpdatedAt.Format("2006-01-02 15:04:05"),
				Reason:     adminLog.Content,
			}
		}

		proposalVOs = append(proposalVOs, proposalVO)
	}

	return &dto.ListProposalLogsGroupedResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: proposalVOs,
	}, nil
}

// ListProposalLogsTimeline 扁平化时间线日志
func (s *ChangeLogService) ListProposalLogsTimeline(ctx context.Context, req *dto.ListProposalLogsTimelineReq) (*dto.ListProposalLogsTimelineResp, error) {
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

	// 查询所有日志
	logList, total, err := s.ChangeLogRepo.FindMany(ctx, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[ChangeLogRepo] [FindMany] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrChangeLogFindFailed)
	}

	// 提取所有提案ID
	proposalIDSet := make(map[string]bool)
	for _, log := range logList {
		if log.ProposalID != "" {
			proposalIDSet[log.ProposalID] = true
		}
	}

	// 批量查询提案信息
	proposalMap := make(map[string]*model.Proposal)
	if len(proposalIDSet) > 0 {
		proposalIDs := make([]string, 0, len(proposalIDSet))
		for id := range proposalIDSet {
			proposalIDs = append(proposalIDs, id)
		}

		proposals, err := s.ProposalRepo.FindByIDs(ctx, proposalIDs)
		if err != nil {
			logs.CtxWarnf(ctx, "[ProposalRepo] [FindByIDs] error: %v", err)
		} else {
			for _, p := range proposals {
				proposalMap[p.ID] = p
			}
		}
	}

	// 收集所有需要查询的用户ID
	userIDSet := make(map[string]bool)
	for _, log := range logList {
		if log.UserID != "" {
			userIDSet[log.UserID] = true
		}
	}

	// 批量查询用户信息
	userMap := make(map[string]*model.User)
	if len(userIDSet) > 0 {
		userIDs := make([]string, 0, len(userIDSet))
		for id := range userIDSet {
			userIDs = append(userIDs, id)
		}

		users, err := s.UserRepo.FindByIDs(ctx, userIDs)
		if err != nil {
			logs.CtxWarnf(ctx, "[UserRepo] [FindByIDs] error: %v", err)
		} else {
			for _, user := range users {
				userMap[user.ID] = user
			}
		}
	}

	// 组装时间线日志
	timelineLogs := make([]*dto.ProposalTimelineLogVO, 0, len(logList))
	for _, log := range logList {
		// 从映射中获取操作者信息
		operatorName := ""
		if operator, ok := userMap[log.UserID]; ok {
			operatorName = operator.Username
			if operatorName == "" {
				operatorName = operator.OpenID
			}
		}

		timelineLog := &dto.ProposalTimelineLogVO{
			LogID:        log.ID,
			ProposalID:   log.ProposalID,
			ActionType:   s.getTimelineActionType(log.Action),
			OperatorID:   log.UserID,
			OperatorName: operatorName,
			ActionTime:   log.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// 添加提案快照信息
		if log.ProposalID != "" {
			if proposal, ok := proposalMap[log.ProposalID]; ok {
				snapshot := &dto.ProposalSnapshotVO{
					Title: proposal.Title,
				}
				if proposal.Course != nil {
					snapshot.CourseName = proposal.Course.Name
					snapshot.Department = s.getDepartmentName(proposal.Course.Department)
					snapshot.Category = s.getCategoryName(proposal.Course.Category)
				}
				timelineLog.ProposalSnapshot = snapshot
			}
		}

		// 添加详细信息
		if log.Content != "" {
			timelineLog.Details = map[string]interface{}{
				"content": log.Content,
			}
		}

		timelineLogs = append(timelineLogs, timelineLog)
	}

	return &dto.ListProposalLogsTimelineResp{
		Resp:  dto.Success(),
		Total: total,
		Logs:  timelineLogs,
	}, nil
}

// getProposalStatusName 获取提案状态名称
func (s *ChangeLogService) getProposalStatusName(status int32) string {
	switch status {
	case 0:
		return "pending"
	case 1:
		return "approved"
	case 2:
		return "rejected"
	default:
		return "unknown"
	}
}

// getActionTypeName 获取操作类型名称
func (s *ChangeLogService) getActionTypeName(action int32) string {
	switch action {
	case 3:
		return "delete"
	case 4:
		return "update"
	case 5:
		return "approve"
	default:
		return "unknown"
	}
}

// getTimelineActionType 获取时间线操作类型
func (s *ChangeLogService) getTimelineActionType(action int32) string {
	switch action {
	case 1:
		return "GRANT_ADMIN"
	case 2:
		return "REVOKE_ADMIN"
	case 3:
		return "DELETE"
	case 4:
		return "UPDATE"
	case 5:
		return "APPROVE"
	default:
		return "UNKNOWN"
	}
}

// getDepartmentName 获取院系名称
func (s *ChangeLogService) getDepartmentName(id int32) string {
	return mapping.Data.GetDepartmentNameByID(id)
}

// getCategoryName 获取课程类别名称
func (s *ChangeLogService) getCategoryName(id int32) string {
	return mapping.Data.GetCategoryNameByID(id)
}
