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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/Boyuan-IT-Club/go-kit/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const decisionCollection = "proposal_decision"

func (s *ProposalService) actor(ctx context.Context, requireAdmin bool) (string, error) {
	id, _ := ctx.Value(consts.CtxUserID).(string)
	if id == "" {
		return "", errorx.New(errno.ErrUserNotLogin)
	}
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errorx.New(errno.ErrUserNotFound)
	}
	if requireAdmin && !user.Admin {
		return "", errorx.New(errno.ErrUserNotAdmin)
	}
	return id, nil
}
func (s *ProposalService) workflowTransaction(ctx context.Context, fn func(mongo.SessionContext) error) error {
	// All proposal mutations acquire the same document before reads. This prevents
	// phantom pending members and competing approvals creating duplicate courses.
	if err := s.ProposalRepo.AcquireCreateGuards(ctx, "workflow", "workflow-v1"); err != nil {
		return err
	}
	return s.ProposalRepo.WithTransaction(ctx, func(tx mongo.SessionContext) error {
		if err := s.ProposalRepo.AcquireCreateGuards(tx, "workflow", "workflow-v1"); err != nil {
			return err
		}
		return fn(tx)
	})
}
func pendingStatus() int32 {
	return mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusPending)
}
func (s *ProposalService) pending(ctx context.Context) ([]*model.Proposal, error) {
	result := []*model.Proposal{}
	cursor, err := s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).Find(ctx, bson.M{"status": pendingStatus(), "deleted": bson.M{"$ne": true}}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &result)
	return result, err
}
func (s *ProposalService) targetPatch(ctx context.Context, kind, id string) (*dto.ProposalPatch, error) {
	if kind == model.ProposalUpdateTeacher {
		var teacher model.Teacher
		err := s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).FindOne(ctx, bson.M{"_id": id}).Decode(&teacher)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errorx.New(errno.ErrProposalTargetNotFound)
		}
		if err != nil {
			return nil, err
		}
		return teacherPatch(&dto.TeacherVO{ID: teacher.ID, Name: teacher.Name, Title: teacher.Title, Department: func() string {
			if teacher.Department == 0 {
				return ""
			}
			return mapping.Data.GetDepartmentNameByID(teacher.Department)
		}()}), nil
	}
	course, err := s.CourseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, errorx.New(errno.ErrProposalTargetNotFound)
	}
	vo, err := s.CourseAssembler.ToProposalCourseVOFromCourse(ctx, course)
	if err != nil {
		return nil, err
	}
	return coursePatch(vo), nil
}
func (s *ProposalService) prepareProposal(ctx context.Context, req *dto.CreateProposalReq) (*model.Proposal, error) {
	kind := req.Type
	if kind == "" {
		kind = model.ProposalCreateCourse
	}
	p := &model.Proposal{Type: kind, TargetID: strings.TrimSpace(req.TargetID), Content: req.Content, ShowUsername: req.ShowUsername, Status: pendingStatus()}
	if kind == model.ProposalCreateCourse {
		if req.Suggested != nil || p.TargetID != "" {
			return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "targetId/suggested"))
		}
		normalizeCourse(req.Course)
		if err := validateProposalInput("course", req.Course); err != nil {
			return nil, err
		}
		if err := s.validateTeacherIDs(ctx, req.Course.Teachers); err != nil {
			return nil, err
		}
		p.Course = copyAs[model.ProposalCourse](req.Course)
		p.DisplayName = req.Course.Name
	} else {
		if kind != model.ProposalUpdateCourse && kind != model.ProposalUpdateTeacher {
			return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "type"))
		}
		if p.TargetID == "" || req.Course != nil {
			return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "targetId/course"))
		}
		suggested := copyAs[dto.ProposalPatch](req.Suggested)
		if err := validatePatch(kind, suggested); err != nil {
			return nil, err
		}
		current, err := s.targetPatch(ctx, kind, p.TargetID)
		if err != nil {
			return nil, err
		}
		base, changes := selectChanged(current, suggested)
		if len(patchValues(changes)) == 0 {
			return nil, errorx.New(errno.ErrProposalNoChanges)
		}
		if err := s.validatePatchReferences(ctx, changes); err != nil {
			return nil, err
		}
		p.Before = copyAs[model.ProposalPatch](base)
		p.Suggested = copyAs[model.ProposalPatch](changes)
		p.DisplayName = *current.Name
	}
	return p, nil
}
func (s *ProposalService) validateTeacherIDs(ctx context.Context, teachers []*dto.TeacherVO) error {
	identities := map[string]bool{}
	for _, teacher := range teachers {
		if teacher == nil || strings.TrimSpace(teacher.Name) == "" {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "teachers"))
		}
		key := teacherIdentities([]*dto.TeacherVO{teacher})[0]
		if identities[key] {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "teachers duplicate"))
		}
		identities[key] = true
		if teacher.ID != "" {
			var stored model.Teacher
			if err := s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).FindOne(ctx, bson.M{"_id": teacher.ID}).Decode(&stored); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					return errorx.New(errno.ErrProposalTargetNotFound)
				}
				return err
			}
			teacher.Name = stored.Name
			teacher.Title = stored.Title
			teacher.Department = teacherDepartmentName(stored.Department)
		}
	}
	return nil
}

func (s *ProposalService) validatePatchReferences(ctx context.Context, patch *dto.ProposalPatch) error {
	if patch.Teachers != nil {
		if err := s.validateTeacherIDs(ctx, *patch.Teachers); err != nil {
			return err
		}
	}
	if patch.Campuses != nil {
		for _, campus := range *patch.Campuses {
			if mapping.Data.GetCampusIDByName(campus) == 0 {
				return errorx.New(errno.ErrProposalInvalidCampus)
			}
		}
	}
	return nil
}

