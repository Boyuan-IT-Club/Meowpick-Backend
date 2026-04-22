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
	"errors"
	"fmt"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName = "proposal"
)

type IProposalRepo interface {
	Insert(ctx context.Context, proposal *model.Proposal) error
	IsCourseInExistingProposals(ctx context.Context, course *model.ProposalCourse) (bool, error)
	FindMany(ctx context.Context, param *dto.PageParam) ([]*model.Proposal, int64, error)
	FindManyByStatus(ctx context.Context, param *dto.PageParam, status int32) ([]*model.Proposal, int64, error)
	FindManyByFilter(ctx context.Context, req *dto.FilterProposalReq, statuses []int32) ([]*model.Proposal, int64, error)
	FindByID(ctx context.Context, proposalID string) (*model.Proposal, error)
	FindByIDs(ctx context.Context, proposalIDs []string) ([]*model.Proposal, error)
	UpdateProposal(ctx context.Context, proposal *model.Proposal) error
	DeleteProposal(ctx context.Context, proposalId string, operatorId string) error
	GetSuggestionsByTitle(ctx context.Context, title string, param *dto.PageParam) ([]*model.Proposal, int64, error)
	UpdateStatusByID(ctx context.Context, proposalID string, statusID int32) (bool, error)
}

type ProposalRepo struct {
	conn *monc.Model
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
// 比较的字段包括: Name, Code, Department, Category, Campuses, Teachers
func (r *ProposalRepo) IsCourseInExistingProposals(ctx context.Context, course *model.ProposalCourse) (bool, error) {
	filter := bson.M{
		consts.PathCourseName:       course.Name,
		consts.PathCourseCode:       course.Code,
		consts.PathCourseDepartment: course.Department,
		consts.PathCourseCategory:   course.Category,
		consts.PathCourseCampuses:   course.Campuses,
		consts.PathCourseTeachers:   course.Teachers,
		consts.Deleted:              false,
	}

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

	if param == nil {
		return proposals, total, nil
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

// FindManyByStatus 分页查询指定状态的提案
func (r *ProposalRepo) FindManyByStatus(ctx context.Context, param *dto.PageParam, status int32) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{
		consts.Status:  status,
		consts.Deleted: bson.M{"$ne": true},
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 如果不需要具体内容（param为nil），则直接返回总数
	if param == nil {
		return proposals, total, nil
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

// FindManyByFilter 按多个字段筛选提案
func (r *ProposalRepo) FindManyByFilter(ctx context.Context, req *dto.FilterProposalReq, statuses []int32) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{
		consts.Deleted:            bson.M{"$ne": true},
		consts.Status:             bson.M{"$in": statuses},
		consts.PathCourseCampuses: bson.M{"$in": req.Campuses},
	}

	if req.Department != "" {
		filter[consts.PathCourseDepartment] = req.Department
	}
	if req.Category != "" {
		filter[consts.PathCourseCategory] = req.Category
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if req.PageParam == nil {
		return proposals, total, nil
	}

	if err = r.conn.Find(
		ctx,
		&proposals,
		filter,
		page.FindPageOption(req.PageParam).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
		return nil, 0, err
	}

	return proposals, total, nil
}

// FindByID 根据提案ID查询单个未删除的提案
func (r *ProposalRepo) FindByID(ctx context.Context, proposalID string) (*model.Proposal, error) {
	proposal := model.Proposal{}
	if err := r.conn.FindOneNoCache(ctx, &proposal,
		bson.M{consts.ID: proposalID, consts.Deleted: bson.M{"$ne": true}}, nil); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &proposal, nil
}

// DeleteProposal 删除单个提案
func (r *ProposalRepo) DeleteProposal(ctx context.Context, proposalId string, operatorId string) error {
	// 查找未删除的提案
	filter := bson.M{
		consts.ID:      proposalId,
		consts.Deleted: bson.M{"$ne": true},
	}

	// 更新删除状态和删除时间
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			consts.Deleted:   true,
			consts.DeletedAt: now,
			consts.UpdatedAt: now,
		},
	}

	// 执行软删除操作
	key := fmt.Sprintf("proposal:%s", proposalId)
	_, err := r.conn.UpdateOne(ctx, key, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// UpdateProposal 更新提案
func (r *ProposalRepo) UpdateProposal(ctx context.Context, proposal *model.Proposal) error {

	filter := bson.M{
		consts.ID:      proposal.ID,
		consts.Deleted: bson.M{"$ne": true},
	}

	update := bson.M{
		"$set": bson.M{
			"title":          proposal.Title,
			"content":        proposal.Content,
			"course":         proposal.Course,
			consts.UpdatedAt: proposal.UpdatedAt,
		},
	}

	_, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	return err
}

// GetSuggestionsByTitle 根据提案标题模糊分页查询提案
func (r *ProposalRepo) GetSuggestionsByTitle(ctx context.Context, title string, param *dto.PageParam) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{
		"title":        bson.M{"$regex": primitive.Regex{Pattern: title, Options: "i"}},
		consts.Deleted: bson.M{"$ne": true},
	}
	sort := bson.D{
		{consts.Status, 1},
		{consts.CreatedAt, -1},
	}

	if err := r.conn.Find(ctx, &proposals, filter, page.FindPageOption(param).SetSort(sort)); err != nil {
		return nil, 0, err
	}

	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return proposals, total, nil
}

// FindByIDs 根据提案ID列表批量查询提案
func (r *ProposalRepo) FindByIDs(ctx context.Context, proposalIDs []string) ([]*model.Proposal, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{
		consts.ID: bson.M{"$in": proposalIDs},
	}

	if err := r.conn.Find(ctx, &proposals, filter); err != nil {
		return nil, err
	}

	return proposals, nil
}

// UpdateStatusByID 根据提案ID更新提案状态
func (r *ProposalRepo) UpdateStatusByID(ctx context.Context, proposalID string, statusID int32) (bool, error) {
	filter := bson.M{consts.ID: proposalID, consts.Deleted: bson.M{"$ne": true}}
	update := bson.M{"$set": bson.M{consts.Status: statusID, consts.UpdatedAt: time.Now()}}

	result, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	if err != nil {
		return false, err
	}

	// 检查是否更新了文档
	updated := result.ModifiedCount > 0
	return updated, nil
}

// FindManyByUserID 根据用户ID批量获取提案
func (r *ProposalRepo) FindManyByUserID(ctx context.Context, param *dto.PageParam, userId string) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{consts.UserID: userId, consts.Deleted: bson.M{"$ne": true}}
	total, err := r.conn.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if err = r.conn.Find(ctx, &proposals, filter,
		page.FindPageOption(param).SetSort(page.DSort(consts.CreatedAt, -1)),
	); err != nil {
		return nil, 0, err
	}
	return proposals, total, nil
}
