//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/searchhistory"
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
	Config               *config.Config
	CommentService       service.CommentService
	SearchHistoryService service.SearchHistoryService
}

var ApplicationSet = wire.NewSet(
	service.CommentServiceSet,
	service.SearchHistoryServiceSet,
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	comment.NewMongoMapper,
	searchhistory.NewMongoMapper,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
