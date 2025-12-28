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

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

var _ IProposalRepo = (*ProposalRepo)(nil)

const (
	ProposalCollectionName = "proposal"
)

type IProposalRepo interface {
	Insert(ctx context.Context, proposal *model.Proposal) error
	IsCourseInExistingProposals(ctx context.Context, courseVO *dto.CourseVO) (bool, error)
	FindMany(ctx context.Context, param *dto.PageParam) ([]*model.Proposal, int64, error)
	Toggle(ctx context.Context, userId, targetId string, targetType int32) (bool, error)
	IsProposal(ctx context.Context, userId, targetId string, targetType int32) (bool, error)
	CountProposalByTarget(ctx context.Context, targetId string, targetType int32) (int64, error)
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
func (s *ProposalRepo) IsCourseInExistingProposals(ctx context.Context, courseVO *dto.CourseVO) (bool, error) {
	// 将DTO中的值转换为ID形式以便数据库查询
	departmentID := mapping.Data.GetDepartmentIDByName(courseVO.Department)
	categoryID := mapping.Data.GetCategoryIDByName(courseVO.Category)

	// 将校区名称转换为ID
	campusIDs := make([]int32, len(courseVO.Campuses))
	for i, campus := range courseVO.Campuses {
		campusIDs[i] = mapping.Data.GetCampusIDByName(campus)
	}

	// 构造查询条件，检查提案中的课程字段
	filter := bson.M{
		"course.name":       courseVO.Name,
		"course.code":       courseVO.Code,
		"course.department": departmentID,
		"course.category":   categoryID,
		"course.campuses":   bson.M{"$all": campusIDs, "$size": len(campusIDs)},
		"deleted":           false, // 只检查未删除的提案
	}

	// 如果提供了教师信息，则也加入查询条件
	if len(courseVO.Teachers) > 0 {
		teacherIDs := make([]string, len(courseVO.Teachers))
		for i, teacher := range courseVO.Teachers {
			teacherIDs[i] = teacher.ID
		}
		filter["course.teacherIds"] = bson.M{"$all": teacherIDs, "$size": len(teacherIDs)}
	} else {
		// 如果没有提供教师信息，则查询teacherIds为空或者不存在的记录
		filter["$or"] = []bson.M{
			{"course.teacherIds": bson.M{"$exists": false}},
			{"course.teacherIds": bson.M{"$size": 0}},
		}
	}

	// 查询提案中是否已存在该课程
	count, err := s.conn.CountDocuments(ctx, filter)
	if err != nil {
		return false, errorx.WrapByCode(err, errno.ErrProposalCourseFindInProposalFailed,
			errorx.KV("operation", "check proposal course existence"))
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
func (r *ProposalRepo) CountProposalByTarget(ctx context.Context, targetId string, targetType int32) (int64, error) {
	return r.conn.CountDocuments(ctx, bson.M{
		consts.TargetID: targetId,
		consts.Active:   bson.M{"$ne": false},
	})
}
