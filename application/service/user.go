package service

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd/resp"
	authtoken "github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/security"

	errorx "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/exception"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/google/wire"
)

type IUserService interface {
	Login(ctx context.Context, req cmd.LoginCMD) (resp *resp.LoginResp, err error)
}

type UserService struct {
	UserMapper *user.MongoMapper
}

var UserServiceSet = wire.NewSet(
	wire.Struct(new(UserService), "*"),
	wire.Bind(new(IUserService), new(*UserService)),
)

func (u *UserService) Login(ctx context.Context, req cmd.LoginCMD) (Resp *resp.LoginResp, err error) {
	// 根据openid查找用户
	oldUser, err := u.UserMapper.FindByWXOpenId(ctx, req.OpenID)
	// 如果没能查到，创建用户
	if err != nil || oldUser == nil {
		newUser := user.User{
			ID:        primitive.NewObjectID(),
			OpenId:    req.OpenID,
			Username:  req.Name,
			Avatar:    req.Avatar,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := u.UserMapper.Insert(ctx, &newUser); err != nil {
			return nil, errorx.ErrUserInsertFailed
		}
		oldUser = &newUser
	}
	// 创建AccessToken
	tokenStr, err := authtoken.NewAuthorizedToken(oldUser, false, "", "")

	// 返回响应resp
	return &resp.LoginResp{
		AccessToken: tokenStr,
	}, nil
}
