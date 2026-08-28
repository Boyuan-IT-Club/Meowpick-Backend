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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName         = "proposal"
	ProposalGuardCollectionName    = "proposal_guard"
	ProposalUserCreatedAtIndexName = "idx_proposal_user_id_created_at"
	ProposalGuardTTLIndexName      = "idx_proposal_guard_expires_at"
	proposalGuardTTL               = 48 * time.Hour
)

type IProposalRepo interface {
	WithTransaction(ctx context.Context, fn func(mongo.SessionContext) error) error
	AcquireCreateGuards(ctx context.Context, userDayKey, courseFingerprint string) error
	Insert(ctx context.Context, proposal *model.Proposal) error
	IsCourseInExistingProposals(ctx context.Context, course *model.ProposalCourse) (bool, error)
	FindMany(ctx context.Context, param *dto.PageParam) ([]*model.Proposal, int64, error)
	FindManyByStatus(ctx context.Context, param *dto.PageParam, status int32) ([]*model.Proposal, int64, error)
	FindManyByFilter(ctx context.Context, req *dto.FilterProposalReq, statuses []int32) ([]*model.Proposal, int64, error)
	FindByID(ctx context.Context, proposalID string) (*model.Proposal, error)
	FindByIDIncludeDeleted(ctx context.Context, proposalID string) (*model.Proposal, error)
	FindByIDs(ctx context.Context, proposalIDs []string) ([]*model.Proposal, error)
	CountByUserToday(ctx context.Context, userId string) (int64, error)
	UpdateProposal(ctx context.Context, proposal *model.Proposal, expectedStatus int32) (bool, error)
	DeleteProposal(ctx context.Context, proposalId, operatorId string, allowedStatuses []int32) (bool, error)
	RestoreProposal(ctx context.Context, proposalId string) error
	GetSuggestionsByTitle(ctx context.Context, title string, param *dto.PageParam, statusID int32) ([]*model.Proposal, int64, error)
	UpdateStatusByID(ctx context.Context, proposalID string, expectedStatusID, statusID int32) (bool, error)
	IncrementLikeCnt(ctx context.Context, proposalID string, delta int64) error
	UpdateStatusAndReasonByID(ctx context.Context, proposalID string, expectedStatusID, statusID int32, rejectReason string) (bool, error)
	UpdateContributionByID(ctx context.Context, proposalID string, contribution int64) error
}

type ProposalRepo struct {
	conn *monc.Model
}

func NewProposalRepo(cfg *config.Config) (*ProposalRepo, error) {
	conn := monc.MustNewModel(cfg.Mongo.URL, cfg.Mongo.DB, ProposalCollectionName, cfg.Cache)
	repository := &ProposalRepo{conn: conn}
	if err := repository.ensureTransactionSupport(context.Background()); err != nil {
		return nil, err
	}
	if err := repository.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repository, nil
}

func (r *ProposalRepo) ensureTransactionSupport(ctx context.Context) error {
	var hello bson.M
	if err := r.conn.Database().RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		return fmt.Errorf("MongoDB transaction preflight: %w", err)
	}
	if msg, _ := hello["msg"].(string); msg == "isdbgrid" {
		return nil
	}
	if setName, _ := hello["setName"].(string); setName != "" {
		return nil
	}
	return errors.New("MongoDB transactions require a replica set or sharded deployment")
}

func (r *ProposalRepo) WithTransaction(ctx context.Context, fn func(mongo.SessionContext) error) error {
	session, err := r.conn.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sessionContext mongo.SessionContext) (any, error) {
		return nil, fn(sessionContext)
	})
	return err
}