func (s *ProposalService) CreateProposal(ctx context.Context, req *dto.CreateProposalReq) (*dto.CreateProposalResp, error) {
	return s.createUnified(ctx, req, "")
}
func (s *ProposalService) ResubmitProposal(ctx context.Context, req *dto.ResubmitProposalReq) (*dto.ResubmitProposalResp, error) {
	resp, err := s.createUnified(ctx, &dto.CreateProposalReq{Type: req.Type, TargetID: req.TargetID, Suggested: req.Suggested, Course: req.Course, Content: req.Content, ShowUsername: req.ShowUsername}, req.ProposalID)
	if err != nil {
		return nil, err
	}
	return &dto.ResubmitProposalResp{Resp: dto.Success(), PreviousProposalID: req.ProposalID, ProposalID: resp.ProposalID, Proposal: resp.Proposal}, nil
}
func (s *ProposalService) createUnified(ctx context.Context, req *dto.CreateProposalReq, previousID string) (*dto.CreateProposalResp, error) {
	actor, err := s.actor(ctx, false)
	if err != nil {
		return nil, err
	}
	var proposal *model.Proposal
	duplicates := []string{}
	err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
		duplicates = []string{}
		p, err := s.prepareProposal(tx, req)
		if err != nil {
			return err
		}
		user, err := s.UserRepo.FindByID(tx, actor)
		if err != nil {
			return err
		}
		if user == nil {
			return errorx.New(errno.ErrUserNotFound)
		}
		count, err := s.ProposalRepo.CountByUserToday(tx, actor)
		if err != nil {
			return err
		}
		limit := getDailyProposalLimit(user.Contribution)
		if count >= limit {
			return errorx.New(errno.ErrDailyProposalLimitReached, errorx.KV("limit", strconv.FormatInt(limit, 10)))
		}
		pending, err := s.pending(tx)
		if err != nil {
			return err
		}
		for _, other := range pending {
			if p.EffectiveType() == other.EffectiveType() && proposalAutoKey(p) == proposalAutoKey(other) {
				if other.UserID == actor {
					return errorx.New(errno.ErrProposalCourseFoundInProposals)
				}
				duplicates = append(duplicates, other.ID)
				if other.FinalCourse != nil {
					p.FinalCourse = copyAs[model.ProposalCourse](other.FinalCourse)
				}
				if other.Final != nil {
					p.Final = copyAs[model.ProposalPatch](other.Final)
				}
			}
		}
		now := time.Now().UTC()
		p.ID = primitive.NewObjectID().Hex()
		p.UserID = actor
		p.CreatedAt = now
		p.UpdatedAt = now
		if previousID != "" {
			previous, err := s.ProposalRepo.FindByID(tx, previousID)
			if err != nil {
				return err
			}
			if previous == nil || previous.UserID != actor {
				return errorx.New(errno.ErrProposalNotFound)
			}
			if previous.Status != mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected) {
				return errorx.New(errno.ErrProposalStatusNotRejected)
			}
			previous.Deleted = true
			previous.DeletedAt = now
			previous.UpdatedAt = now
			if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": previous.ID}, previous); err != nil {
				return err
			}
			if err = s.auditProposal(tx, previous, consts.ActionTypeDeleteProposal, actor, "重新提交：删除旧提案", "", false); err != nil {
				return err
			}
		}
		if err = s.ProposalRepo.Insert(tx, p); err != nil {
			return err
		}
		proposal = p
		return s.auditProposal(tx, p, consts.ActionTypeCreateProposal, actor, "创建提案", "", false)
	})
	if err != nil {
		return nil, err
	}
	vo, err := s.ProposalAssembler.ToProposalVO(ctx, proposal, actor)
	if err != nil {
		return nil, err
	}
	return &dto.CreateProposalResp{Resp: dto.Success(), ProposalID: proposal.ID, Proposal: vo, PendingDuplicateIDs: duplicates}, nil
}
func (s *ProposalService) auditProposal(ctx context.Context, p *model.Proposal, action int32, actor, content, trigger string, automatic bool) error {
	kind := "course"
	if p.EffectiveType() == model.ProposalUpdateTeacher {
		kind = "teacher"
	}
	source := consts.UpdateSourceUser
	if action != consts.ActionTypeCreateProposal && action != consts.ActionTypeDeleteProposal {
		source = consts.UpdateSourceAdmin
	}
	return s.ChangeLogRepo.Insert(ctx, &model.ChangeLog{ID: primitive.NewObjectID().Hex(), TargetID: p.ID, TargetType: consts.TargetTypeProposal, Action: action, Content: content, UpdateSource: source, ProposalID: p.ID, UserID: actor, UpdatedAt: time.Now().UTC(), ProposalType: p.EffectiveType(), EntityType: kind, EntityID: p.TargetID, DecisionBatchID: p.DecisionBatchID, TriggerProposalID: trigger, Automatic: automatic, Snapshot: copyAs[model.Proposal](p)})
}
func (s *ProposalService) UpdateProposal(ctx context.Context, req *dto.UpdateProposalReq) (*dto.UpdateProposalResp, error) {
	actor, err := s.actor(ctx, true)
	if err != nil {
		return nil, err
	}
	err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
		old, err := s.ProposalRepo.FindByID(tx, req.ProposalID)
		if err != nil {
			return err
		}
		if old == nil {
			return errorx.New(errno.ErrProposalNotFound)
		}
		if old.Status != pendingStatus() {
			return errorx.New(errno.ErrProposalAlreadyProcessed)
		}
		kind := req.Type
		if kind == "" {
			kind = old.EffectiveType()
		}
		target := req.TargetID
		if target == "" {
			target = old.TargetID
		}
		if kind != old.EffectiveType() || target != old.TargetID {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "type/targetId"))
		}
		if kind == model.ProposalCreateCourse && req.Suggested != nil {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "suggested"))
		}
		if kind != model.ProposalCreateCourse && req.Course != nil {
			return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "course"))
		}
		var finalCourse *model.ProposalCourse
		var finalPatch *model.ProposalPatch
		if kind == model.ProposalCreateCourse {
			course := copyAs[dto.ProposalCourseVO](req.Course)
			normalizeCourse(course)
			if err := validateProposalInput("course", course); err != nil {
				return err
			}
			if err := s.validateTeacherIDs(tx, course.Teachers); err != nil {
				return err
			}
			finalCourse = copyAs[model.ProposalCourse](course)
		} else {
			patch := copyAs[dto.ProposalPatch](req.Suggested)
			if err := validatePatch(kind, patch); err != nil {
				return err
			}
			allowed := patchValues(copyAs[dto.ProposalPatch](old.Suggested))
			suggested := patchValues(copyAs[dto.ProposalPatch](old.Suggested))
			if old.Final != nil {
				suggested = patchValues(copyAs[dto.ProposalPatch](old.Final))
			}
			for field, value := range patchValues(patch) {
				if _, ok := allowed[field]; !ok {
					return errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "suggested."+field))
				}
				suggested[field] = value
			}
			editedValues := patchFromValues(suggested)
			if err := s.validatePatchReferences(tx, editedValues); err != nil {
				return err
			}
			finalPatch = copyAs[model.ProposalPatch](editedValues)
		}
		pending, err := s.pending(tx)
		if err != nil {
			return err
		}
		for _, member := range pending {
			if member.EffectiveType() != kind || proposalAutoKey(member) != proposalAutoKey(old) {
				continue
			}
			// Administrator edits apply to pending proposals. Author submissions and their
			// original baselines remain immutable for scoring and attribution.
			member.FinalCourse = copyAs[model.ProposalCourse](finalCourse)
			member.Final = copyAs[model.ProposalPatch](finalPatch)
			member.UpdatedAt = time.Now().UTC()
			if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": member.ID}, member); err != nil {
				return err
			}
			if err = s.auditProposal(tx, member, consts.ActionTypeUpdateProposal, actor, "管理员修改待审提案资料（同步相同提案）", old.ID, member.ID != old.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &dto.UpdateProposalResp{Resp: dto.Success(), ProposalID: req.ProposalID}, nil
}

