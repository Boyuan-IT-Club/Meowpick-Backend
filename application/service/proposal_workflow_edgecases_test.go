// Copyright 2026 Boyuan-IT-Club
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
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func exerciseWorkflowEdges(t *testing.T, ctx context.Context, s *ProposalService, feedback *FeedbackService, db *mongo.Database, target, prefix string) {
	t.Helper()
	as := func(role string) context.Context { return context.WithValue(ctx, consts.CtxUserID, prefix+role) }
	seed := func(t *testing.T, role string, points int64, admin bool) {
		t.Helper()
		if _, err := db.Collection("user").InsertOne(ctx, &model.User{ID: prefix + role, Username: role, Contribution: points, Admin: admin}); err != nil {
			t.Fatal(err)
		}
	}
	input := func(name string) *dto.CreateProposalReq {
		return &dto.CreateProposalReq{ShowUsername: true, Course: &dto.ProposalCourseVO{Name: name, Code: "EDGE", Category: "测试分类", Department: "测试学院", Campuses: []string{"普陀校区"}, Teachers: []*dto.TeacherVO{{Name: name + "教师"}}}}
	}
	count := func(t *testing.T, collection string, filter bson.M) int64 {
		t.Helper()
		n, err := db.Collection(collection).CountDocuments(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	create := func(t *testing.T, role, name string) string {
		t.Helper()
		result, err := s.CreateProposal(as(role), input(name))
		if err != nil {
			t.Fatal(err)
		}
		return result.ProposalID
	}

	t.Run("invalid values never write proposals or audit", func(t *testing.T) {
		seed(t, "edge-invalid", 500, false)
		cases := []struct {
			name   string
			mutate func(*dto.CreateProposalReq)
			code   int32
		}{
			{"nil course", func(p *dto.CreateProposalReq) { p.Course = nil }, errno.ErrProposalInvalidField},
			{"whitespace name", func(p *dto.CreateProposalReq) { p.Course.Name = " \t\u3000\n" }, errno.ErrProposalInvalidField},
			{"empty campus", func(p *dto.CreateProposalReq) { p.Course.Campuses = []string{} }, errno.ErrProposalInvalidField},
			{"unknown campus", func(p *dto.CreateProposalReq) { p.Course.Campuses = []string{"不存在的校区"} }, errno.ErrProposalInvalidCampus},
			{"null teacher", func(p *dto.CreateProposalReq) { p.Course.Teachers = []*dto.TeacherVO{nil} }, errno.ErrProposalInvalidField},
			{"duplicate new teacher", func(p *dto.CreateProposalReq) {
				p.Course.Teachers = []*dto.TeacherVO{{Name: "重复教师", Title: "讲师"}, {Name: " 重复教师 ", Title: "教授"}}
			}, errno.ErrProposalInvalidField},
			{"unknown teacher id", func(p *dto.CreateProposalReq) {
				p.Course.Teachers = []*dto.TeacherVO{{ID: "missing-teacher", Name: "教师"}}
			}, errno.ErrProposalTargetNotFound},
			{"invalid type", func(p *dto.CreateProposalReq) { p.Type = "delete_course" }, errno.ErrProposalInvalidField},
			{"create with patch", func(p *dto.CreateProposalReq) { p.Suggested = &dto.ProposalPatch{Name: textPointer("覆盖")} }, errno.ErrProposalInvalidField},
			{"update without target", func(p *dto.CreateProposalReq) {
				p.Type = model.ProposalUpdateCourse
				p.Course = nil
				p.Suggested = &dto.ProposalPatch{Code: textPointer("X")}
			}, errno.ErrProposalInvalidField},
			{"update missing target", func(p *dto.CreateProposalReq) {
				p.Type = model.ProposalUpdateCourse
				p.Course = nil
				p.TargetID = "missing-course"
				p.Suggested = &dto.ProposalPatch{Code: textPointer("X")}
			}, errno.ErrProposalTargetNotFound},
			{"course with teacher field", func(p *dto.CreateProposalReq) {
				p.Type = model.ProposalUpdateCourse
				p.Course = nil
				p.TargetID = target
				p.Suggested = &dto.ProposalPatch{Title: textPointer("教授")}
			}, errno.ErrProposalInvalidField},
			{"empty patch", func(p *dto.CreateProposalReq) {
				p.Type = model.ProposalUpdateCourse
				p.Course = nil
				p.TargetID = target
				p.Suggested = &dto.ProposalPatch{}
			}, errno.ErrProposalNoChanges},
		}
		before := count(t, "proposal", bson.M{})
		logsBefore := count(t, "changelog", bson.M{})
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				req := input("坏数据")
				tt.mutate(req)
				_, err := s.CreateProposal(as("edge-invalid"), req)
				assertWorkflowCode(t, err, tt.code)
			})
		}
		if count(t, "proposal", bson.M{}) != before || count(t, "changelog", bson.M{}) != logsBefore {
			t.Fatal("invalid submissions left partial data")
		}
	})
	t.Run("same author concurrent duplicates", func(t *testing.T) {
		seed(t, "edge-duplicate", 500, false)
		var wg sync.WaitGroup
		results := make(chan error, 6)
		for i := 0; i < 6; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := s.CreateProposal(as("edge-duplicate"), input("并发同一个建议"))
				results <- err
			}()
		}
		wg.Wait()
		close(results)
		success := 0
		for err := range results {
			if err == nil {
				success++
			} else {
				assertWorkflowCode(t, err, errno.ErrProposalCourseFoundInProposals)
			}
		}
		if success != 1 || count(t, "proposal", bson.M{"userId": prefix + "edge-duplicate"}) != 1 {
			t.Fatal("duplicate race wrote multiple proposals", success)
		}
	})
	t.Run("last quota slot is shared and deletion cannot reset it", func(t *testing.T) {
		seed(t, "edge-quota", 0, false)
		for i := 0; i < 4; i++ {
			if _, err := db.Collection("proposal").InsertOne(ctx, &model.Proposal{ID: fmt.Sprintf("%squota-%d", prefix, i), UserID: prefix + "edge-quota", Status: 3, CreatedAt: time.Now()}); err != nil {
				t.Fatal(err)
			}
		}
		var wg sync.WaitGroup
		results := make(chan error, 2)
		ids := make(chan string, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				resp, err := s.CreateProposal(as("edge-quota"), input(fmt.Sprintf("最后额度%d", i)))
				if err == nil {
					ids <- resp.ProposalID
				}
				results <- err
			}(i)
		}
		wg.Wait()
		close(results)
		close(ids)
		success := 0
		for err := range results {
			if err == nil {
				success++
			} else {
				assertWorkflowCode(t, err, errno.ErrDailyProposalLimitReached)
			}
		}
		if success != 1 {
			t.Fatal("quota overspent", success)
		}
		_, err := s.CreateProposal(as("edge-quota"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: target, Suggested: &dto.ProposalPatch{Code: textPointer("额度后修改")}})
		assertWorkflowCode(t, err, errno.ErrDailyProposalLimitReached)
		for id := range ids {
			if _, err := s.DeleteProposal(as("edge-quota"), &dto.DeleteProposalReq{ProposalID: id}); err != nil {
				t.Fatal(err)
			}
		}
		_, err = s.CreateProposal(as("edge-quota"), input("删除不能绕过额度"))
		assertWorkflowCode(t, err, errno.ErrDailyProposalLimitReached)
	})
	t.Run("batch reject rolls back earlier members on missing last id", func(t *testing.T) {
		seed(t, "edge-reject", 500, false)
		id := create(t, "edge-reject", "批拒原子性")
		_, err := s.RejectProposal(as("admin"), &dto.RejectProposalReq{ProposalID: id, ProposalIDs: []string{id, "zz-missing"}, Reason: "整批拒绝"})
		assertWorkflowCode(t, err, errno.ErrProposalNotFound)
		p, err := s.ProposalRepo.FindByID(ctx, id)
		if err != nil || p.Status != pendingStatus() || p.RejectReason != "" {
			t.Fatal("partial rejection persisted", p, err)
		}
		if count(t, "changelog", bson.M{"proposalId": id, "action": consts.ActionTypeRejectProposal}) != 0 {
			t.Fatal("rolled-back audit persisted")
		}
	})
	t.Run("approval failure after entity write rolls everything back", func(t *testing.T) {
		seed(t, "edge-settle-a", 500, false)
		seed(t, "edge-settle-b", 500, false)
		id := create(t, "edge-settle-a", "积分中途失败")
		other := create(t, "edge-settle-b", "积分中途失败")
		if _, err := db.Collection("user").DeleteOne(ctx, bson.M{"_id": prefix + "edge-settle-b"}); err != nil {
			t.Fatal(err)
		}
		req := &dto.ToggleProposalReq{ProposalID: id}
		preview, err := s.PreviewApproval(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		req.PreviewToken = preview.PreviewToken
		if _, err = s.ApproveProposal(as("admin"), req); err == nil {
			t.Fatal("missing author settlement unexpectedly succeeded")
		}
		for _, member := range []string{id, other} {
			p, err := s.ProposalRepo.FindByID(ctx, member)
			if err != nil || p.Status != pendingStatus() || p.Contribution != 0 || p.DecisionBatchID != "" {
				t.Fatal("partial approval persisted", p, err)
			}
		}
		for _, collection := range []string{"course", "teacher"} {
			if count(t, collection, bson.M{"name": bson.M{"$in": []string{"积分中途失败", "积分中途失败教师"}}}) != 0 {
				t.Fatal("rolled-back entity persisted", collection)
			}
		}
		user, err := s.UserRepo.FindByID(ctx, prefix+"edge-settle-a")
		if err != nil || user.Contribution != 500 {
			t.Fatal("first author's rolled-back points persisted", user, err)
		}
		if count(t, "proposal_decision", bson.M{"proposalIds": id}) != 0 || count(t, "changelog", bson.M{"proposalId": id, "action": consts.ActionTypeApproveProposal}) != 0 {
			t.Fatal("rolled-back decision/audit persisted")
		}
	})
	t.Run("feedback day boundary and invalid read cursors", func(t *testing.T) {
		seed(t, "edge-feedback-day", 500, false)
		local := time.Now().In(time.FixedZone("UTC+8", 8*3600))
		start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
		for i := 0; i < 10; i++ {
			created := start
			if i == 9 {
				created = start.Add(-time.Millisecond)
			}
			if _, err := db.Collection("feedback").InsertOne(ctx, &model.Feedback{ID: fmt.Sprintf("%sday-%d", prefix, i), UserID: prefix + "edge-feedback-day", Category: "other", Status: "pending", CreatedAt: created}); err != nil {
				t.Fatal(err)
			}
		}
		entry, err := feedback.Create(as("edge-feedback-day"), &dto.CreateFeedbackReq{Text: "第十条"})
		if err != nil {
			t.Fatal("yesterday counted against today", err)
		}
		_, err = feedback.Create(as("edge-feedback-day"), &dto.CreateFeedbackReq{Text: "第十一条"})
		assertWorkflowCode(t, err, errno.ErrFeedbackDailyLimit)
		for _, sequence := range []int64{-1, 2, 1 << 62} {
			_, err := feedback.Read(as("edge-feedback-day"), entry.Feedback.ID, &dto.FeedbackReadReq{Sequence: sequence}, false)
			assertWorkflowCode(t, err, errno.ErrFeedbackInvalid)
		}
	})
	t.Run("concurrent feedback rate limit has no partial writes", func(t *testing.T) {
		seed(t, "edge-feedback-flood", 500, false)
		var wg sync.WaitGroup
		results := make(chan error, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := feedback.Create(as("edge-feedback-flood"), &dto.CreateFeedbackReq{Text: fmt.Sprintf("并发反馈%d", i)})
				results <- err
			}(i)
		}
		wg.Wait()
		close(results)
		success := 0
		for err := range results {
			if err == nil {
				success++
			} else {
				assertWorkflowCode(t, err, errno.ErrFeedbackRateLimit)
			}
		}
		if success != 5 || count(t, "feedback", bson.M{"userId": prefix + "edge-feedback-flood"}) != 5 || count(t, "feedback_message", bson.M{"userId": prefix + "edge-feedback-flood"}) != 5 {
			t.Fatal("rate limit overspent or left partial messages", success)
		}
	})
	t.Run("concurrent opposite replies preserve sequence and read cursor", func(t *testing.T) {
		seed(t, "edge-reply-author", 500, false)
		seed(t, "edge-reply-admin", 500, true)
		entry, err := feedback.Create(as("edge-reply-author"), &dto.CreateFeedbackReq{Text: strings.Repeat("😀", 2000)})
		if err != nil {
			t.Fatal(err)
		}
		_, err = feedback.Send(as("edge-reply-author"), entry.Feedback.ID, &dto.FeedbackMessageReq{Text: strings.Repeat("😀", 2001)}, false)
		assertWorkflowCode(t, err, errno.ErrFeedbackInvalid)
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for _, admin := range []bool{false, true} {
			wg.Add(1)
			go func(admin bool) {
				defer wg.Done()
				role := "edge-reply-author"
				if admin {
					role = "edge-reply-admin"
				}
				_, err := feedback.Send(as(role), entry.Feedback.ID, &dto.FeedbackMessageReq{Text: "同时追加"}, admin)
				results <- err
			}(admin)
		}
		wg.Wait()
		close(results)
		for err := range results {
			if err != nil {
				t.Fatal(err)
			}
		}
		detail, err := feedback.Detail(as("edge-reply-admin"), entry.Feedback.ID, nil, true)
		if err != nil || detail.Total != 3 || len(detail.Messages) != 3 {
			t.Fatal("concurrent reply lost a message", detail, err)
		}
		for i, message := range detail.Messages {
			if message.Sequence != int64(3-i) {
				t.Fatal("duplicated or skipped sequence", detail.Messages)
			}
		}
		wantStatus := "pending"
		if detail.Messages[0].Role == "admin" {
			wantStatus = "answered"
		}
		if detail.Feedback.Status != wantStatus {
			t.Fatal("status differs from final message role")
		}
		for _, sequence := range []int64{2, 1} {
			if _, err := feedback.Read(as("edge-reply-admin"), entry.Feedback.ID, &dto.FeedbackReadReq{Sequence: sequence}, true); err != nil {
				t.Fatal(err)
			}
		}
		detail, err = feedback.Detail(as("edge-reply-admin"), entry.Feedback.ID, nil, true)
		wantUnread := int64(0)
		if detail.Messages[0].Role == "author" {
			wantUnread = 1
		}
		if err != nil || detail.Feedback.UnreadCount != wantUnread {
			t.Fatal("old read moved cursor backward", detail, err)
		}
	})
	t.Run("expired approval cannot partially revoke", func(t *testing.T) {
		seed(t, "edge-expired", 500, false)
		id := create(t, "edge-expired", "撤回时间边界")
		req := &dto.ToggleProposalReq{ProposalID: id}
		preview, err := s.PreviewApproval(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		req.PreviewToken = preview.PreviewToken
		approved, err := s.ApproveProposal(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Collection("proposal_decision").UpdateOne(ctx, bson.M{"_id": approved.DecisionBatchID}, bson.M{"$set": bson.M{"createdAt": time.Now().Add(-24*time.Hour - time.Second)}}); err != nil {
			t.Fatal(err)
		}
		_, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: id, ActionType: "approve"})
		assertWorkflowCode(t, err, errno.ErrProposalRevokeTimeLimitExceeded)
		p, err := s.ProposalRepo.FindByID(ctx, id)
		if err != nil || p.Status != 2 || p.Contribution != 6 {
			t.Fatal("expired revoke changed proposal/points", p, err)
		}
		course, err := s.CourseRepo.FindByID(ctx, approved.TargetID)
		if err != nil || course == nil {
			t.Fatal("expired revoke deleted entity", err)
		}
	})
}
