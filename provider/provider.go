//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
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
	Config         *config.Config
	CommentService service.CommentService
}

var RpcSet = wire.NewSet(
	// TODO: 在这里添加 RPC 客户端的 Set
	// 例如: platform_sts.PlatformStsSet,
)

var ApplicationSet = wire.NewSet(
	service.CommentServiceSet,
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	comment.NewMongoMapper,
	wire.Bind(new(comment.IMongoMapper), new(*comment.MongoMapper)),

	RpcSet,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