// AcquireCreateGuards serializes quota and duplicate checks inside a transaction.
// The guard documents contain no business data and are never used as a source of truth.
func (r *ProposalRepo) AcquireCreateGuards(ctx context.Context, userDayKey, courseFingerprint string) error {
	collection := r.conn.Database().Collection(ProposalGuardCollectionName)
	_, inTransaction := ctx.(mongo.SessionContext)
	for _, id := range []string{"quota:" + userDayKey, "course:" + courseFingerprint} {
		if !inTransaction {
			_, err := collection.UpdateOne(ctx, bson.M{consts.ID: id}, bson.M{
				"$setOnInsert": bson.M{"version": 0, consts.CreatedAt: time.Now()},
				"$set":         bson.M{"expiresAt": time.Now().Add(proposalGuardTTL)},
			}, options.Update().SetUpsert(true))
			if err != nil && !mongo.IsDuplicateKeyError(err) {
				return err
			}
			continue
		}
		_, err := collection.UpdateOne(ctx, bson.M{consts.ID: id}, bson.M{
			"$inc": bson.M{"version": 1},
			"$set": bson.M{consts.UpdatedAt: time.Now(), "expiresAt": time.Now().Add(proposalGuardTTL)},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ensureIndexes 创建 proposal 集合所需的索引。CreateOne 对同名同定义索引是幂等的，
// 因此可以在每次服务启动时安全调用。
func (r *ProposalRepo) ensureIndexes(ctx context.Context) error {
	_, err := r.conn.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: consts.UserID, Value: 1},
			{Key: consts.CreatedAt, Value: -1},
		},
		Options: options.Index().SetName(ProposalUserCreatedAtIndexName),
	})
	if err != nil {
		return err
	}
	_, err = r.conn.Database().Collection(ProposalGuardCollectionName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetName(ProposalGuardTTLIndexName).SetExpireAfterSeconds(0),
	})
	return err
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
		consts.Deleted: bson.M{"$ne": true},
	}

	if len(statuses) > 0 {
		filter[consts.Status] = bson.M{"$in": statuses}
	}
	if len(req.Campuses) > 0 {
		filter[consts.PathCourseCampuses] = bson.M{"$in": req.Campuses}
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

// FindByIDIncludeDeleted 根据提案ID查询单个提案（包含已删除的）
func (r *ProposalRepo) FindByIDIncludeDeleted(ctx context.Context, proposalID string) (*model.Proposal, error) {
	proposal := model.Proposal{}
	if err := r.conn.FindOneNoCache(ctx, &proposal,
		bson.M{consts.ID: proposalID}, nil); err != nil {
		if errors.Is(err, monc.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &proposal, nil
}

// DeleteProposal 删除单个提案
func (r *ProposalRepo) DeleteProposal(ctx context.Context, proposalId, operatorId string, allowedStatuses []int32) (bool, error) {
	// 查找未删除的提案
	filter := bson.M{
		consts.ID:      proposalId,
		consts.UserID:  operatorId,
		consts.Status:  bson.M{"$in": allowedStatuses},
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
	result, err := r.conn.UpdateOne(ctx, key, filter, update)
	if err != nil {
		return false, err
	}
	return result.ModifiedCount == 1, nil
}

// UpdateProposal 更新提案
func (r *ProposalRepo) UpdateProposal(ctx context.Context, proposal *model.Proposal, expectedStatus int32) (bool, error) {

	filter := bson.M{
		consts.ID:      proposal.ID,
		consts.Status:  expectedStatus,
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

	result, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	if err != nil {
		return false, err
	}
	return result.ModifiedCount == 1, nil
}

// GetSuggestionsByTitle 根据提案标题模糊分页查询指定状态的提案
func (r *ProposalRepo) GetSuggestionsByTitle(ctx context.Context, title string, param *dto.PageParam, statusID int32) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{
		"title":        bson.M{"$regex": primitive.Regex{Pattern: title, Options: "i"}},
		consts.Status:  statusID,
		consts.Deleted: bson.M{"$ne": true},
	}
	sort := bson.D{
		{Key: consts.CreatedAt, Value: -1},
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
func (r *ProposalRepo) UpdateStatusByID(ctx context.Context, proposalID string, expectedStatusID, statusID int32) (bool, error) {
	filter := bson.M{consts.ID: proposalID, consts.Status: expectedStatusID, consts.Deleted: bson.M{"$ne": true}}
	update := bson.M{"$set": bson.M{consts.Status: statusID, consts.UpdatedAt: time.Now()}}

	result, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	if err != nil {
		return false, err
	}

	// 检查是否更新了文档
	updated := result.ModifiedCount > 0
	return updated, nil
}

// UpdateStatusAndReasonByID 根据提案ID更新提案状态和拒绝理由
func (r *ProposalRepo) UpdateStatusAndReasonByID(ctx context.Context, proposalID string, expectedStatusID, statusID int32, rejectReason string) (bool, error) {
	filter := bson.M{consts.ID: proposalID, consts.Status: expectedStatusID, consts.Deleted: bson.M{"$ne": true}}
	update := bson.M{"$set": bson.M{consts.Status: statusID, "rejectReason": rejectReason, consts.UpdatedAt: time.Now()}}

	result, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	if err != nil {
		return false, err
	}

	updated := result.ModifiedCount > 0
	return updated, nil
}

// FindManyByUserID 根据用户ID批量获取提案（包含所有状态和已删除的提案）
func (r *ProposalRepo) FindManyByUserID(ctx context.Context, param *dto.PageParam, userId string) ([]*model.Proposal, int64, error) {
	proposals := []*model.Proposal{}
	filter := bson.M{consts.UserID: userId}
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

// CountByUserToday 按中国时区（UTC+8）自然日统计用户今日创建的提案数
func (r *ProposalRepo) CountByUserToday(ctx context.Context, userId string) (int64, error) {
	loc := time.FixedZone("CST", 8*3600) // 中国时区 UTC+8
	now := time.Now().In(loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	filter := bson.M{
		consts.UserID: userId,
		consts.CreatedAt: bson.M{
			"$gte": dayStart,
			"$lt":  dayStart.Add(24 * time.Hour),
		},
	}
	return r.conn.CountDocuments(ctx, filter)
}

// RestoreProposal 恢复已删除的提案（将deleted设为false，清空deletedAt）
func (r *ProposalRepo) RestoreProposal(ctx context.Context, proposalId string) error {
	filter := bson.M{
		consts.ID:      proposalId,
		consts.Deleted: true,
	}
	update := bson.M{
		"$set": bson.M{
			consts.Deleted:   false,
			consts.UpdatedAt: time.Now(),
		},
		"$unset": bson.M{
			consts.DeletedAt: "",
		},
	}

	key := fmt.Sprintf("proposal:%s", proposalId)
	_, err := r.conn.UpdateOne(ctx, key, filter, update)
	return err
}

func (r *ProposalRepo) IncrementLikeCnt(ctx context.Context, proposalID string, delta int64) error {
	filter := bson.M{consts.ID: proposalID, consts.Deleted: bson.M{"$ne": true}}
	update := bson.M{"$inc": bson.M{"likeCnt": delta}, "$set": bson.M{consts.UpdatedAt: time.Now()}}
	_, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	return err
}

// UpdateContributionByID 更新提案记录的贡献值（撤回审批通过时置0）
func (r *ProposalRepo) UpdateContributionByID(ctx context.Context, proposalID string, contribution int64) error {
	filter := bson.M{consts.ID: proposalID, consts.Deleted: bson.M{"$ne": true}}
	update := bson.M{"$set": bson.M{consts.Contribution: contribution, consts.UpdatedAt: time.Now()}}
	_, err := r.conn.UpdateOneNoCache(ctx, filter, update)
	return err
}
