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
	"math"
	"os"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"go.mongodb.org/mongo-driver/bson"
)

func exerciseWorkflowResilience(t *testing.T, ctx context.Context, s *ProposalService, feedback *FeedbackService, cfg *config.Config, prefix string) {
	db := s.ProposalRepo.Database()
	as := func(role string) context.Context { return context.WithValue(ctx, consts.CtxUserID, prefix+role) }
	input := func(name string) *dto.ProposalCourseVO {
		return &dto.ProposalCourseVO{Name: name, Code: "RESILIENCE", Category: "测试分类", Department: "测试学院", Campuses: []string{"普陀校区"}, Teachers: []*dto.TeacherVO{{Name: name + "教师"}}}
	}
	count := func(collection string, filter bson.M) int64 {
		t.Helper()
		n, err := db.Collection(collection).CountDocuments(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	create := func(role string) string {
		t.Helper()
		if _, err := db.Collection("user").InsertOne(ctx, &model.User{ID: prefix + role, Username: role, Contribution: 500}); err != nil {
			t.Fatal(err)
		}
		p, err := s.CreateProposal(as(role), &dto.CreateProposalReq{Course: input(role), ShowUsername: true})
		if err != nil {
			t.Fatal(err)
		}
		return p.ProposalID
	}
	preview := func(id string) *dto.ToggleProposalReq {
		t.Helper()
		req := &dto.ToggleProposalReq{ProposalID: id}
		p, err := s.PreviewApproval(as("admin"), req)
		if err != nil || !p.CanApprove {
			t.Fatal(p, err)
		}
		req.PreviewToken = p.PreviewToken
		return req
	}

	t.Run("injected transaction and commit faults settle only once", func(t *testing.T) {
		if os.Getenv("MEOWPICK_TEST_FAILPOINTS") != "1" {
			t.Skip("explicit isolated MongoDB enableTestCommands opt-in required")
		}
		admin := db.Client().Database("admin")
		fault := func(data bson.M, fn func()) {
			t.Helper()
			data["appName"] = cfg.Mongo.DB
			var before, after struct {
				Count int64 `bson:"count"`
			}
			if err := admin.RunCommand(ctx, bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: bson.M{"times": 1}}, {Key: "data", Value: data}}).Decode(&before); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := admin.RunCommand(context.Background(), bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: "off"}}).Decode(&after); err != nil {
					t.Error(err)
				} else if after.Count-before.Count != 1 {
					t.Errorf("fault did not activate exactly once: %d", after.Count-before.Count)
				}
			}()
			fn()
		}
		for i, data := range []bson.M{
			{"failCommands": []string{"insert"}, "errorCode": 112, "errorLabels": []string{"TransientTransactionError"}},
			{"failCommands": []string{"commitTransaction"}, "errorCode": 91, "errorLabels": []string{"UnknownTransactionCommitResult"}},
			{"failCommands": []string{"commitTransaction"}, "closeConnection": true},
		} {
			role := fmt.Sprintf("resilience-retry-%d", i)
			id := create(role)
			req := preview(id)
			var batch *dto.ToggleProposalResp
			fault(data, func() {
				var err error
				batch, err = s.ApproveProposal(as("admin"), req)
				if err != nil {
					t.Fatal(err)
				}
			})
			p, _ := s.ProposalRepo.FindByID(ctx, id)
			var u model.User
			if err := db.Collection("user").FindOne(ctx, bson.M{"_id": prefix + role}).Decode(&u); err != nil {
				t.Fatal(err)
			}
			if p.Status != 2 || p.Contribution <= 0 || u.Contribution != 500+p.Contribution || count("course", bson.M{"name": role}) != 1 || count("teacher", bson.M{"name": role + "教师"}) != 1 || count(decisionCollection, bson.M{"targetId": batch.TargetID}) != 1 {
				t.Fatal("retry duplicated settlement", p, u)
			}
			// Simulate a successful response being lost and the same request being sent again.
			logs := count("changelog", bson.M{})
			_, err := s.ApproveProposal(as("admin"), req)
			assertWorkflowCode(t, err, errno.ErrProposalAlreadyProcessed)
			if count("changelog", bson.M{}) != logs || count(decisionCollection, bson.M{"targetId": batch.TargetID}) != 1 {
				t.Fatal("replayed request wrote data")
			}
		}
		role := "resilience-permanent"
		id := create(role)
		req := preview(id)
		beforeLogs := count("changelog", bson.M{})
		beforeDecisions := count(decisionCollection, bson.M{})
		fault(bson.M{"failCommands": []string{"commitTransaction"}, "errorCode": 2, "errorLabels": []string{}}, func() {
			if _, err := s.ApproveProposal(as("admin"), req); err == nil {
				t.Fatal("permanent commit failure succeeded")
			}
		})
		p, _ := s.ProposalRepo.FindByID(ctx, id)
		if p.Status != 1 || p.Contribution != 0 || count("course", bson.M{"name": role}) != 0 || count("teacher", bson.M{"name": role + "教师"}) != 0 {
			t.Fatal("failed commit left partial state", p)
		}
		var unchanged model.User
		if err := db.Collection("user").FindOne(ctx, bson.M{"_id": prefix + role}).Decode(&unchanged); err != nil {
			t.Fatal(err)
		}
		if unchanged.Contribution != 500 || count("changelog", bson.M{}) != beforeLogs || count(decisionCollection, bson.M{}) != beforeDecisions {
			t.Fatal("failed commit wrote settlement or audit")
		}
		if _, err := s.ApproveProposal(as("admin"), req); err != nil {
			t.Fatal("retry after recovery failed", err)
		}
		// Context cancellation must not create proposals, entities, or audit records.
		canceled, cancel := context.WithCancel(as(role))
		cancel()
		n := count("proposal", bson.M{})
		logs := count("changelog", bson.M{})
		_, err := s.CreateProposal(canceled, &dto.CreateProposalReq{Course: input("取消请求")})
		if err == nil || count("proposal", bson.M{}) != n || count("changelog", bson.M{}) != logs {
			t.Fatal("canceled request wrote data", err)
		}
	})
	t.Run("thousand same-name pending proposals and stable pages", func(t *testing.T) {
		const total, exact = 1000, 100
		stamp := time.Now().UTC()
		proposals := make([]any, 0, total)
		users := make([]any, 0, exact)
		for i := 0; i < total; i++ {
			id := fmt.Sprintf("%sbulk-proposal-%04d", prefix, i)
			role := fmt.Sprintf("%sbulk-author-%04d", prefix, i)
			c := copyAs[model.ProposalCourse](input("大量同名课程"))
			if i >= exact {
				c.Code = fmt.Sprintf("OTHER-%04d", i)
			}
			proposals = append(proposals, &model.Proposal{ID: id, Type: model.ProposalCreateCourse, UserID: role, Course: c, Status: pendingStatus(), ShowUsername: true, CreatedAt: stamp, UpdatedAt: stamp})
			if i < exact {
				users = append(users, &model.User{ID: role, Username: role, Contribution: 500})
			}
		}
		if _, err := db.Collection("user").InsertMany(ctx, users); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Collection("proposal").InsertMany(ctx, proposals); err != nil {
			t.Fatal(err)
		}
		defer db.Collection("proposal").DeleteMany(context.Background(), bson.M{"_id": bson.M{"$regex": "^" + prefix + "bulk-proposal-"}})
		likedID := fmt.Sprintf("%sbulk-proposal-%04d", prefix, 0)
		if _, err := db.Collection("like").InsertOne(ctx, &model.Like{ID: prefix + "bulk-like", UserID: prefix + "admin", TargetID: likedID, TargetType: mapping.Data.GetLikeTargetTypeIDByName(consts.LikeTargetTypeProposal), Active: true}); err != nil {
			t.Fatal(err)
		}
		// Traverse equal-timestamp records at two page sizes. No duplicate or missing IDs.
		for _, size := range []int64{37, 100} {
			seen := map[string]bool{}
			for page := int64(1); ; page++ {
				rows, _, err := s.ProposalRepo.FindManyByStatus(ctx, &dto.PageParam{Page: page, PageSize: size}, pendingStatus())
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) == 0 {
					break
				}
				for _, p := range rows {
					if seen[p.ID] {
						t.Fatalf("pagination repeated ID at page %d size %d: %s", page, size, p.ID)
					}
					seen[p.ID] = true
				}
			}
			for i := 0; i < total; i++ {
				if !seen[fmt.Sprintf("%sbulk-proposal-%04d", prefix, i)] {
					t.Fatal("pagination skipped a seeded proposal", i)
				}
			}
		}
		req := &dto.ToggleProposalReq{ProposalID: fmt.Sprintf("%sbulk-proposal-%04d", prefix, 0)}
		started := time.Now()
		p, err := s.PreviewApproval(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("1000 pending preview: %s", time.Since(started))
		if len(p.Members) != exact || len(p.Suspicious) != total-exact {
			t.Fatal("large duplicate grouping incorrect", len(p.Members), len(p.Suspicious))
		}
		for _, member := range p.Members {
			if member.Proposal.ID == likedID && (!member.Proposal.Like || member.Proposal.LikeCnt != 1) {
				t.Fatal("preview lost like data", member)
			}
		}
		req.PreviewToken = p.PreviewToken
		started = time.Now()
		approved, err := s.ApproveProposal(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("100-member approval: %s", time.Since(started))
		if len(approved.ProposalIDs) != exact || count("course", bson.M{"name": "大量同名课程"}) != 1 || count("teacher", bson.M{"name": "大量同名课程教师"}) != 1 {
			t.Fatal("large group wrote duplicate entities", approved)
		}
		for i := 0; i < exact; i++ {
			var u model.User
			id := fmt.Sprintf("%sbulk-author-%04d", prefix, i)
			if err := db.Collection("user").FindOne(ctx, bson.M{"_id": id}).Decode(&u); err != nil {
				t.Fatal(err)
			}
			if u.Contribution != 506 {
				t.Fatal("large group contribution incorrect", id, u.Contribution)
			}
		}
		contributors, err := s.CourseAssembler.EntityContributors(ctx, "course", approved.TargetID, "")
		if err != nil || len(contributors) != exact {
			t.Fatal("large contributor list incomplete", len(contributors), err)
		}
	})
	t.Run("equal-time feedback messages and history pagination", func(t *testing.T) {
		const total = 201
		role := "resilience-pages"
		id := create(role)
		approved, err := s.ApproveProposal(as("admin"), preview(id))
		if err != nil {
			t.Fatal(err)
		}
		stamp := time.Now().UTC()
		feedbacks, messages, decisions := []any{}, []any{}, []any{}
		conversation := prefix + "pages-conversation"
		for i := 0; i < total; i++ {
			fid := fmt.Sprintf("%sfeedback-%04d", prefix, i)
			feedbacks = append(feedbacks, &model.Feedback{ID: fid, UserID: prefix + role, Status: "pending", Category: "other", Sequence: 0, CreatedAt: stamp, UpdatedAt: stamp})
			messages = append(messages, &model.FeedbackMessage{ID: fmt.Sprintf("%smessage-%04d", prefix, i), FeedbackID: conversation, UserID: prefix + role, Role: "author", Sequence: int64(i + 1), Text: "分页.*[]", CreatedAt: stamp})
			decisions = append(decisions, &model.ProposalDecision{ID: fmt.Sprintf("%shistory-%04d", prefix, i), Type: model.ProposalUpdateCourse, TargetID: approved.TargetID, CreatedAt: stamp, ProposalIDs: []string{}})
		}
		for name, values := range map[string][]any{feedbackCollection: feedbacks, feedbackMessageCollection: messages, decisionCollection: decisions} {
			if _, err := db.Collection(name).InsertMany(ctx, values); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.Collection(feedbackCollection).InsertOne(ctx, &model.Feedback{ID: conversation, UserID: prefix + role, Status: "answered", Category: "bug", Sequence: total, CreatedAt: stamp, UpdatedAt: stamp}); err != nil {
			t.Fatal(err)
		}
		for _, kind := range []string{"feedback", "message", "history"} {
			seen := map[string]bool{}
			for page := int64(1); page <= 9; page++ {
				param := &dto.PageParam{Page: page, PageSize: 31}
				ids := []string{}
				switch kind {
				case "feedback":
					r, err := feedback.List(as(role), &dto.ListFeedbackReq{Status: "pending", PageParam: param}, false)
					if err != nil || r.Total != total {
						t.Fatal(r, err)
					}
					for _, v := range r.Feedbacks {
						ids = append(ids, v.ID)
					}
				case "message":
					r, err := feedback.Detail(as(role), conversation, param, false)
					if err != nil || r.Total != total {
						t.Fatal(r, err)
					}
					for _, v := range r.Messages {
						ids = append(ids, v.ID)
					}
				case "history":
					r, err := s.EntityHistory(as(role), &dto.EntityHistoryReq{TargetType: "course", TargetID: approved.TargetID, PageParam: param})
					if err != nil || r.Total != total+1 {
						t.Fatal(r, err)
					}
					for _, v := range r.History {
						ids = append(ids, v.DecisionBatchID)
					}
				}
				for _, id := range ids {
					if seen[id] {
						t.Fatal("duplicate pagination", kind, id)
					}
					seen[id] = true
				}
			}
			want := total
			if kind == "history" {
				want++
			}
			if len(seen) != want {
				t.Fatal("pagination lost records", kind, len(seen), want)
			}
		}
		huge := &dto.PageParam{Page: math.MaxInt64, PageSize: 100}
		empty, err := feedback.Detail(as(role), conversation, huge, false)
		if err != nil || len(empty.Messages) != 0 || empty.Total != total {
			t.Fatal("huge page overflow", empty, err)
		}

		for _, target := range []string{approved.TargetID, "legacy-course"} {
			h, err := s.EntityHistory(as(role), &dto.EntityHistoryReq{TargetType: "course", TargetID: target, PageParam: &dto.PageParam{Page: math.MaxInt64, PageSize: 100}})
			if err != nil || len(h.History) != 0 {
				t.Fatal("huge history page leaked records or failed", h, err)
			}
		}
		emptyList, err := feedback.List(as(role), &dto.ListFeedbackReq{Status: "pending", PageParam: &dto.PageParam{Page: math.MaxInt64, PageSize: 100}}, false)
		if err != nil || emptyList.Total != total || len(emptyList.Feedbacks) != 0 {
			t.Fatal("huge list page failed", emptyList, err)
		}
		searched, err := feedback.List(as("admin"), &dto.ListFeedbackReq{Keyword: ".*[]"}, true)
		if err != nil || searched.Total != 1 {
			t.Fatal("keyword metacharacters treated as regex", searched, err)
		}
	})
}
