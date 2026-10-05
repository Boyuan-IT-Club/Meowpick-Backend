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
	"errors"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *ProposalService) GetTeacher(ctx context.Context, id string) (*dto.GetTeacherResp, error) {
	if _, err := s.actor(ctx, false); err != nil {
		return nil, err
	}
	var teacher model.Teacher
	err := s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).FindOne(ctx, bson.M{"_id": id}).Decode(&teacher)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errorx.New(errno.ErrProposalTargetNotFound)
	}
	if err != nil {
		return nil, err
	}
	contributors, err := s.CourseAssembler.EntityContributors(ctx, "teacher", id, "")
	if err != nil {
		return nil, err
	}
	return &dto.GetTeacherResp{Resp: dto.Success(), Teacher: &dto.TeacherVO{ID: teacher.ID, Name: teacher.Name, Title: teacher.Title, Department: teacherDepartmentName(teacher.Department)}, Contributors: contributors}, nil
}
func (s *ProposalService) EntityHistory(ctx context.Context, req *dto.EntityHistoryReq) (*dto.EntityHistoryResp, error) {
	if _, err := s.actor(ctx, false); err != nil {
		return nil, err
	}
	kind := model.ProposalUpdateCourse
	if req.TargetType == "teacher" {
		kind = model.ProposalUpdateTeacher
	}
	if _, err := s.targetPatch(ctx, kind, req.TargetID); err != nil {
		return nil, err
	}
	filter := bson.M{"revoked": bson.M{"$ne": true}, "targetId": req.TargetID, "type": bson.M{"$in": []string{model.ProposalCreateCourse, model.ProposalUpdateCourse}}}
	if req.TargetType == "teacher" {
		delete(filter, "targetId")
		delete(filter, "type")
		filter["$or"] = []bson.M{{"targetId": req.TargetID, "type": model.ProposalUpdateTeacher}, {"createdTeacherIds": req.TargetID}}
	}
	collection := s.ProposalRepo.Database().Collection(decisionCollection)
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}
	cursor, err := collection.Find(ctx, filter, page.FindPageOption(req.PageParam).SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var decisions []*model.ProposalDecision
	if err = cursor.All(ctx, &decisions); err != nil {
		return nil, err
	}
	history := []*dto.EntityHistoryItem{}
	for _, decision := range decisions {
		proposals, err := s.ProposalRepo.FindByIDs(ctx, decision.ProposalIDs)
		if err != nil {
			return nil, err
		}
		contributors, err := s.CourseAssembler.PublicContributors(ctx, proposals)
		if err != nil {
			return nil, err
		}
		item := &dto.EntityHistoryItem{DecisionBatchID: decision.ID, Type: decision.Type, ProposalIDs: decision.ProposalIDs, Contributors: contributors, Before: copyAs[dto.ProposalPatch](decision.BeforeValues), Final: copyAs[dto.ProposalPatch](decision.AfterValues), CreatedAt: decision.CreatedAt, Suggestions: []*dto.ProposalPatch{}}
		for _, p := range proposals {
			if p.Suggested != nil {
				item.Suggestions = append(item.Suggestions, copyAs[dto.ProposalPatch](p.Suggested))
			} else if p.Course != nil {
				item.Suggestions = append(item.Suggestions, coursePatch(copyAs[dto.ProposalCourseVO](p.Course)))
			}
		}
		if req.TargetType == "teacher" && decision.Type != model.ProposalUpdateTeacher {
			item.Type = "create_teacher"
			item.Before = nil
			item.Final = nil
			item.Suggestions = []*dto.ProposalPatch{}
			for _, teacher := range decision.CreatedTeachers {
				if teacher.ID == req.TargetID {
					item.Final = teacherPatch(&dto.TeacherVO{ID: teacher.ID, Name: teacher.Name, Title: teacher.Title, Department: teacherDepartmentName(teacher.Department)})
				}
			}
			for _, p := range proposals {
				if p.Course == nil {
					continue
				}
				for _, teacher := range p.Course.Teachers {
					if item.Final != nil && teacher.TeacherID == "" && teacher.Name == *item.Final.Name {
						item.Suggestions = append(item.Suggestions, teacherPatch(copyAs[dto.TeacherVO](teacher)))
					}
				}
			}
		}
		history = append(history, item)
	}
	// Older courses may carry an original proposal but no decision snapshot.
	// Keep that creation provenance without inventing the historical final fields.
	if req.TargetType == "course" {
		course, err := s.CourseRepo.FindByID(ctx, req.TargetID)
		if err != nil {
			return nil, err
		}
		if course != nil && course.ProposalID != "" {
			source, err := s.ProposalRepo.FindByIDIncludeDeleted(ctx, course.ProposalID)
			if err != nil {
				return nil, err
			}
			if source != nil && source.DecisionBatchID == "" && source.Status == mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved) {
				count, size := int64(1), int64(10)
				if req.PageParam != nil {
					count, size = req.PageParam.UnWrap()
				}
				offset := (count - 1) * size
				if offset <= total && offset+size > total {
					contributors, err := s.CourseAssembler.PublicContributors(ctx, []*model.Proposal{source})
					if err != nil {
						return nil, err
					}
					createdAt, err := s.ChangeLogRepo.FindLatestProposalApprovalTime(ctx, source.ID)
					if err != nil {
						return nil, err
					}
					if createdAt.IsZero() {
						createdAt = source.UpdatedAt
					}
					history = append(history, &dto.EntityHistoryItem{Legacy: true, DecisionBatchID: "legacy:" + source.ID, Type: model.ProposalCreateCourse, ProposalIDs: []string{source.ID}, Contributors: contributors, Suggestions: []*dto.ProposalPatch{coursePatch(copyAs[dto.ProposalCourseVO](source.Course))}, CreatedAt: createdAt})
				}
				total++
			}
		}
	}
	return &dto.EntityHistoryResp{Resp: dto.Success(), Total: total, History: history}, nil
}