type approvalState struct {
	Preview         *dto.ApprovalPreviewResp
	Members         []*model.Proposal
	Primary         *model.Proposal
	Before          *dto.ProposalPatch
	Final           *dto.ProposalPatch
	Course          *dto.ProposalCourseVO
	BlockedConflict bool
	FormalDuplicate bool
}

func (s *ProposalService) buildApproval(ctx context.Context, req *dto.ToggleProposalReq, actor string) (*approvalState, error) {
	all, err := s.pending(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[string]*model.Proposal{}
	for _, p := range all {
		byID[p.ID] = p
	}
	primary := byID[req.ProposalID]
	if primary == nil {
		return nil, errorx.New(errno.ErrProposalAlreadyProcessed)
	}
	if primary.EffectiveType() == model.ProposalCreateCourse && req.Final != nil {
		return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "final"))
	}
	if primary.EffectiveType() != model.ProposalCreateCourse && req.FinalCourse != nil {
		return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "finalCourse"))
	}
	selected := map[string]bool{primary.ID: true}
	for _, id := range req.ProposalIDs {
		selected[id] = true
	}
	for id := range selected {
		p := byID[id]
		if p == nil {
			return nil, errorx.New(errno.ErrProposalPreviewStale)
		}
		if p.EffectiveType() != primary.EffectiveType() || (p.EffectiveType() != model.ProposalCreateCourse && p.TargetID != primary.TargetID) {
			return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "proposalIds"))
		}
		if p.EffectiveType() != model.ProposalCreateCourse && proposalAutoKey(p) != proposalAutoKey(primary) {
			return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "proposalIds"))
		}
	}
	keys := map[string]bool{}
	for id := range selected {
		keys[proposalAutoKey(byID[id])] = true
	}
	state := &approvalState{Primary: primary, Preview: &dto.ApprovalPreviewResp{Resp: dto.Success(), Members: []*dto.ProposalCandidate{}, Suspicious: []*dto.ProposalCandidate{}, ExistingCourses: []*dto.ProposalCourseVO{}, TeacherCandidates: map[string][]*dto.TeacherVO{}, Conflicts: []dto.ProposalDifference{}, CanApprove: true}}
	candidateNames := map[string]bool{}
	for id := range selected {
		if byID[id].Course != nil {
			candidateNames[byID[id].Course.Name] = true
		}
	}
	if req.FinalCourse != nil {
		candidateNames[strings.TrimSpace(req.FinalCourse.Name)] = true
	}
	relevant := []*model.Proposal{}
	for _, p := range all {
		if p.EffectiveType() == primary.EffectiveType() && keys[proposalAutoKey(p)] {
			state.Members = append(state.Members, p)
			selected[p.ID] = true
			relevant = append(relevant, p)
		} else if primary.EffectiveType() == model.ProposalCreateCourse && p.EffectiveType() == model.ProposalCreateCourse && p.Course != nil && candidateNames[p.Course.Name] {
			relevant = append(relevant, p)
		}
	}
	if primary.EffectiveType() == model.ProposalCreateCourse {
		state.Course = copyAs[dto.ProposalCourseVO](primary.Course)
		if primary.FinalCourse != nil {
			state.Course = copyAs[dto.ProposalCourseVO](primary.FinalCourse)
		}
		if req.FinalCourse != nil {
			state.Course = copyAs[dto.ProposalCourseVO](req.FinalCourse)
		}
		normalizeCourse(state.Course)
		if err = validateProposalInput("course", state.Course); err != nil {
			return nil, err
		}
		if err = s.validateTeacherIDs(ctx, state.Course.Teachers); err != nil {
			return nil, err
		}
		state.Preview.FinalCourse = state.Course
	} else {
		state.Before, err = s.targetPatch(ctx, primary.EffectiveType(), primary.TargetID)
		if err != nil {
			return nil, err
		}
		proposed := patchValues(copyAs[dto.ProposalPatch](primary.Suggested))
		overrides := patchValues(copyAs[dto.ProposalPatch](primary.Final))
		for field, value := range patchValues(req.Final) {
			overrides[field] = value
		}
		for field, value := range overrides {
			if _, ok := proposed[field]; !ok {
				return nil, errorx.New(errno.ErrProposalInvalidField, errorx.KV("field", "final."+field))
			}
			proposed[field] = value
		}
		state.Final = patchFromValues(proposed)
		if err = validatePatch(primary.EffectiveType(), state.Final); err != nil {
			return nil, err
		}
		current := patchValues(state.Before)
		original := patchValues(copyAs[dto.ProposalPatch](primary.Before))
		suggested := patchValues(copyAs[dto.ProposalPatch](primary.Suggested))
		fields := []string{}
		for field := range suggested {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, field := range fields {
			status := "unchanged"
			if fieldEqual(field, current[field], suggested[field]) {
				status = "realized"
			} else if !fieldEqual(field, current[field], original[field]) {
				status = "conflict"
				if _, ok := overrides[field]; !ok {
					state.BlockedConflict = true
				}
			}
			state.Preview.Conflicts = append(state.Preview.Conflicts, dto.ProposalDifference{Field: field, Original: original[field], Current: current[field], Suggested: suggested[field], State: status})
		}
		_, actual := selectChanged(state.Before, state.Final)
		if len(patchValues(actual)) == 0 {
			state.Preview.CanApprove = false
		}
		state.Preview.Final = state.Final
		if primary.EffectiveType() == model.ProposalUpdateCourse {
			merged := patchValues(state.Before)
			for field, value := range patchValues(state.Final) {
				merged[field] = value
			}
			state.Course = copyAs[dto.ProposalCourseVO](merged)
			state.Course.ID = primary.TargetID
			normalizeCourse(state.Course)
			if err = validateProposalInput("course", state.Course); err != nil {
				return nil, err
			}
			if err = s.validateTeacherIDs(ctx, state.Course.Teachers); err != nil {
				return nil, err
			}
			state.Preview.FinalCourse = state.Course
		}
	}
	if state.BlockedConflict {
		state.Preview.CanApprove = false
	}
	// Formal course candidates remain separate from pending proposals.
	if state.Course != nil {
		names := normalizedSet([]string{state.Course.Name})
		if primary.Course != nil {
			names = normalizedSet(append(names, primary.Course.Name))
		}
		var courses []*model.Course
		cursor, err := s.ProposalRepo.Database().Collection(repo.CourseCollectionName).Find(ctx, bson.M{"name": bson.M{"$in": names}, "deleted": bson.M{"$ne": true}}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)
		if err = cursor.All(ctx, &courses); err != nil {
			return nil, err
		}
		for _, course := range courses {
			if course.ID == primary.TargetID {
				continue
			}
			vo, err := s.CourseAssembler.ToProposalCourseVOFromCourse(ctx, course)
			if err != nil {
				return nil, err
			}
			vo.ID = course.ID
			state.Preview.ExistingCourses = append(state.Preview.ExistingCourses, vo)
			if newCourseKey(vo) == newCourseKey(state.Course) {
				state.FormalDuplicate = true
				state.Preview.CanApprove = false
			}
		}
		for _, teacher := range state.Course.Teachers {
			if teacher.ID != "" {
				continue
			}
			var matches []*model.Teacher
			cursor, err := s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).Find(ctx, bson.M{"name": teacher.Name}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
			if err != nil {
				return nil, err
			}
			err = cursor.All(ctx, &matches)
			cursor.Close(ctx)
			if err != nil {
				return nil, err
			}
			if len(matches) > 0 {
				vos := []*dto.TeacherVO{}
				for _, stored := range matches {
					vos = append(vos, &dto.TeacherVO{ID: stored.ID, Name: stored.Name, Title: stored.Title, Department: teacherDepartmentName(stored.Department)})
				}
				state.Preview.TeacherCandidates[teacher.Name] = vos
			}
		}
	}
	previewAuthors := map[string]bool{}
	previewProposals, err := s.ProposalAssembler.ToProposalVOArray(ctx, relevant, actor)
	if err != nil {
		return nil, err
	}
	for i, p := range relevant {
		vo := previewProposals[i]
		candidate := &dto.ProposalCandidate{Proposal: vo, Differences: []dto.ProposalDifference{}, Automatic: proposalAutoKey(p) == proposalAutoKey(primary)}
		if state.Course != nil && p.EffectiveType() == model.ProposalCreateCourse {
			original := copyAs[dto.ProposalCourseVO](p.Course)
			candidate.Differences = courseDifferences(original, state.Course)
			candidate.ExpectedContribution = calcContributionScore(original, state.Course)
		} else {
			candidate.ExpectedContribution = modificationScore(copyAs[dto.ProposalPatch](p.Suggested), state.Before, state.Final)
		}
		if selected[p.ID] {
			if previewAuthors[p.UserID] {
				candidate.ExpectedContribution = 0
			}
			previewAuthors[p.UserID] = true
			state.Preview.Members = append(state.Preview.Members, candidate)
		} else {
			state.Preview.Suspicious = append(state.Preview.Suspicious, candidate)
		}
	}
	var revision bson.M
	if primary.EffectiveType() != model.ProposalCreateCourse {
		collection := repo.CourseCollectionName
		if primary.EffectiveType() == model.ProposalUpdateTeacher {
			collection = repo.TeacherCollectionName
		}
		if err := s.ProposalRepo.Database().Collection(collection).FindOne(ctx, bson.M{"_id": primary.TargetID}).Decode(&revision); err != nil {
			return nil, err
		}
	}
	state.Preview.PreviewToken = valueHash([]interface{}{revision, relevant, state.Before, state.Course, state.Final, state.Preview.ExistingCourses, state.Preview.TeacherCandidates, normalizedSet(req.ProposalIDs), normalizedSet(req.ConfirmedNewTeachers)})
	return state, nil
}
func (s *ProposalService) PreviewApproval(ctx context.Context, req *dto.ToggleProposalReq) (*dto.ApprovalPreviewResp, error) {
	actor, err := s.actor(ctx, true)
	if err != nil {
		return nil, err
	}
	var state *approvalState
	err = s.ProposalRepo.WithTransaction(ctx, func(tx mongo.SessionContext) error {
		var err error
		state, err = s.buildApproval(tx, req, actor)
		return err
	})
	if err != nil {
		return nil, err
	}
	return state.Preview, nil
}
func (s *ProposalService) materializeCourse(ctx context.Context, course *dto.ProposalCourseVO, batchID string) (*model.Course, []string, error) {
	// Existing teachers are reused by ID; names never implicitly merge records.
	created := []string{}
	now := time.Now().UTC()
	for _, teacher := range course.Teachers {
		if teacher.ID != "" {
			continue
		}
		department, err := resolveTeacherDepartment(ctx, teacher.Department)
		if err != nil {
			return nil, nil, err
		}
		teacher.ID = primitive.NewObjectID().Hex()
		stored := &model.Teacher{ID: teacher.ID, Name: teacher.Name, Title: teacher.Title, Department: department, CreatedAt: now, UpdatedAt: now, DecisionBatchID: batchID}
		if err = s.TeacherRepo.Insert(ctx, stored); err != nil {
			return nil, nil, err
		}
		created = append(created, teacher.ID)
	}
	stored, err := s.CourseAssembler.ToCourseDBFromProposalCourse(ctx, course)
	if err != nil {
		return nil, nil, err
	}
	stored.DecisionBatchID = batchID
	return stored, created, nil
}
func resolveTeacherDepartment(ctx context.Context, name string) (int32, error) {
	if strings.TrimSpace(name) == "" {
		return 0, nil
	}
	return mapping.Data.ResolveOrCreateDepartment(ctx, name)
}
func (s *ProposalService) ApproveProposal(ctx context.Context, req *dto.ToggleProposalReq) (*dto.ToggleProposalResp, error) {
	actor, err := s.actor(ctx, true)
	if err != nil {
		return nil, err
	}
	if req.PreviewToken == "" {
		return nil, errorx.New(errno.ErrProposalPreviewRequired)
	}
	var batch *model.ProposalDecision
	users := map[string]bool{}
	err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
		users = map[string]bool{}
		state, err := s.buildApproval(tx, req, actor)
		if err != nil {
			return err
		}
		if req.PreviewToken != state.Preview.PreviewToken {
			return errorx.New(errno.ErrProposalPreviewStale)
		}
		if state.BlockedConflict {
			return errorx.New(errno.ErrProposalFieldConflict)
		}
		if !state.Preview.CanApprove {
			if state.FormalDuplicate {
				return errorx.New(errno.ErrProposalCourseFoundInCourses)
			}
			return errorx.New(errno.ErrProposalNoChanges)
		}
		for name := range state.Preview.TeacherCandidates {
			found := false
			for _, confirmed := range req.ConfirmedNewTeachers {
				if strings.TrimSpace(confirmed) == name {
					found = true
				}
			}
			if !found {
				return errorx.New(errno.ErrProposalTeacherConfirmation)
			}
		}
		now := time.Now().UTC()
		batch = &model.ProposalDecision{ID: primitive.NewObjectID().Hex(), Type: state.Primary.EffectiveType(), TriggerID: state.Primary.ID, CreatedAt: now, ProposalIDs: []string{}, CreatedTeacherIDs: []string{}}
		if state.Course != nil {
			if batch.Type == model.ProposalUpdateCourse {
				batch.BeforeCourse, err = s.CourseRepo.FindByID(tx, state.Primary.TargetID)
				if err != nil {
					return err
				}
			}
			stored, created, err := s.materializeCourse(tx, state.Course, batch.ID)
			if err != nil {
				return err
			}
			batch.CreatedTeacherIDs = created
			for _, id := range created {
				var teacher model.Teacher
				if err = s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).FindOne(tx, bson.M{"_id": id}).Decode(&teacher); err != nil {
					return err
				}
				batch.CreatedTeachers = append(batch.CreatedTeachers, &teacher)
			}
			stored.UpdatedAt = now
			if batch.BeforeCourse != nil {
				stored.ID = batch.BeforeCourse.ID
				stored.CreatedAt = batch.BeforeCourse.CreatedAt
				stored.ProposalID = batch.BeforeCourse.ProposalID
				if _, err = s.ProposalRepo.Database().Collection(repo.CourseCollectionName).ReplaceOne(tx, bson.M{"_id": stored.ID}, stored); err != nil {
					return err
				}
			} else {
				stored.ID = primitive.NewObjectID().Hex()
				stored.CreatedAt = now
				var previous model.ProposalDecision
				previousErr := s.ProposalRepo.Database().Collection(decisionCollection).FindOne(tx, bson.M{"proposalIds": state.Primary.ID, "type": model.ProposalCreateCourse, "revoked": true}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&previous)
				if previousErr != nil && !errors.Is(previousErr, mongo.ErrNoDocuments) {
					return previousErr
				}
				var restore *model.Course
				if previousErr == nil {
					var old model.Course
					findErr := s.ProposalRepo.Database().Collection(repo.CourseCollectionName).FindOne(tx, bson.M{"_id": previous.TargetID, "deleted": true}).Decode(&old)
					if findErr == nil {
						restore = &old
					} else if !errors.Is(findErr, mongo.ErrNoDocuments) {
						return findErr
					}
				}
				if restore == nil && errors.Is(previousErr, mongo.ErrNoDocuments) {
					old, findErr := s.CourseRepo.FindByProposalIDIncludeDeleted(tx, state.Primary.ID)
					if findErr != nil {
						return findErr
					}
					if old != nil && old.Deleted {
						restore = old
					}
				}
				if restore != nil {
					stored.ID = restore.ID
					stored.CreatedAt = restore.CreatedAt
				}
				stored.ProposalID = state.Primary.ID
				if restore != nil {
					if _, err = s.ProposalRepo.Database().Collection(repo.CourseCollectionName).ReplaceOne(tx, bson.M{"_id": restore.ID, "deleted": true}, stored); err != nil {
						return err
					}
				} else if err = s.CourseRepo.Insert(tx, stored); err != nil {
					return err
				}
			}
			batch.TargetID = stored.ID
			batch.AfterCourse = stored
		} else {
			var teacher model.Teacher
			err = s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).FindOne(tx, bson.M{"_id": state.Primary.TargetID}).Decode(&teacher)
			if err != nil {
				return err
			}
			batch.BeforeTeacher = copyAs[model.Teacher](&teacher)
			if state.Final.Name != nil {
				teacher.Name = *state.Final.Name
			}
			if state.Final.Title != nil {
				teacher.Title = *state.Final.Title
			}
			if state.Final.Department != nil {
				teacher.Department, err = resolveTeacherDepartment(tx, *state.Final.Department)
				if err != nil {
					return err
				}
			}
			teacher.UpdatedAt = now
			teacher.DecisionBatchID = batch.ID
			if _, err = s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).ReplaceOne(tx, bson.M{"_id": teacher.ID}, &teacher); err != nil {
				return err
			}
			batch.TargetID = teacher.ID
			batch.AfterTeacher = &teacher
		}
		batch.BeforeValues = copyAs[model.ProposalPatch](state.Before)
		batch.AfterValues = copyAs[model.ProposalPatch](state.Final)
		if batch.Type == model.ProposalCreateCourse {
			batch.AfterValues = copyAs[model.ProposalPatch](coursePatch(state.Course))
		}
		for _, p := range state.Members {
			score := int64(0)
			if batch.Type == model.ProposalCreateCourse {
				score = calcContributionScore(copyAs[dto.ProposalCourseVO](p.Course), state.Course)
			} else {
				score = modificationScore(copyAs[dto.ProposalPatch](p.Suggested), state.Before, state.Final)
			}
			if users[p.UserID] {
				score = 0
			}
			users[p.UserID] = true
			p.Type = p.EffectiveType()
			p.Status = mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved)
			p.TargetID = batch.TargetID
			p.DecisionBatchID = batch.ID
			p.UpdatedAt = now
			p.Contribution = score
			p.RejectReason = ""
			p.Final = copyAs[model.ProposalPatch](state.Final)
			if state.Course != nil {
				p.FinalCourse = copyAs[model.ProposalCourse](state.Course)
			}
			if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": p.ID, "status": pendingStatus()}, p); err != nil {
				return err
			}
			if err = s.UserRepo.IncrementContribution(tx, p.UserID, score); err != nil {
				return err
			}
			batch.ProposalIDs = append(batch.ProposalIDs, p.ID)
			if err = s.auditProposal(tx, p, consts.ActionTypeApproveProposal, actor, "审批通过", state.Primary.ID, proposalAutoKey(p) == proposalAutoKey(state.Primary) && p.ID != state.Primary.ID); err != nil {
				return err
			}
		}
		if _, err = s.ProposalRepo.Database().Collection(decisionCollection).InsertOne(tx, batch); err != nil {
			return err
		}
		targetType := mapping.Data.GetChangeLogTargetTypeIDByName("course")
		if batch.Type == model.ProposalUpdateTeacher {
			targetType = mapping.Data.GetChangeLogTargetTypeIDByName("teacher")
		}
		return s.ChangeLogRepo.Insert(tx, &model.ChangeLog{ID: primitive.NewObjectID().Hex(), TargetID: batch.TargetID, TargetType: targetType, Action: consts.ActionTypeApproveProposal, Content: "共同审批：一次资料变更", UpdateSource: consts.UpdateSourceAdmin, UserID: actor, UpdatedAt: now, DecisionBatchID: batch.ID, EntityID: batch.TargetID, EntityType: func() string {
			if batch.Type == model.ProposalUpdateTeacher {
				return "teacher"
			}
			return "course"
		}(), ProposalType: batch.Type, TriggerProposalID: state.Primary.ID, BeforeValues: copyAs[model.ProposalPatch](batch.BeforeValues), AfterValues: copyAs[model.ProposalPatch](batch.AfterValues)})
	})
	if err != nil {
		return nil, err
	}
	s.afterWorkflow(ctx, users)
	count, _ := s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).CountDocuments(ctx, bson.M{"status": pendingStatus(), "deleted": bson.M{"$ne": true}})
	return &dto.ToggleProposalResp{Resp: dto.Success(), Proposal: true, ProposalCnt: count, DecisionBatchID: batch.ID, ProposalIDs: batch.ProposalIDs, TargetID: batch.TargetID}, nil
}
func (s *ProposalService) afterWorkflow(ctx context.Context, users map[string]bool) {
	for id := range users {
		if err := s.UserRepo.InvalidateByID(ctx, id); err != nil {
			logs.CtxWarnf(ctx, "workflow user cache invalidation failed: %v", err)
		}
	}
	if err := mapping.Data.Refresh(ctx); err != nil {
		logs.CtxWarnf(ctx, "workflow mapping refresh failed: %v", err)
	}
}
func (s *ProposalService) RejectProposal(ctx context.Context, req *dto.RejectProposalReq) (*dto.RejectProposalResp, error) {
	actor, err := s.actor(ctx, true)
	if err != nil {
		return nil, err
	}
	ids := normalizedSet(append(req.ProposalIDs, req.ProposalID))
	if len(ids) == 0 || ids[0] == "" {
		return nil, errorx.New(errno.ErrProposalIDRequired)
	}
	err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
		for _, id := range ids {
			p, err := s.ProposalRepo.FindByID(tx, id)
			if err != nil {
				return err
			}
			if p == nil {
				return errorx.New(errno.ErrProposalNotFound)
			}
			if p.Status != pendingStatus() {
				return errorx.New(errno.ErrProposalAlreadyProcessed)
			}
			p.Status = mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected)
			p.RejectReason = strings.TrimSpace(req.Reason)
			p.UpdatedAt = time.Now().UTC()
			if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": id}, p); err != nil {
				return err
			}
			if err = s.auditProposal(tx, p, consts.ActionTypeRejectProposal, actor, p.RejectReason, req.ProposalID, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	count, _ := s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).CountDocuments(ctx, bson.M{"status": pendingStatus(), "deleted": bson.M{"$ne": true}})
	return &dto.RejectProposalResp{Resp: dto.Success(), Rejected: true, PendingCount: count, ProposalIDs: ids}, nil
}

