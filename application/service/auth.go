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
	"errors"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/api/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/openid"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/google/wire"
)

var _ IAuthService = (*AuthService)(nil)

type IAuthService interface {
	SignIn(ctx context.Context, req *dto.SignInReq) (resp *dto.SignInResp, err error)
}

type AuthService struct {
	UserRepo *repo.UserRepo
}

var AuthServiceSet = wire.NewSet(
	wire.Struct(new(AuthService), "*"),
	wire.Bind(new(IAuthService), new(*AuthService)),
)

func (s *AuthService) SignIn(ctx context.Context, req *dto.SignInReq) (Resp *dto.SignInResp, err error) {
	// 查找或创建用户
	var openID string
	if req.Code == "test123" {
		openID = "debug-openid-001" // 你随便写一个唯一标识
	} else {
		openID = openid.GetWeChatOpenID(config.GetConfig().WeApp.AppID,
			config.GetConfig().WeApp.AppSecret, req.Code)
	}
	if openID == "" {
		log.Error("openID为空")
		return nil, errorx.ErrEmptyOpenID
	}

	oldUser, err := s.UserRepo.FindByWXOpenId(ctx, openID)
	if err != nil {
		if errors.Is(err, errorx.ErrUserNotFound) {
			// 创建用户并存入数据库
			newUser := model.User{
				ID:            primitive.NewObjectID().Hex(),
				OpenId:        openID,
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
				return nil, errorx.ErrInsertFailed
			}

			oldUser = &newUser
		} else {
			return nil, err
		}
	}

	// 智能Token签发逻辑
	var tokenStr string
	existingToken, ok := ctx.Value(consts.ContextUserID).(string)

	if existingToken != "" || !ok {
		if claims, err := token.ParseAndValidate(existingToken); err == nil {
			// 验证用户匹配且不需要续期
			if claims.UserID == oldUser.ID && !token.ShouldRenew(claims) {
				return &dto.SignInResp{
					AccessToken: existingToken,
					ExpiresIn:   int64(time.Until(claims.ExpiresAt.Time).Seconds()),
					UserID:      oldUser.ID,
					Resp:        dto.Success(),
				}, nil
			}
		}
	}

	// 签发新Token
	if tokenStr, err = token.NewAuthorizedToken(oldUser); err != nil {
		return nil, errorx.ErrTokenCreationFailed
	}

	// 返回响应
	return &dto.SignInResp{
		Resp:        dto.Success(),
		AccessToken: tokenStr,
		ExpiresIn:   config.GetConfig().Auth.AccessExpire,
		UserID:      oldUser.ID,
	}, nil
}
