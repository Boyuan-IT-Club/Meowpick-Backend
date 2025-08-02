package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/google/wire"
)

type IUserService interface {
	SignIn(ctx context.Context, req *cmd.SignInRequest) (resp *cmd.SignInResponse, err error)
}

type UserService struct {
	UserMapper *user.MongoMapper
}

var UserServiceSet = wire.NewSet(
	wire.Struct(new(UserService), "*"),
	wire.Bind(new(IUserService), new(*UserService)),
)

func (u *UserService) SignIn(ctx context.Context, req *cmd.SignInRequest) (Resp *cmd.SignInResponse, err error) {
	// 1. 查找或创建用户
	oldUser, err := u.UserMapper.FindByWXOpenId(ctx, req.OpenID)
	if err != nil || oldUser == nil {
		newUser := user.User{
			ID:        primitive.NewObjectID(),
			OpenId:    req.OpenID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := u.UserMapper.Insert(ctx, &newUser); err != nil {
			return nil, errorx.ErrUserInsertFailed
		}
		oldUser = &newUser
	}

	// 2. 从Header中尝试获取现有Token
	existingToken, _ := token.ExtractToken(ctx.Value("httpRequest").(*http.Request).Header) // 忽略错误，可能没有Token

	// 3. 智能Token签发逻辑
	var tokenStr string
	if existingToken != "" {
		if claims, err := token.ParseAndValidate(existingToken); err == nil {
			// 验证用户匹配且不需要续期
			if claims.UserID == oldUser.ID && !token.ShouldRenew(claims) {
				return &cmd.SignInResponse{
					AccessToken: existingToken,
					ExpiresIn:   int64(time.Until(claims.ExpiresAt.Time).Seconds()),
				}, nil
			}
		}
	}

	// 4. 签发新Token
	tokenStr, err = token.NewAuthorizedToken(oldUser)
	if err != nil {
		return nil, errorx.ErrTokenCreationFailed
	}

	// 5. 返回响应
	return &cmd.SignInResponse{
		AccessToken: tokenStr,
		ExpiresIn:   config.GetConfig().Auth.AccessExpire,
	}, nil
}
