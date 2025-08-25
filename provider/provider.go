//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/google/wire"
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
	wire.Bind(new(service.ICourseService), new(*service.CourseService)),
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	// TODO: 在这里添加 Mapper 的构造函数
	// 例如: user.NewMongoMapper,
	course.NewCourseMapper,
	wire.Bind(new(course.IMongoMapper), new(*course.MongoMapper)),
	RpcSet,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