func (s *ProposalService) RevokeProposal(ctx context.Context, req *dto.RevokeProposalReq) (*dto.RevokeProposalResp, error) {
	actor, err := s.actor(ctx, true)
	if err != nil {
		return nil, err
	}
	p, err := s.ProposalRepo.FindByID(ctx, req.ProposalID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errorx.New(errno.ErrProposalNotFound)
	}
	if req.ActionType == consts.RevokeActionReject {
		err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
			member, err := s.ProposalRepo.FindByID(tx, req.ProposalID)
			if err != nil {
				return err
			}
			if member == nil {
				return errorx.New(errno.ErrProposalNotFound)
			}
			if member.Status != mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusRejected) {
				return errorx.New(errno.ErrProposalStatusNotRejected)
			}
			member.Status, member.RejectReason, member.UpdatedAt = pendingStatus(), "", time.Now().UTC()
			if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": member.ID}, member); err != nil {
				return err
			}
			return s.auditProposal(tx, member, consts.ActionTypeRevokeRejectProposal, actor, "撤回拒绝", member.ID, false)
		})
		if err != nil {
			return nil, err
		}
		return &dto.RevokeProposalResp{Resp: dto.Success(), ProposalID: req.ProposalID, ProposalIDs: []string{req.ProposalID}}, nil
	}
	if p.DecisionBatchID == "" {
		return s.legacyRevokeProposal(ctx, req)
	}
	if req.ActionType != consts.RevokeActionApprove {
		return nil, errorx.New(errno.ErrRevokeActionTypeInvalid)
	}
	var batch model.ProposalDecision
	users := map[string]bool{}
	deletedTeachers := []*model.Teacher{}
	err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
		users = map[string]bool{}
		deletedTeachers = []*model.Teacher{}
		current, err := s.ProposalRepo.FindByID(tx, req.ProposalID)
		if err != nil {
			return err
		}
		if current == nil || current.Status != mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved) {
			return errorx.New(errno.ErrProposalStatusNotApproved)
		}
		if err = s.ProposalRepo.Database().Collection(decisionCollection).FindOne(tx, bson.M{"_id": current.DecisionBatchID, "revoked": bson.M{"$ne": true}}).Decode(&batch); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return errorx.New(errno.ErrProposalStatusNotApproved)
			}
			return err
		}
		now := time.Now().UTC()
		if !now.Before(batch.CreatedAt.Add(approvalRevokeWindow)) {
			return errorx.New(errno.ErrProposalRevokeTimeLimitExceeded)
		}
		if batch.AfterCourse != nil {
			collection := s.ProposalRepo.Database().Collection(repo.CourseCollectionName)
			filter := bson.M{"_id": batch.TargetID, "decisionBatchId": batch.ID, "deleted": bson.M{"$ne": true}}
			if batch.BeforeCourse != nil {
				restored := copyAs[model.Course](batch.BeforeCourse)
				restored.UpdatedAt = now
				result, err := collection.ReplaceOne(tx, filter, restored)
				if err != nil {
					return err
				}
				if result.MatchedCount != 1 {
					return errorx.New(errno.ErrCourseModifiedCannotRevoke)
				}
			} else {
				result, err := collection.UpdateOne(tx, filter, bson.M{"$set": bson.M{"deleted": true, "deletedAt": now, "updatedAt": now}})
				if err != nil {
					return err
				}
				if result.MatchedCount != 1 {
					return errorx.New(errno.ErrCourseModifiedCannotRevoke)
				}
				ids, err := s.CommentRepo.SoftDeleteByCourseID(tx, batch.TargetID)
				if err != nil {
					return err
				}
				if err = s.LikeRepo.DeleteByTargets(tx, ids, mapping.Data.GetLikeTargetTypeIDByName(consts.LikeTargetTypeComment)); err != nil {
					return err
				}
			}
		} else {
			teacher := copyAs[model.Teacher](batch.BeforeTeacher)
			teacher.UpdatedAt = now
			result, err := s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).ReplaceOne(tx, bson.M{"_id": batch.TargetID, "decisionBatchId": batch.ID}, teacher)
			if err != nil {
				return err
			}
			if result.MatchedCount != 1 {
				return errorx.New(errno.ErrCourseModifiedCannotRevoke)
			}
		}
		for _, id := range batch.ProposalIDs {
			member, err := s.ProposalRepo.FindByID(tx, id)
			if err != nil {
				return err
			}
			if member == nil || member.DecisionBatchID != batch.ID || member.Status != mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved) {
				return errorx.New(errno.ErrProposalStatusNotApproved)
			}
			if err = s.UserRepo.IncrementContribution(tx, member.UserID, -member.Contribution); err != nil {
				return err
			}
			users[member.UserID] = true
			member.Status = pendingStatus()
			member.UpdatedAt = now
			member.Contribution = 0
			member.Final = nil
			member.FinalCourse = nil
			member.DecisionBatchID = ""
			if member.EffectiveType() == model.ProposalCreateCourse {
				member.TargetID = ""
			}
			if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": id}, member); err != nil {
				return err
			}
			// Retain the revoked decision ID in the immutable event, not on pending data.
			snapshot := copyAs[model.Proposal](member)
			snapshot.DecisionBatchID = batch.ID
			if err = s.auditProposal(tx, snapshot, consts.ActionTypeRevokeApproveProposal, actor, "整批撤回审批", req.ProposalID, id != req.ProposalID); err != nil {
				return err
			}
		}
		for _, id := range batch.CreatedTeacherIDs {
			count, err := s.ProposalRepo.Database().Collection(repo.CourseCollectionName).CountDocuments(tx, bson.M{"teacherIds": id, "deleted": bson.M{"$ne": true}})
			if err != nil {
				return err
			}
			referenced, err := s.ProposalRepo.IsTeacherReferenced(tx, id)
			if err != nil {
				return err
			}
			if count > 0 || referenced {
				continue
			}
			var teacher model.Teacher
			err = s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).FindOne(tx, bson.M{"_id": id, "decisionBatchId": batch.ID}).Decode(&teacher)
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			if err != nil {
				return err
			}
			if _, err = s.ProposalRepo.Database().Collection(repo.TeacherCollectionName).DeleteOne(tx, bson.M{"_id": id, "decisionBatchId": batch.ID}); err != nil {
				return err
			}
			deletedTeachers = append(deletedTeachers, &teacher)
		}
		batch.Revoked = true
		batch.RevokedAt = now
		_, err = s.ProposalRepo.Database().Collection(decisionCollection).ReplaceOne(tx, bson.M{"_id": batch.ID}, &batch)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, teacher := range deletedTeachers {
		if err = s.TeacherRepo.InvalidateDeleted(ctx, teacher); err != nil {
			logs.CtxWarnf(ctx, "teacher invalidation: %v", err)
		}
	}
	if batch.AfterCourse != nil && batch.BeforeCourse == nil && s.CommentCache != nil {
		if err := s.CommentCache.DeleteCount(ctx); err != nil {
			logs.CtxWarnf(ctx, "workflow comment count invalidation failed: %v", err)
		}
	}
	s.afterWorkflow(ctx, users)
	return &dto.RevokeProposalResp{Resp: dto.Success(), ProposalID: req.ProposalID, ProposalIDs: batch.ProposalIDs, DecisionBatchID: batch.ID, TargetID: batch.TargetID}, nil
}

