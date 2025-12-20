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

package service

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/google/wire"
)

var _ IProposalService = (*ProposalService)(nil)

type IProposalService interface {
	ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error)
}

type ProposalService struct {
	ProposalRepo      repo.IProposalRepo
	ProposalAssembler *assembler.ProposalAssembler
}

var ProposalServiceSet = wire.NewSet(
	wire.Struct(new(ProposalService), "*"),
	wire.Bind(new(IProposalService), new(*ProposalService)),
)

// ListProposals 分页查询所有提案，用于投票列表或管理端审核
func (s *ProposalService) ListProposals(ctx context.Context, req *dto.ListProposalReq) (*dto.ListProposalResp, error) {
	// 鉴权
	userId, ok := ctx.Value(consts.CtxUserID).(string)
	if !ok || userId == "" {
		return nil, errorx.New(errno.ErrUserNotLogin)
	}

	// 查询提案列表
	proposals, total, err := s.ProposalRepo.FindMany(ctx, req.PageParam)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalRepo] [FindMany] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalFindFailed)
	}

	// 转换为VO
	vos, err := s.ProposalAssembler.ToProposalVOArray(ctx, proposals)
	if err != nil {
		logs.CtxErrorf(ctx, "[ProposalAssembler] [ToProposalVOArray] error: %v", err)
		return nil, errorx.WrapByCode(err, errno.ErrProposalCvtFailed,
			errorx.KV("src", "database proposals"), errorx.KV("dst", "proposal vos"))
	}

	return &dto.ListProposalResp{
		Resp:      dto.Success(),
		Total:     total,
		Proposals: vos,
	}, nil
}
