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
	"fmt"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/openid"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/google/wire"
)

var _ IAuthService = (*AuthService)(nil)

type IAuthService interface {
	SignIn(ctx context.Context, req *dto.SignInReq) (resp *dto.SignInResp, err error)
	IsAdmin(ctx context.Context) (resp *dto.IsAdminResp, err error)
	GrantAdmin(ctx context.Context, req *dto.GrantAdminReq) (resp *dto.GrantAdminResp, err error)
}

type AuthService struct {
	UserRepo         *repo.UserRepo
	ChangeLogService IChangeLogService
}

var AuthServiceSet = wire.NewSet(
	wire.Struct(new(AuthService), "*"),
	wire.Bind(new(IAuthService), new(*AuthService)),
)

func (s *AuthService) SignIn(ctx context.Context, req *dto.SignInReq) (Resp *dto.SignInResp, err error) {
	// 查找或创建用户
	var openId string
	if req.VerifyCode == "test123" {
		openId = "debug-openid-001" // 测试环境固定openid
	} else {
		// 为微信API调用设置超时
		openId = openid.GetWeChatOpenID(
			config.GetConfig().WeApp.AppID,
			config.GetConfig().WeApp.AppSecret,
			req.VerifyCode,
		)
	}
	if openId == "" {
		logs.CtxErrorf(ctx, "[AuthService] [SignIn] openid is empty")
		return nil, errorx.New(errno.ErrAuthOpenIDEmpty)
	}

	// 查找用户
	oldUser, err := s.UserRepo.FindByOpenID(ctx, openId)
	if err != nil {
		logs.CtxErrorf(ctx, "[AuthRepo] [FindByOpenID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.ReqOpenID), errorx.KV("value", openId))
	}

	// 用户不存在则创建新用户
	if oldUser == nil {
		newUser := model.User{ // 创建用户并存入数据库
			ID:            primitive.NewObjectID().Hex(),
			OpenID:        openId,
			Admin:         false,
			Email:         "",
			EmailVerified: false,
			Ban:           false,
			Avatar:        "",
			Username:      "",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err = s.UserRepo.Insert(ctx, &newUser); err != nil {
			logs.CtxErrorf(ctx, "[AuthRepo] [Insert] error: %v", err)
			return nil, errorx.WrapByCode(err, errno.ErrUserInsertFailed, errorx.KV("id", newUser.ID))
		}
		oldUser = &newUser
	}

	// 智能Token签发逻辑
	var tokenStr string
	existingToken, ok := ctx.Value(consts.CtxToken).(string)
	if ok && existingToken != "" {
		if claims, err := token.ParseAndValidate(existingToken); err == nil {
			// 验证用户匹配且不需要续期
			if claims.UserID == oldUser.ID && !token.ShouldRenew(claims) {
				return &dto.SignInResp{
					AccessToken: existingToken,
					ExpiresIn:   int64(time.Until(claims.ExpiresAt.Time).Seconds()),
					UserID:      oldUser.ID,
					IsAdmin:     oldUser.Admin,
					Resp:        dto.Success(),
				}, nil
			}
		} else {
			logs.CtxInfof(ctx, "[Token] [ParseAndValidate] error: %v", err)
		}
	}

	// 签发新Token
	if tokenStr, err = token.NewAuthorizedToken(oldUser); err != nil {
		logs.CtxErrorf(ctx, "[Token] [NewAuthorizedToken] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrAuthTokenGenerateFailed)
	}

	return &dto.SignInResp{
		Resp:        dto.Success(),
		AccessToken: tokenStr,
		ExpiresIn:   config.GetConfig().Auth.AccessExpire,
		UserID:      oldUser.ID,
		IsAdmin:     oldUser.Admin,
	}, nil
}

// IsAdmin 判断当前用户是否为管理员
func (s *AuthService) IsAdmin(ctx context.Context) (resp *dto.IsAdminResp, err error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询
	isAdmin, err := s.UserRepo.IsAdminByID(ctx, userId)
	if err != nil {
		logs.CtxErrorf(ctx, "[AuthRepo] [IsAdminByID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userId),
		)
	}

	return &dto.IsAdminResp{
		IsAdmin: isAdmin,
		Resp:    dto.Success(),
	}, nil
}

func (s *AuthService) GrantAdmin(ctx context.Context, req *dto.GrantAdminReq) (resp *dto.GrantAdminResp, err error) {
	// 查询目标用户信息
	targetUser, err := s.UserRepo.FindByID(ctx, req.UserID)
	if err != nil {
		logs.CtxErrorf(ctx, "[AuthRepo] [FindByID] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", req.UserID),
		)
	}
	if targetUser == nil {
		return nil, errorx.New(errno.ErrUserFindFailed, errorx.KV("userId", req.UserID))
	}

	// 获取当前管理员状态
	isAdmin := targetUser.Admin
	
	// 切换管理员状态
	newAdminStatus := !isAdmin
	
	// 更新用户的管理员状态
	if err = s.UserRepo.Update(ctx, &model.User{ID: req.UserID, Admin: newAdminStatus}); err != nil {
		logs.CtxErrorf(ctx, "[AuthRepo] [Update] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrUserUpdateFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", req.UserID),
		)
	}

	// 获取操作者信息
	operatorId, _ := ctx.Value(consts.CtxUserID).(string)
	operatorUser, _ := s.UserRepo.FindByID(ctx, operatorId)
	operatorName := "系统"
	if operatorUser != nil && operatorUser.Username != "" {
		operatorName = operatorUser.Username
	} else if operatorUser != nil && operatorUser.OpenID != "" {
		operatorName = operatorUser.OpenID
	}
	
	// 获取目标用户名称
	targetUserName := targetUser.Username
	if targetUserName == "" {
		targetUserName = targetUser.OpenID
	}
	if targetUserName == "" {
		targetUserName = req.UserID
	}
	
	// 根据操作类型记录不同的日志
	var action int32
	var content string
	if newAdminStatus {
		// 授予管理员权限
		action = 1
		content = fmt.Sprintf("授予管理员权限 | 目标用户: %s(%s) | 状态变更: 普通用户 → 管理员 | 操作者: %s(%s)", 
			targetUserName, req.UserID, operatorName, operatorId)
	} else {
		// 取消管理员权限
		action = 2
		content = fmt.Sprintf("取消管理员权限 | 目标用户: %s(%s) | 状态变更: 管理员 → 普通用户 | 操作者: %s(%s)", 
			targetUserName, req.UserID, operatorName, operatorId)
	}
	
	// 记录管理员操作日志
	_, _ = s.ChangeLogService.CreateChangeLog(ctx, &dto.CreateChangeLogReq{
		TargetID:     req.UserID,
		TargetType:   1, // 用户类型
		Action:       action,
		Content:      content,
		UpdateSource: 1, // 管理后台
	})

	return &dto.GrantAdminResp{
		Resp:    dto.Success(),
		IsAdmin: newAdminStatus, // 返回操作后的状态
	}, nil
}
