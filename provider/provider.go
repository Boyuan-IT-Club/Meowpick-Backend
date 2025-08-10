//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/google/wire"
	// TODO: 当你创建第一个 service 时，取消这里的注释
	// "github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
)

var provider *Provider

func Init() {
	var err error
	provider, err = NewProvider()
	if err != nil {
		panic(err)
	}
}

func Get() *Provider {
	return provider
}

// Provider 提供controller依赖的对象
type Provider struct {
	Config *config.Config
	// TODO: 在这里添加需要注入的 Service 接口
	// 例如: UserService service.IUserService
	CourseService service.ICourseService
}

var RpcSet = wire.NewSet(
// TODO: 在这里添加 RPC 客户端的 Set
// 例如: platform_sts.PlatformStsSet,
)

var ApplicationSet = wire.NewSet(
	// TODO: 在这里添加 Service 的 Set
	// 例如: service.UserServiceSet,
	service.NewCourseService,
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	// TODO: 在这里添加 Mapper 的构造函数
	// 例如: user.NewMongoMapper,
	course.NewCourseMapper,
	RpcSet,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
