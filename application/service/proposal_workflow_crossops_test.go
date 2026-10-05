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
	"sync"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"go.mongodb.org/mongo-driver/bson"
)

func exerciseWorkflowCrossOps(t *testing.T, ctx context.Context, s *ProposalService, cfg *config.Config, prefix string) {
	db := s.ProposalRepo.Database()
	as := func(role string) context.Context { return context.WithValue(ctx, consts.CtxUserID, prefix+role) }
	input := func(name string) *dto.ProposalCourseVO {
		return &dto.ProposalCourseVO{Name: name, Code: "CROSS", Category: "测试分类", Department: "测试学院", Campuses: []string{"普陀校区"}, Teachers: []*dto.TeacherVO{{Name: name + "教师"}}}
	}
	seed := func(role string) {
		t.Helper()
		if _, err := db.Collection("user").InsertOne(ctx, &model.User{ID: prefix + role, Username: role, Contribution: 500}); err != nil {
			t.Fatal(err)
		}
	}
	create := func(role, name string) string {
		t.Helper()
		seed(role)
		p, err := s.CreateProposal(as(role), &dto.CreateProposalReq{Course: input(name), ShowUsername: true})
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
	approve := func(id string) *dto.ToggleProposalResp {
		t.Helper()
		p, err := s.ApproveProposal(as("admin"), preview(id))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	points := func(role string) int64 {
		t.Helper()
		var u model.User
		if err := db.Collection("user").FindOne(ctx, bson.M{"_id": prefix + role}).Decode(&u); err != nil {
			t.Fatal(err)
		}
		return u.Contribution
	}
	race := func(a, b func() error) []error {
		start := make(chan struct{})
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i, fn := range []func() error{a, b} {
			wg.Add(1)
			go func(i int, fn func() error) { defer wg.Done(); <-start; errs[i] = fn() }(i, fn)
		}
		close(start)
		wg.Wait()
		return errs
	}

	t.Run("approve versus reject or author delete", func(t *testing.T) {
		for _, op := range []string{"reject", "delete"} {
			for iteration := 0; iteration < 5; iteration++ {
				role := fmt.Sprintf("cross-%s-%d", op, iteration)
				id := create(role, role)
				req := preview(id)
				errs := race(func() error { _, err := s.ApproveProposal(as("admin"), req); return err }, func() error {
					if op == "delete" {
						_, err := s.DeleteProposal(as(role), &dto.DeleteProposalReq{ProposalID: id})
						return err
					}
					_, err := s.RejectProposal(as("admin2"), &dto.RejectProposalReq{ProposalID: id, Reason: "并发拒绝"})
					return err
				})
				if (errs[0] == nil) == (errs[1] == nil) {
					t.Fatal("expected exactly one winner", errs)
				}
				p, err := s.ProposalRepo.FindByIDIncludeDeleted(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				n, err := db.Collection("course").CountDocuments(ctx, bson.M{"name": role, "deleted": bson.M{"$ne": true}})
				if err != nil {
					t.Fatal(err)
				}
				want := int64(500)
				if errs[0] == nil {
					want += p.Contribution
					if p.Status != 2 || p.Deleted || n != 1 || p.Contribution <= 0 {
						t.Fatal("bad approved state", p, n)
					}
				} else if p.Status == 2 || n != 0 || p.Contribution != 0 {
					t.Fatal("losing approval wrote data", p, n)
				}
				if points(role) != want {
					t.Fatal("wrong points", points(role), want)
				}
			}
		}
	})
	t.Run("revoke versus approving later modification", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			role := fmt.Sprintf("cross-revoke-%d", i)
			id := create(role, role)
			first := approve(id)
			editor := role + "-editor"
			seed(editor)
			edit, err := s.CreateProposal(as(editor), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: first.TargetID, Suggested: &dto.ProposalPatch{Code: textPointer("NEW")}})
			if err != nil {
				t.Fatal(err)
			}
			req := preview(edit.ProposalID)
			errs := race(func() error {
				_, err := s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: id, ActionType: "approve"})
				return err
			}, func() error { _, err := s.ApproveProposal(as("admin2"), req); return err })
			if (errs[0] == nil) == (errs[1] == nil) {
				t.Fatal("expected exactly one winner", errs)
			}
			var c model.Course
			if err = db.Collection("course").FindOne(ctx, bson.M{"_id": first.TargetID}).Decode(&c); err != nil {
				t.Fatal(err)
			}
			p, _ := s.ProposalRepo.FindByID(ctx, id)
			e, _ := s.ProposalRepo.FindByID(ctx, edit.ProposalID)
			if errs[0] == nil {
				if !c.Deleted || p.Status != 1 || e.Status != 1 || points(role) != 500 || points(editor) != 500 {
					t.Fatal("bad revoke winner", c, p, e)
				}
			} else if c.Deleted || c.Code != "NEW" || p.Status != 2 || e.Status != 2 || points(role) != 500+p.Contribution || points(editor) != 500+e.Contribution {
				t.Fatal("bad edit winner", c, p, e)
			}
		}
	})
	t.Run("rejected resubmit shared approve revoke and reapprove", func(t *testing.T) {
		a := create("cross-chain-a", "重新提交链")
		_, err := s.RejectProposal(as("admin"), &dto.RejectProposalReq{ProposalID: a, Reason: "更正后重提"})
		if err != nil {
			t.Fatal(err)
		}
		// A different author cannot replace the rejected proposal.
		seed("cross-chain-outsider")
		_, err = s.ResubmitProposal(as("cross-chain-outsider"), &dto.ResubmitProposalReq{ProposalID: a, Course: input("重新提交链")})
		assertWorkflowCode(t, err, errno.ErrProposalNotFound)
		r, err := s.ResubmitProposal(as("cross-chain-a"), &dto.ResubmitProposalReq{ProposalID: a, Course: input("重新提交链"), ShowUsername: true})
		if err != nil {
			t.Fatal(err)
		}
		old, _ := s.ProposalRepo.FindByIDIncludeDeleted(ctx, a)
		if !old.Deleted || old.Status != 3 || r.PreviousProposalID != a || r.ProposalID == a {
			t.Fatal("resubmit lost original", old, r)
		}
		b := create("cross-chain-b", "重新提交链")
		first := approve(r.ProposalID)
		if len(first.ProposalIDs) != 2 {
			t.Fatal(first)
		}
		if _, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: b, ActionType: "approve"}); err != nil {
			t.Fatal(err)
		}
		if points("cross-chain-a") != 500 || points("cross-chain-b") != 500 {
			t.Fatal("points not restored")
		}
		second := approve(b)
		if second.TargetID != first.TargetID || second.DecisionBatchID == first.DecisionBatchID || len(second.ProposalIDs) != 2 {
			t.Fatal("reapproval duplicated course", first, second)
		}
		for _, role := range []string{"cross-chain-a", "cross-chain-b"} {
			id := r.ProposalID
			if role == "cross-chain-b" {
				id = b
			}
			p, _ := s.ProposalRepo.FindByID(ctx, id)
			if points(role) != 500+p.Contribution || p.Contribution <= 0 {
				t.Fatal("duplicate settlement", p)
			}
		}
		c, err := s.CourseAssembler.EntityContributors(ctx, "course", second.TargetID, "")
		if err != nil || len(c) != 2 {
			t.Fatal("wrong contributors", c, err)
		}
	})
	t.Run("invalid batch selections cannot mutate any member", func(t *testing.T) {
		base := create("cross-batch-base", "批量选择基准")
		formal := approve(create("cross-batch-formal", "批量正式目标"))
		deleted := create("cross-batch-deleted", "删除提案")
		if _, err := s.DeleteProposal(as("cross-batch-deleted"), &dto.DeleteProposalReq{ProposalID: deleted}); err != nil {
			t.Fatal(err)
		}
		seed("cross-batch-editor")
		edit, err := s.CreateProposal(as("cross-batch-editor"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: formal.TargetID, Suggested: &dto.ProposalPatch{Code: textPointer("BATCH")}})
		if err != nil {
			t.Fatal(err)
		}
		otherFormal := approve(create("cross-batch-other", "批量另一目标"))
		seed("cross-batch-editor2")
		edit2, err := s.CreateProposal(as("cross-batch-editor2"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: otherFormal.TargetID, Suggested: &dto.ProposalPatch{Code: textPointer("BATCH")}})
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct {
			primary, extra string
			code           int32
		}{{base, deleted, errno.ErrProposalPreviewStale}, {base, formal.ProposalIDs[0], errno.ErrProposalPreviewStale}, {base, edit.ProposalID, errno.ErrProposalInvalidField}, {edit.ProposalID, edit2.ProposalID, errno.ErrProposalInvalidField}} {
			before, _ := db.Collection("changelog").CountDocuments(ctx, bson.M{})
			req := &dto.ToggleProposalReq{ProposalID: tt.primary, ProposalIDs: []string{tt.extra}, PreviewToken: "forged"}
			_, err := s.PreviewApproval(as("admin"), req)
			assertWorkflowCode(t, err, tt.code)
			_, err = s.ApproveProposal(as("admin"), req)
			assertWorkflowCode(t, err, tt.code)
			p, _ := s.ProposalRepo.FindByID(ctx, tt.primary)
			after, _ := db.Collection("changelog").CountDocuments(ctx, bson.M{})
			if p.Status != 1 || p.Contribution != 0 || p.DecisionBatchID != "" || after != before {
				t.Fatal("invalid selection mutated primary", p)
			}
		}
		// Rejecting a batch with an already approved or deleted member also rolls back.
		for _, extra := range []string{deleted, formal.ProposalIDs[0]} {
			_, err := s.RejectProposal(as("admin"), &dto.RejectProposalReq{ProposalID: base, ProposalIDs: []string{extra}, Reason: "无效整批"})
			if err == nil {
				t.Fatal("invalid rejection succeeded")
			}
			p, _ := s.ProposalRepo.FindByID(ctx, base)
			if p.Status != 1 || p.RejectReason != "" {
				t.Fatal("partial rejection", p)
			}
		}
	})
	t.Run("course revoke cleans comments likes and count cache", func(t *testing.T) {
		a := approve(create("cross-comments", "带评价撤回"))
		b := approve(create("cross-comments-other", "保留评价课程"))
		commentID := prefix + "revoke-comment"
		otherID := prefix + "keep-comment"
		for _, c := range []*model.Comment{{ID: commentID, CourseID: a.TargetID, UserID: prefix + "cross-comments", Tags: []string{"标签"}}, {ID: otherID, CourseID: b.TargetID, UserID: prefix + "cross-comments-other"}} {
			if _, err := db.Collection("comment").InsertOne(ctx, c); err != nil {
				t.Fatal(err)
			}
		}
		likeType := mapping.Data.GetLikeTargetTypeIDByName(consts.LikeTargetTypeComment)
		for _, id := range []string{commentID, otherID} {
			if _, err := db.Collection("like").InsertOne(ctx, &model.Like{ID: id, UserID: prefix + "cross-comments", TargetID: id, TargetType: likeType, Active: true}); err != nil {
				t.Fatal(err)
			}
		}
		cc := cache.NewCommentCache(cfg)
		if err := cc.SetCount(ctx, 2, time.Minute); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: a.ProposalIDs[0], ActionType: "approve"}); err != nil {
			t.Fatal(err)
		}
		c, err := s.CommentRepo.FindByID(ctx, commentID)
		if err != nil || c != nil {
			t.Fatal("revoked comment still public", c, err)
		}
		keep, err := s.CommentRepo.FindByID(ctx, otherID)
		if err != nil || keep == nil {
			t.Fatal("unrelated comment removed", err)
		}
		n, err := db.Collection("like").CountDocuments(ctx, bson.M{"targetId": commentID})
		if err != nil || n != 0 {
			t.Fatal("dangling likes", n, err)
		}
		n, err = db.Collection("like").CountDocuments(ctx, bson.M{"targetId": otherID})
		if err != nil || n != 1 {
			t.Fatal("unrelated likes removed", n, err)
		}
		if _, hit, err := cc.GetCount(ctx); err != nil || hit {
			t.Fatal("comment count cache stale", hit, err)
		}
	})
}
