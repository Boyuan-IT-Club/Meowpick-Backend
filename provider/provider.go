package provider

import (
	"github.com/google/wire"
	// TODO: 当你创建第一个 service 时，取消这里的注释
	// "Meowpick-Backend/application/service"
	"Meowpick-Backend/infra/config"
	// TODO: 当你创建第一个 mapper 时，取消这里的注释
	// "Meowpick-Backend/infra/mapper/user"
)

var provider *Provider

// Init 初始化全局的 Provider 实例，通常在 main 函数开始时调用
func Init() {
	var err error
	provider, err = NewProvider()
	if err != nil {
		panic(err)
	}
}

// Get 获取全局 Provider 实例
func Get() *Provider {
	return provider
}

// Provider 聚合了所有依赖项，供 wire 进行注入。
// 未来添加新的 Service 接口时，请在这里声明。
type Provider struct {
	Config *config.Config
	// TODO: 在这里添加需要注入的 Service 接口
	// 例如: UserService service.IUserService
}

// RpcSet 用于聚合所有 RPC 客户端的依赖定义
var RpcSet = wire.NewSet(
// TODO: 在这里添加 RPC 客户端的 Set
// 例如: platform_sts.PlatformStsSet,
)

// ApplicationSet 用于聚合所有 Service 的依赖定义
var ApplicationSet = wire.NewSet(
// TODO: 在这里添加 Service 的 Set
// 例如: service.UserServiceSet,
)

// InfrastructureSet 用于聚合所有基础设施层（如 config, mappers, rpc）的依赖定义
var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	// TODO: 在这里添加 Mapper 的构造函数
	// 例如: user.NewMongoMapper,

	RpcSet,
)

// AllProvider 包含了项目所有的依赖集合，供 wire 使用
var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
