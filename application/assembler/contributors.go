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
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (a *CourseAssembler) EntityContributors(ctx context.Context, kind, id, legacyID string) ([]*dto.CourseContributorVO, error) {
	result := []*dto.CourseContributorVO{}
	if a.ProposalRepo == nil {
		return result, nil
	}
	ids := []string{}
	if legacyID != "" {
		ids = append(ids, legacyID)
	}
	if kind == "teacher" {
		var decisions []*model.ProposalDecision
		cursor, err := a.ProposalRepo.Database().Collection("proposal_decision").Find(ctx, bson.M{"createdTeacherIds": id, "revoked": bson.M{"$ne": true}})
		if err != nil {
			return nil, err
		}
		err = cursor.All(ctx, &decisions)
		cursor.Close(ctx)
		if err != nil {
			return nil, err
		}
		for _, decision := range decisions {
			ids = append(ids, decision.ProposalIDs...)
		}
	}
	kinds := []string{model.ProposalCreateCourse, model.ProposalUpdateCourse}
	if kind == "teacher" {
		kinds = []string{model.ProposalUpdateTeacher}
	}
	filter := bson.M{"status": mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved), "deleted": bson.M{"$ne": true}, "showUsername": true, "$or": []bson.M{{"targetId": id, "type": bson.M{"$in": kinds}}, {"_id": bson.M{"$in": ids}}}}
	var proposals []*model.Proposal
	cursor, err := a.ProposalRepo.Database().Collection(repo.ProposalCollectionName).Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "updatedAt", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &proposals)
	cursor.Close(ctx)
	if err != nil {
		return nil, err
	}
	return a.PublicContributors(ctx, proposals)
}
func (a *CourseAssembler) PublicContributors(ctx context.Context, proposals []*model.Proposal) ([]*dto.CourseContributorVO, error) {
	result := []*dto.CourseContributorVO{}
	seen := map[string]bool{}
	ids := []string{}
	for _, p := range proposals {
		if p.ShowUsername && !seen[p.UserID] {
			ids = append(ids, p.UserID)
			seen[p.UserID] = true
		}
	}
	users, err := a.UserRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, user := range users {
		names[user.ID] = user.Username
	}
	seen = map[string]bool{}
	for _, p := range proposals {
		if !p.ShowUsername || seen[p.UserID] {
			continue
		}
		seen[p.UserID] = true
		result = append(result, &dto.CourseContributorVO{ProposalID: p.ID, UserID: p.UserID, Username: names[p.UserID], ShowUsername: true})
	}
	return result, nil
}
