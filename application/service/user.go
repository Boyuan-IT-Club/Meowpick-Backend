package service

import (
	"context"
	//"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
	"github.com/google/wire"
)

type IUserService interface {
	ChangeUserName(ctx context.Context, newName string) error

	// 邮箱认证相关
	SendEmailVerifyCode(ctx context.Context, userId string, target string) error
	VerifyEmail(ctx context.Context, userID string, code string) error
}

type UserService struct {
	UserMapper *user.MongoMapper
}

var UserServiceSet = wire.NewSet(
	wire.Struct(new(UserService), "*"),
	wire.Bind(new(IUserService), new(*AuthService)),
)

// TODO 在controller层 接收gin.context参数获取到userid
func (u *UserService) SendEmailVerifyCode(ctx context.Context, userId string, target string) error {
	return nil
}
