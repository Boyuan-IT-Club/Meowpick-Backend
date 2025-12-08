//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/searchhistory"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/user"
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
	SearchService        service.SearchService
}

var ApplicationSet = wire.NewSet(
	service.CommentServiceSet,
	service.SearchHistoryServiceSet,
	service.AuthServiceSet,
	service.LikeServiceSet,
	service.CourseServiceSet,
	service.TeacherServiceSet,
	service.SearchServiceSet,
	// Assembler 相关
	assembler.CommentDTOSet,
	assembler.CourseDTOSet,
	assembler.TeacherDTOSet,
)

var InfraSet = wire.NewSet(
	config.NewConfig,
	comment.NewMongoRepo,
	searchhistory.NewMongoRepo,
	user.NewMongoRepo,
	like.NewMongoRepo,
	course.NewMongoRepo,
	teacher.NewMongoRepo,
	// 缓存相关
	cache.NewLikeCache,
	// 映射相关
	mapping.NewStaticData,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfraSet,
)
