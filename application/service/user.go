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
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	usernameMaxRunes = 15
	usernameCooldown = 30 * 24 * time.Hour
)

var _ IUserService = (*UserService)(nil)

type IUserService interface {
	GetUserProfile(ctx context.Context) (*dto.GetUserProfileResp, error)
	GetUsernameByUserID(ctx context.Context, userID, proposalID string) (*dto.GetUsernameByUserIDResp, error)
	UpdateUserProfile(ctx context.Context, req *dto.UpdateUserProfileReq) (*dto.UpdateUserProfileResp, error)
}

type UserService struct {
	UserRepo     *repo.UserRepo
	ProposalRepo *repo.ProposalRepo
}

var UserServiceSet = wire.NewSet(
	wire.Struct(new(UserService), "*"),
	wire.Bind(new(IUserService), new(*UserService)),
)

// GetUserProfile 获取当前登录用户资料及其当天提案额度使用情况。
func (s *UserService) GetUserProfile(ctx context.Context) (*dto.GetUserProfileResp, error) {
	userID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [FindByID] error: %v, userId: %s", err, userID)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}
	if user == nil {
		return nil, errorx.New(errno.ErrUserNotFound,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}

	todayCount, err := s.ProposalRepo.CountByUserToday(ctx, userID)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [CountByUserToday] error: %v, userId: %s", err, userID)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCountFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}

	return &dto.GetUserProfileResp{
		Resp:            dto.Success(),
		Username:        user.Username,
		Avatar:          user.Avatar,
		Contribution:    user.Contribution,
		DailyQuota:      todayCount,
		DailyQuotaLimit: getDailyProposalLimit(user.Contribution),
		CanEditUsername: canEditUsername(user.UsernameUpdatedAt, time.Now()),
	}, nil
}

// GetUsernameByUserID 获取指定用户的昵称。
func (s *UserService) GetUsernameByUserID(ctx context.Context, userID, proposalID string) (*dto.GetUsernameByUserIDResp, error) {
	currentUserID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || currentUserID == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}
	if currentUserID != userID {
		isAdmin, adminErr := s.UserRepo.IsAdminByID(ctx, currentUserID)
		if adminErr != nil {
			return nil, errorx.WrapByCode(adminErr, errno.ErrUserFindFailed,
				errorx.KV("key", consts.CtxUserID), errorx.KV("value", currentUserID))
		}
		if !isAdmin {
			proposal, proposalErr := s.ProposalRepo.FindByID(ctx, proposalID)
			if proposalErr != nil {
				return nil, errorx.WrapByCode(proposalErr, errno.ErrProposalFindFailed)
			}
			if proposal == nil || proposal.UserID != userID || !proposal.ShowUsername {
				return nil, errorx.New(errno.ErrUserNotOwner, errorx.KV("id", currentUserID))
			}
		}
	}

	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [FindByID] error: %v, userId: %s", err, userID)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}
	if user == nil {
		return nil, errorx.New(errno.ErrUserNotFound,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}

	return &dto.GetUsernameByUserIDResp{
		Resp:     dto.Success(),
		Username: user.Username,
	}, nil
}

// UpdateUserProfile 原子更新当前登录用户实际变更的昵称和头像。
func (s *UserService) UpdateUserProfile(ctx context.Context, req *dto.UpdateUserProfileReq) (*dto.UpdateUserProfileResp, error) {
	userID, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userID == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logs.CtxErrorf(ctx, "[UserRepo] [FindByID] error: %v, userId: %s", err, userID)
		return nil, errorx.WrapByCode(err, errno.ErrUserFindFailed,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}
	if user == nil {
		return nil, errorx.New(errno.ErrUserNotFound,
			errorx.KV("key", consts.CtxUserID), errorx.KV("value", userID))
	}

	var usernameUpdate *string
	var usernameUpdatedAt *time.Time
	usernameUpdate, usernameUpdatedAt, err = prepareNicknameUpdate(
		user.Username, user.UsernameUpdatedAt, req.Username, time.Now(),
	)
	if err != nil {
		return nil, err
	}
	if usernameUpdate != nil && *usernameUpdate != "" {
		exists, findErr := s.UserRepo.IsUsernameExist(ctx, *usernameUpdate, userID)
		if findErr != nil {
			logs.CtxErrorf(ctx, "[UserRepo] [IsUsernameExist] error: %v, username: %s", findErr, *usernameUpdate)
			return nil, errorx.WrapByCode(findErr, errno.ErrUserFindFailed,
				errorx.KV("key", consts.Username), errorx.KV("value", *usernameUpdate))
		}
		if exists {
			return nil, errorx.New(errno.ErrUsernameAlreadyTaken,
				errorx.KV("username", *usernameUpdate))
		}
	}

	var avatarUpdate *string
	if req.Avatar != nil && *req.Avatar != user.Avatar {
		avatar := *req.Avatar
		avatarUpdate = &avatar
	}

	if usernameUpdate == nil && avatarUpdate == nil {
		return &dto.UpdateUserProfileResp{
			Resp:     dto.Success(),
			Username: user.Username,
			Avatar:   user.Avatar,
		}, nil
	}

	var expectedUsernameUpdatedAt *time.Time
	if usernameUpdate != nil {
		expected := user.UsernameUpdatedAt
		expectedUsernameUpdatedAt = &expected
	}
	matched, updateErr := s.UserRepo.UpdateProfile(ctx, userID, usernameUpdate, avatarUpdate, usernameUpdatedAt, expectedUsernameUpdatedAt)
	if updateErr != nil {
		err = updateErr
		if mongo.IsDuplicateKeyError(err) && usernameUpdate != nil {
			return nil, errorx.New(errno.ErrUsernameAlreadyTaken,
				errorx.KV("username", *usernameUpdate))
		}
		logs.CtxErrorf(ctx, "[UserRepo] [UpdateProfile] error: %v, userId: %s", err, userID)
		return nil, errorx.WrapByCode(err, errno.ErrUserUpdateFailed, errorx.KV("id", userID))
	}
	if !matched {
		if usernameUpdate != nil && *usernameUpdate != "" {
			return nil, errorx.New(errno.ErrUsernameUpdateLimitExceeded)
		}
		return nil, errorx.New(errno.ErrUserUpdateFailed, errorx.KV("id", userID))
	}

	if usernameUpdate != nil {
		user.Username = *usernameUpdate
	}
	if avatarUpdate != nil {
		user.Avatar = *avatarUpdate
	}

	return &dto.UpdateUserProfileResp{
		Resp:     dto.Success(),
		Username: user.Username,
		Avatar:   user.Avatar,
	}, nil
}

func normalizeNickname(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	nickname := strings.TrimSpace(raw)
	if nickname == "" {
		return "", errorx.New(errno.ErrUsernameInvalid, errorx.KV("reason", "blank"))
	}
	if !utf8.ValidString(nickname) {
		return "", errorx.New(errno.ErrUsernameInvalid, errorx.KV("reason", "invalid unicode"))
	}
	if utf8.RuneCountInString(nickname) > usernameMaxRunes {
		return "", errorx.New(errno.ErrUsernameInvalid, errorx.KV("reason", "too long"))
	}
	for _, r := range nickname {
		if unicode.IsControl(r) {
			return "", errorx.New(errno.ErrUsernameInvalid, errorx.KV("reason", "control character"))
		}
	}
	return nickname, nil
}

func prepareNicknameUpdate(
	current string,
	updatedAt time.Time,
	requested *string,
	now time.Time,
) (*string, *time.Time, error) {
	if requested == nil {
		return nil, nil, nil
	}

	candidate, err := normalizeNickname(*requested)
	if err != nil {
		return nil, nil, err
	}
	if candidate == strings.TrimSpace(current) {
		return nil, nil, nil
	}
	if candidate == "" {
		return &candidate, nil, nil
	}
	if !canEditUsername(updatedAt, now) {
		return nil, nil, errorx.New(errno.ErrUsernameUpdateLimitExceeded)
	}
	return &candidate, &now, nil
}

func canEditUsername(updatedAt, now time.Time) bool {
	return updatedAt.IsZero() || !now.Before(updatedAt.Add(usernameCooldown))
}
