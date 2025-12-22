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

package repo

import (
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName = "proposal"
)

type IProposalRepo interface {
}

type ProposalRepo struct {
	conn *monc.Model
}

func NewProposalRepo(cfg *config.Config) *ProposalRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ProposalCollectionName, cfg.Cache)
	return &ProposalRepo{conn: conn}
}
