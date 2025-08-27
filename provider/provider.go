//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/searchhistory"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/user"
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
	AuthService          service.AuthService
	LikeService          service.LikeService
	CourseService        service.CourseService
	TeacherService       service.TeacherService
}

var ApplicationSet = wire.NewSet(
	service.CommentServiceSet,
	service.SearchHistoryServiceSet,
	service.AuthServiceSet,
	service.LikeServiceSet,
	service.CourseServiceSet,
	service.TeacherServiceSet,
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	consts.NewStaticData,
	comment.NewMongoMapper,
	searchhistory.NewMongoMapper,
	user.NewMongoMapper,
	like.NewMongoMapper,
	course.NewMongoMapper,
	teacher.NewMongoMapper,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)
