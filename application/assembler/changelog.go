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

package assembler

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/google/wire"
)

var _ IChangeLogAssembler = (*ChangeLogAssembler)(nil)

type IChangeLogAssembler interface {
	ToChangeLogDB(ctx context.Context, vo *dto.ChangeLogVO) (*model.ChangeLog, error)
}

type ChangeLogAssembler struct{}

var ChangeLogAssemblerSet = wire.NewSet(
	wire.Struct(new(ChangeLogAssembler), "*"),
	wire.Bind(new(IChangeLogAssembler), new(*ChangeLogAssembler)),
)

// ToChangeLogDB 单个ChangeLogVO转ChangeLogDB (VO to DB)
func (a *ChangeLogAssembler) ToChangeLogDB(ctx context.Context, vo *dto.ChangeLogVO) (*model.ChangeLog, error) {
	return &model.ChangeLog{
		ID:           vo.ID,
		TargetID:     vo.TargetID,
		TargetType:   vo.TargetType,
		Action:       vo.Action,
		Content:      vo.Content,
		UpdateSource: vo.UpdateSource,
		ProposalID:   vo.ProposalID,
		UserID:       vo.UserID,
		UpdatedAt:    vo.UpdatedAt,
	}, nil
}
