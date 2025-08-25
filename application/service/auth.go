package service

import (
	"context"
	"errors"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/google/wire"
)

type IAuthService interface {
	SignIn(ctx context.Context, req *cmd.SignInRequest) (resp *cmd.SignInResponse, err error)
}

type AuthService struct {
	UserMapper *user.MongoMapper
}

var AuthServiceSet = wire.NewSet(
	wire.Struct(new(AuthService), "*"),
	wire.Bind(new(IAuthService), new(*AuthService)),
)

func (a *AuthService) SignIn(ctx context.Context, req *cmd.SignInRequest) (Resp *cmd.SignInResponse, err error) {
	// 1. 查找或创建用户
	var openID string
	//if openID = util.GetWeChatOpenID(config.GetConfig().WeApp.AppID, config.GetConfig().WeApp.AppSecret, req.Code); openID == "" {
	//	log.Error("openID为空")
	//	return nil, errorx.ErrEmptyOpenID
	//}

	// TODO: 模拟登录逻辑（开发环境用）
	if req.Code == "test123" {
		openID = "debug-openid-001" // 你随便写一个唯一标识
	} else {
		openID = util.GetWeChatOpenID(config.GetConfig().WeApp.AppID,
			config.GetConfig().WeApp.AppSecret,
			req.Code)
	}

	if openID == "" {
		log.Error("openID为空")
		return nil, errorx.ErrEmptyOpenID
	}

	oldUser, err := a.UserMapper.FindByWXOpenId(ctx, openID)
	if err != nil {
		if errors.Is(err, errorx.ErrUserNotFound) {
			// 创建用户并存入数据库
			newUser := user.User{
				ID:        primitive.NewObjectID(),
				OpenId:    openID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err = a.UserMapper.Insert(ctx, &newUser); err != nil {
				return nil, errorx.ErrInsertFailed
			}

			oldUser = &newUser
		} else {
			return nil, err
		}
	}

	// 3. 智能Token签发逻辑
	var tokenStr string
	existingToken, ok := ctx.Value(consts.ContextUserID).(string)

	if existingToken != "" || !ok {
		if claims, err := token.ParseAndValidate(existingToken); err == nil {
			// 验证用户匹配且不需要续期
			if claims.UserID == oldUser.ID.Hex() && !token.ShouldRenew(claims) {
				return &cmd.SignInResponse{
					AccessToken: existingToken,
					ExpiresIn:   int64(time.Until(claims.ExpiresAt.Time).Seconds()),
					UserID:      oldUser.ID.Hex(),
					Resp:        cmd.Success(),
				}, nil
			}
		}
	}

	// 4. 签发新Token
	if tokenStr, err = token.NewAuthorizedToken(oldUser); err != nil {
		return nil, errorx.ErrTokenCreationFailed
	}

	// 5. 返回响应
	return &cmd.SignInResponse{
		Resp:        cmd.Success(),
		AccessToken: tokenStr,
		ExpiresIn:   config.GetConfig().Auth.AccessExpire,
		UserID:      oldUser.ID.Hex(),
	}, nil
}
