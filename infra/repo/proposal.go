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
	"context"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName = "proposal"
)

type IProposalRepo interface {
	Insert(ctx context.Context, proposal *model.Proposal) error
	IsCourseInExistingProposals(ctx context.Context, courseVO *model.Course) (bool, error)
	FindMany(ctx context.Context, param *dto.PageParam) ([]*model.Proposal, int64, error)
	Toggle(ctx context.Context, userId, targetId string, targetType int32) (bool, error)
	IsProposal(ctx context.Context, userId, targetId string, targetType int32) (bool, error)
	CountByTarget(ctx context.Context, targetId string, targetType int32) (int64, error)
	FindByID(ctx context.Context, proposalID string) (*model.Proposal, error) // 修改方法名
}

type ProposalRepo struct {
	conn  *monc.Model
	cache *cache.ProposalCache
}

func NewProposalRepo(cfg *config.Config) *ProposalRepo {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ProposalCollectionName, cfg.Cache)
	return &ProposalRepo{conn: conn}
}

// Insert 插入一个新的提案
func (r *ProposalRepo) Insert(ctx context.Context, proposal *model.Proposal) error {
	_, err := r.conn.InsertOneNoCache(ctx, proposal)
	return err
}

// IsCourseInExistingProposals 检查课程是否已经存在于现有提案中
// 比较的字段包括: Name, Code, Department, Category, Campuses, TeacherIDs
func (r *ProposalRepo) IsCourseInExistingProposals(ctx context.Context, courseDB *model.Course) (bool, error) {
	filter := bson.M{
		consts.Name:       courseDB.Name,
		consts.Code:       courseDB.Code,
		consts.Department: courseDB.Department,
		consts.Categories: courseDB.Category,
		consts.Campuses:   courseDB.Campuses,
		consts.TeacherIDs: courseDB.TeacherIDs,
		consts.Deleted:    false, // 只检查未删除的提案
	}

	// 查询提案中是否已存在该课程
	count, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindMany 分页查询所有未删除的提案
func (r *ProposalRepo) FindMany(ctx context.Context, param *dto.PageParam) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{consts.Deleted: bson.M{"$ne": true}}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if err = r.conn.Find(
		ctx,
		&proposals,
		filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
		return nil, 0, err
	}

	return proposals, total, nil
}

// Toggle 翻转投票状态
func (r *ProposalRepo) Toggle(ctx context.Context, userId, targetId string, targetType int32) (bool, error) {
	now := time.Now()
	pipeline := mongo.Pipeline{
		{{"$set", bson.M{
			consts.ID: bson.M{"$ifNull": bson.A{"$" + consts.ID, primitive.NewObjectID().Hex()}},

			consts.UserID:   bson.M{"$ifNull": bson.A{"$" + consts.UserID, userId}},
			consts.TargetID: bson.M{"$ifNull": bson.A{"$" + consts.TargetID, targetId}},

			consts.CreatedAt: bson.M{"$ifNull": bson.A{"$" + consts.CreatedAt, now}},
			consts.UpdatedAt: now,

			consts.Active: bson.M{"$cond": bson.A{
				bson.M{"$not": bson.M{"$ifNull": bson.A{"$" + consts.ID, nil}}},
				true,
				bson.M{"$not": "$active"},
			}},
		}}},
	}
	var proposal struct {
		Active bool `bson:"active"`
	}

	err := r.conn.FindOneAndUpdateNoCache(ctx,
		&proposal,
		bson.M{consts.UserID: userId, consts.TargetID: targetId},
		pipeline,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)
	return proposal.Active, err
}

// IsProposal 获取一个用户对一个目标的当前投票状态
func (r *ProposalRepo) IsProposal(ctx context.Context, userId, targetId string, targetType int32) (bool, error) {
	cnt, err := r.conn.CountDocuments(ctx, bson.M{
		consts.UserID:   userId,
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false},
	})
	return cnt > 0, err
}

// CountProposalByTarget 获得目标的总投票数
func (r *ProposalRepo) CountByTarget(ctx context.Context, targetId string, targetType int32) (int64, error) {
	return r.conn.CountDocuments(ctx, bson.M{
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false},
	})
}

// FindProposalByID 根据提案ID查询单个未删除的提案
func (r *ProposalRepo) FindByID(ctx context.Context, proposalID string) (*model.Proposal, error) {
	var proposal model.Proposal

	filter := bson.M{
		consts.ID:      proposalID,
		consts.Deleted: bson.M{"$ne": true},
	}

	err := r.conn.FindOneNoCache(ctx, &proposal, filter, nil)
	if err != nil {
		if err == monc.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &proposal, nil
}
