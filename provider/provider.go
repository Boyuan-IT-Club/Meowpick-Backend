// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate wire .

package provider

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/service"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
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
	assembler.CommentAssemblerSet,
	assembler.CourseAssemblerSet,
	assembler.TeacherAssemblerSet,
)

var InfraSet = wire.NewSet(
	config.NewConfig,
	repo.NewLikeRepo,
	repo.NewUserRepo,
	repo.NewCourseRepo,
	repo.NewTeacherRepo,
	repo.NewCommentRepo,
	repo.NewSearchHistoryRepo,
	// 缓存相关
	cache.NewLikeCache,
	cache.NewCommentCache,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfraSet,
)