// DeleteProposal hides an author's pending or rejected proposal atomically with
// its immutable audit event, sharing the approval serialization guard.
func (s *ProposalService) DeleteProposal(ctx context.Context, req *dto.DeleteProposalReq) (*dto.DeleteProposalResp, error) {
	actor, err := s.actor(ctx, false)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = s.workflowTransaction(ctx, func(tx mongo.SessionContext) error {
		p, err := s.ProposalRepo.FindByID(tx, req.ProposalID)
		if err != nil {
			return err
		}
		if p == nil {
			return errorx.New(errno.ErrProposalNotFound)
		}
		if p.UserID != actor {
			return errorx.New(errno.ErrUserNotOwner)
		}
		if p.Status == mapping.Data.GetProposalStatusIDByName(consts.ProposalStatusApproved) {
			return errorx.New(errno.ErrProposalCannotDeleteApproved)
		}
		p.Deleted, p.DeletedAt, p.UpdatedAt = true, now, now
		if _, err = s.ProposalRepo.Database().Collection(repo.ProposalCollectionName).ReplaceOne(tx, bson.M{"_id": p.ID}, p); err != nil {
			return err
		}
		return s.auditProposal(tx, p, consts.ActionTypeDeleteProposal, actor, "删除提案", p.ID, false)
	})
	if err != nil {
		return nil, err
	}
	return &dto.DeleteProposalResp{Resp: dto.Success(), ProposalID: req.ProposalID, DeletedAt: now, OperatorID: actor, Deleted: true}, nil
}

func teacherDepartmentName(id int32) string {
	if id == 0 {
		return ""
	}
	return mapping.Data.GetDepartmentNameByID(id)
}
