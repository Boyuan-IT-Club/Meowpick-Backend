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
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ IProposalAssembler = (*ProposalAssembler)(nil)

type IProposalAssembler interface {
	ToProposalVO(ctx context.Context, db *model.Proposal) (*dto.ProposalVO, error)
	ToProposalVOArray(ctx context.Context, dbs []*model.Proposal) ([]*dto.ProposalVO, error)
}

type ProposalAssembler struct {
	CourseAssembler *CourseAssembler
}

var ProposalAssemblerSet = wire.NewSet(
	wire.Struct(new(ProposalAssembler), "*"),
	wire.Bind(new(IProposalAssembler), new(*ProposalAssembler)),
)

// ToProposalVO 单个ProposalDB转ProposalVO (DB to VO)
func (a *ProposalAssembler) ToProposalVO(ctx context.Context, db *model.Proposal) (*dto.ProposalVO, error) {
	var courseVO *dto.CourseVO
	if db.Course != nil {
		var err error
		courseVO, err = a.CourseAssembler.ToCourseVO(ctx, db.Course)
		if err != nil {
			logs.CtxErrorf(ctx, "[CourseAssembler] [ToCourseVO] error: %v", err)
			return nil, err
		}
	}

	return &dto.ProposalVO{
		ID:        db.ID,
		UserID:    db.UserID,
		Title:     db.Title,
		Content:   db.Content,
		Course:    courseVO,
		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}, nil
}

// ToProposalVOArray ProposalDB数组转ProposalVO数组 (DB Array to VO Array)
func (a *ProposalAssembler) ToProposalVOArray(ctx context.Context, dbs []*model.Proposal) ([]*dto.ProposalVO, error) {
	if len(dbs) == 0 {
		logs.CtxWarnf(ctx, "[ProposalAssembler] [ToProposalVOArray] empty proposal db array")
		return []*dto.ProposalVO{}, nil
	}

	vos := make([]*dto.ProposalVO, 0, len(dbs))
	for _, db := range dbs {
		vo, err := a.ToProposalVO(ctx, db)
		if err != nil {
			logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVO] error: %v", err)
			return nil, err
		}
		if vo != nil {
			vos = append(vos, vo)
		}
	}

	return vos, nil
}

