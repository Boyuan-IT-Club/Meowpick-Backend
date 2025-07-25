package provider

import (
	"github.com/google/wire"
	"github.com/xh-polaris/essay-show/biz/application/service"
	"github.com/xh-polaris/essay-show/biz/infrastructure/config"
	"github.com/xh-polaris/essay-show/biz/infrastructure/mapper/attend"
	"github.com/xh-polaris/essay-show/biz/infrastructure/mapper/exercise"
	"github.com/xh-polaris/essay-show/biz/infrastructure/mapper/feedback"
	"github.com/xh-polaris/essay-show/biz/infrastructure/mapper/invitation"
	"github.com/xh-polaris/essay-show/biz/infrastructure/mapper/log"
	"github.com/xh-polaris/essay-show/biz/infrastructure/mapper/user"
	"github.com/xh-polaris/essay-show/biz/infrastructure/rpc/platform_sts"
)

var provider *Provider

func Init() {
	var err error
	provider, err = NewProvider()
	if err != nil {
		panic(err)
	}
}

// Provider 提供controller依赖的对象
type Provider struct {
	Config          *config.Config
	UserService     service.UserService
	EssayService    service.EssayService
	StsService      service.StsService
	ExerciseService service.ExerciseService
	FeedBackService service.FeedBackService
}

func Get() *Provider {
	return provider
}

var RpcSet = wire.NewSet(
	platform_sts.PlatformStsSet,
)

var ApplicationSet = wire.NewSet(
	service.UserServiceSet,
	service.EssayServiceSet,
	service.StsServiceSet,
	service.ExerciseServiceSet,
	service.FeedbackServiceSet,
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	user.NewMongoMapper,
	log.NewMongoMapper,
	exercise.NewMongoMapper,
	attend.NewMongoMapper,
	invitation.NewCodeMongoMapper,
	invitation.NewLogMongoMapper,
	feedback.NewMongoMapper,
	RpcSet,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
