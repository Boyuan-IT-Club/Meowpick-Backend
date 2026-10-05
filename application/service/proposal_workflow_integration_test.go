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
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	storecache "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func assertWorkflowCode(t *testing.T, err error, want int32) {
	t.Helper()
	var status errorx.StatusError
	if !errors.As(err, &status) || status.Code() != want {
		t.Fatalf("error=%v, want code %d", err, want)
	}
}
func TestProposalWorkflowIntegration(t *testing.T) {
	uri, redisAddr := os.Getenv("MEOWPICK_TEST_MONGO_URI"), os.Getenv("MEOWPICK_TEST_REDIS_ADDR")
	if uri == "" || redisAddr == "" {
		t.Skip("isolated replica-set MongoDB and Redis required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	cfg := &config.Config{Redis: &redis.RedisConf{Host: redisAddr, Type: "node"}}
	cfg.Mongo.URL = uri
	cfg.Mongo.DB = fmt.Sprintf("proposal_workflow_test_%d", time.Now().UnixNano())
	testURI, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	query := testURI.Query()
	query.Set("appName", cfg.Mongo.DB)
	testURI.RawQuery = query.Encode()
	cfg.Mongo.URL = testURI.String()
	cfg.Cache = storecache.CacheConf{{RedisConf: *cfg.Redis, Weight: 100}}
	db := client.Database(cfg.Mongo.DB)
	defer db.Drop(context.Background())
	proposals, err := repo.NewProposalRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	users, err := repo.NewUserRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	courses, err := repo.NewCourseRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	teachers := repo.NewTeacherRepo(cfg)
	comments, err := repo.NewCommentRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	likes := repo.NewLikeRepo(cfg)
	logs := repo.NewChangeLogRepo(cfg)
	mappings, err := repo.NewMappingRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Collection("mapping").InsertOne(ctx, &model.Mapping{Type: model.MappingTypeCampus, Name: "普陀校区", Code: 1, Canonical: true}); err != nil {
		t.Fatal(err)
	}
	if err = mapping.Data.InitWithDependencies(ctx, &mapping.MappingDependencies{MappingRepo: mappings, MappingCache: cache.NewMappingCache(cfg)}); err != nil {
		t.Fatal(err)
	}
	ca := &assembler.CourseAssembler{CourseRepo: courses, TeacherRepo: teachers, CommentRepo: comments, ProposalRepo: proposals, UserRepo: users}
	pa := &assembler.ProposalAssembler{CourseAssembler: ca, LikeRepo: likes}
	s := &ProposalService{ProposalRepo: proposals, UserRepo: users, CourseRepo: courses, TeacherRepo: teachers, CommentRepo: comments, CommentCache: cache.NewCommentCache(cfg), LikeRepo: likes, CourseAssembler: ca, ProposalAssembler: pa, ChangeLogRepo: logs, MappingRepo: mappings, ChangeLogService: &ChangeLogService{ChangeLogRepo: logs}}
	feedback := &FeedbackService{ProposalRepo: proposals, UserRepo: users, ChangeLogRepo: logs}
	userPrefix := cfg.Mongo.DB + "_"
	for _, id := range []string{"a", "b", "c", "d", "e", "admin", "admin2"} {
		if _, err = db.Collection("user").InsertOne(ctx, &model.User{ID: userPrefix + id, Username: id, Admin: id == "admin" || id == "admin2", Contribution: 100}); err != nil {
			t.Fatal(err)
		}
	}
	as := func(id string) context.Context { return context.WithValue(ctx, consts.CtxUserID, userPrefix+id) }
	course := func(name string) *dto.ProposalCourseVO {
		return &dto.ProposalCourseVO{Name: name, Code: "CS101", Department: "测试学院", Category: "测试分类", Campuses: []string{"普陀校区"}, Teachers: []*dto.TeacherVO{{Name: "新教师", Title: "讲师"}}}
	}
	create := func(user string, c *dto.ProposalCourseVO) *dto.CreateProposalResp {
		t.Helper()
		resp, err := s.CreateProposal(as(user), &dto.CreateProposalReq{Course: c, ShowUsername: true})
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	approve := func(req *dto.ToggleProposalReq) *dto.ToggleProposalResp {
		t.Helper()
		preview, err := s.PreviewApproval(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		req.PreviewToken = preview.PreviewToken
		resp, err := s.ApproveProposal(as("admin"), req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	a := create("a", course("自动组"))
	bCourse := course("自动组")
	bCourse.Teachers[0].Title = "教授"
	b := create("b", bCourse)
	if len(b.PendingDuplicateIDs) != 1 || b.PendingDuplicateIDs[0] != a.ProposalID {
		t.Fatal("different authors should receive nonblocking duplicate notice")
	}
	_, err = s.CreateProposal(as("a"), &dto.CreateProposalReq{Course: course("自动组")})
	assertWorkflowCode(t, err, errno.ErrProposalCourseFoundInProposals)
	draft := course("自动组")
	draft.Department = "管理员确认院系"
	if _, err = s.UpdateProposal(as("admin"), &dto.UpdateProposalReq{ProposalID: a.ProposalID, Course: draft}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a.ProposalID, b.ProposalID} {
		p, err := proposals.FindByID(ctx, id)
		if err != nil || p.Course.Department != "测试学院" || p.FinalCourse == nil || p.FinalCourse.Department != "管理员确认院系" {
			t.Fatal("group draft did not preserve original and sync final", p, err)
		}
	}
	different := course("自动组")
	different.Code = "错误代码"
	different.Teachers[0].Name = "另一教师"
	c := create("c", different)
	preview, err := s.PreviewApproval(as("admin"), &dto.ToggleProposalReq{ProposalID: a.ProposalID})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Members) != 2 || len(preview.Suspicious) != 1 {
		t.Fatalf("members=%d suspicious=%d", len(preview.Members), len(preview.Suspicious))
	}
	_, err = s.ApproveProposal(as("admin"), &dto.ToggleProposalReq{ProposalID: a.ProposalID})
	assertWorkflowCode(t, err, errno.ErrProposalPreviewRequired)
	stale := &dto.ToggleProposalReq{ProposalID: a.ProposalID, PreviewToken: preview.PreviewToken}
	extra := course("自动组")
	extra.Category = "不同分类"
	d := create("d", extra)
	_, err = s.ApproveProposal(as("admin"), stale)
	assertWorkflowCode(t, err, errno.ErrProposalPreviewStale)
	accepted := approve(&dto.ToggleProposalReq{ProposalID: a.ProposalID, ProposalIDs: []string{c.ProposalID}})
	if len(accepted.ProposalIDs) != 3 {
		t.Fatalf("manual+automatic closure=%v", accepted.ProposalIDs)
	}
	count, err := db.Collection("course").CountDocuments(ctx, bson.M{"deleted": bson.M{"$ne": true}})
	if err != nil || count != 1 {
		t.Fatalf("course count=%d err=%v", count, err)
	}
	for _, id := range []string{a.ProposalID, b.ProposalID, c.ProposalID} {
		p, err := proposals.FindByID(ctx, id)
		if err != nil || p.Status != 2 || p.TargetID != accepted.TargetID {
			t.Fatalf("member %s %#v %v", id, p, err)
		}
	}
	p, _ := proposals.FindByID(ctx, d.ProposalID)
	if p.Status != 1 {
		t.Fatal("unselected suspicious proposal changed")
	}
	contributors, err := ca.EntityContributors(ctx, "course", accepted.TargetID, "")
	if err != nil || len(contributors) != 3 {
		t.Fatalf("contributors=%v err=%v", contributors, err)
	}
	history, err := s.EntityHistory(as("a"), &dto.EntityHistoryReq{TargetType: "course", TargetID: accepted.TargetID, PageParam: &dto.PageParam{Page: 1, PageSize: 10}})
	if err != nil || history.Total != 1 {
		t.Fatalf("history=%#v %v", history, err)
	}
	formal, _ := courses.FindByID(ctx, accepted.TargetID)
	teacherID := formal.TeacherIDs[0]
	_, err = s.CreateProposal(as("a"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: formal.ID, Suggested: &dto.ProposalPatch{Code: textPointer(formal.Code)}})
	assertWorkflowCode(t, err, errno.ErrProposalNoChanges)
	modA, err := s.CreateProposal(as("a"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: formal.ID, Suggested: &dto.ProposalPatch{Code: textPointer("CS102")}, ShowUsername: true})
	if err != nil {
		t.Fatal(err)
	}
	modB, err := s.CreateProposal(as("b"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: formal.ID, Suggested: &dto.ProposalPatch{Code: textPointer("CS102")}, ShowUsername: true})
	if err != nil {
		t.Fatal(err)
	}
	update := approve(&dto.ToggleProposalReq{ProposalID: modA.ProposalID})
	if len(update.ProposalIDs) != 2 {
		t.Fatal("modification group not approved together")
	}
	if p, _ := proposals.FindByID(ctx, modB.ProposalID); p.Contribution != 2 {
		t.Fatalf("modify points=%d", p.Contribution)
	}
	_, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: a.ProposalID, ActionType: "approve"})
	assertWorkflowCode(t, err, errno.ErrCourseModifiedCannotRevoke)
	_, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: modB.ProposalID, ActionType: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	restored, _ := courses.FindByID(ctx, formal.ID)
	if restored.Code != "CS101" {
		t.Fatal("course not restored")
	}
	teacherChange, err := s.CreateProposal(as("c"), &dto.CreateProposalReq{Type: model.ProposalUpdateTeacher, TargetID: teacherID, Suggested: &dto.ProposalPatch{Title: textPointer("副教授")}, ShowUsername: true})
	if err != nil {
		t.Fatal(err)
	}
	approve(&dto.ToggleProposalReq{ProposalID: teacherChange.ProposalID})
	detail, err := s.GetTeacher(as("a"), teacherID)
	if err != nil || detail.Teacher.Title != "副教授" {
		t.Fatalf("teacher detail=%v %v", detail, err)
	}
	courseVO, err := ca.ToCourseVO(ctx, restored)
	if err != nil || courseVO.Teachers[0].Title != "副教授" {
		t.Fatalf("global teacher update=%v %v", courseVO, err)
	}
	_, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: teacherChange.ProposalID, ActionType: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: a.ProposalID, ActionType: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	if count, _ = db.Collection("teacher").CountDocuments(ctx, bson.M{}); count != 1 {
		t.Fatalf("teacher targeted by the reverted pending modification must survive, count=%d", count)
	}
	// Concurrent attempts with the same preview must write exactly once.
	req := &dto.ToggleProposalReq{ProposalID: a.ProposalID}
	preview, err = s.PreviewApproval(as("admin"), req)
	if err != nil {
		t.Fatal(err)
	}
	req.PreviewToken = preview.PreviewToken
	req.ConfirmedNewTeachers = []string{"新教师"}
	preview, err = s.PreviewApproval(as("admin"), req)
	if err != nil {
		t.Fatal(err)
	}
	req.PreviewToken = preview.PreviewToken
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.ApproveProposal(as("admin"), copyAs[dto.ToggleProposalReq](req))
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("concurrent successful approvals=%d", success)
	}
	// Conflicting proposals must require explicit final values, and approval
	// must preserve a concurrently changed field outside the proposed fields.
	var active model.Course
	if err = db.Collection("course").FindOne(ctx, bson.M{"name": "自动组", "deleted": bson.M{"$ne": true}}).Decode(&active); err != nil {
		t.Fatal(err)
	}
	first, err := s.CreateProposal(as("a"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: active.ID, Suggested: &dto.ProposalPatch{Code: textPointer("FIRST")}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateProposal(as("b"), &dto.CreateProposalReq{Type: model.ProposalUpdateCourse, TargetID: active.ID, Suggested: &dto.ProposalPatch{Code: textPointer("SECOND")}})
	if err != nil {
		t.Fatal(err)
	}
	approve(&dto.ToggleProposalReq{ProposalID: first.ProposalID})
	conflicting, err := s.PreviewApproval(as("admin"), &dto.ToggleProposalReq{ProposalID: second.ProposalID})
	if err != nil || conflicting.CanApprove || conflicting.Conflicts[0].State != "conflict" {
		t.Fatal("missing conflict", conflicting, err)
	}
	_, err = s.ApproveProposal(as("admin"), &dto.ToggleProposalReq{ProposalID: second.ProposalID, PreviewToken: conflicting.PreviewToken})
	assertWorkflowCode(t, err, errno.ErrProposalFieldConflict)
	approve(&dto.ToggleProposalReq{ProposalID: second.ProposalID, Final: &dto.ProposalPatch{Code: textPointer("THIRD")}})
	if current, _ := courses.FindByID(ctx, active.ID); current.Code != "THIRD" || current.Department != active.Department {
		t.Fatal("conflict re-review or untouched fields wrong", current)
	}
	// Exact formal duplicates block approval and never issue points.
	exactCourse, err := ca.ToProposalCourseVOFromCourse(ctx, func() *model.Course { c, _ := courses.FindByID(ctx, active.ID); return c }())
	if err != nil {
		t.Fatal(err)
	}
	exact := create("e", exactCourse)
	duplicatePreview, err := s.PreviewApproval(as("admin"), &dto.ToggleProposalReq{ProposalID: exact.ProposalID})
	if err != nil || duplicatePreview.CanApprove || len(duplicatePreview.ExistingCourses) == 0 {
		t.Fatal("formal duplicate missing", duplicatePreview, err)
	}
	_, err = s.ApproveProposal(as("admin"), &dto.ToggleProposalReq{ProposalID: exact.ProposalID, PreviewToken: duplicatePreview.PreviewToken})
	assertWorkflowCode(t, err, errno.ErrProposalCourseFoundInCourses)
	if record, _ := proposals.FindByID(ctx, exact.ProposalID); record.Status != 1 || record.Contribution != 0 {
		t.Fatal("duplicate was processed", record)
	}
	if _, err = s.RejectProposal(as("admin"), &dto.RejectProposalReq{ProposalID: exact.ProposalID, ProposalIDs: []string{d.ProposalID}, Reason: "已有课程"}); err != nil {
		t.Fatal(err)
	}
	// Reused formal teachers must survive course creation revocation.
	teacherID = "existing-independent-teacher"
	if _, err = db.Collection("teacher").InsertOne(ctx, &model.Teacher{ID: teacherID, Name: "独立教师", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	reuse := course("复用教师")
	reuse.Teachers = []*dto.TeacherVO{{ID: teacherID, Name: "新教师"}}
	reused := create("e", reuse)
	approvedReuse := approve(&dto.ToggleProposalReq{ProposalID: reused.ProposalID})
	_, err = s.RevokeProposal(as("admin"), &dto.RevokeProposalReq{ProposalID: reused.ProposalID, ActionType: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	if teacher, _ := teachers.FindByID(ctx, teacherID); teacher == nil {
		t.Fatal("reused teacher deleted")
	}
	if record, _ := courses.FindByID(ctx, approvedReuse.TargetID); record != nil {
		t.Fatal("created course not deleted")
	}

	t.Run("anonymous identity and current username", func(t *testing.T) {
		anon, err := s.CreateProposal(as("e"), &dto.CreateProposalReq{Course: course("匿名课程")})
		if err != nil {
			t.Fatal(err)
		}
		approved := approve(&dto.ToggleProposalReq{ProposalID: anon.ProposalID, ConfirmedNewTeachers: []string{"新教师"}})
		for _, viewer := range []string{"a", "e", "admin"} {
			record, err := s.GetProposal(as(viewer), &dto.GetProposalReq{ProposalID: anon.ProposalID})
			if err != nil {
				t.Fatal(err)
			}
			if (record.Proposal.UserID != "") != (viewer != "a") {
				t.Fatal("anonymous identity exposed or hidden from owner/admin", viewer, record)
			}
		}
		contributors, err := ca.EntityContributors(ctx, "course", approved.TargetID, "")
		if err != nil || len(contributors) != 0 {
			t.Fatal("anonymous contributor exposed", contributors, err)
		}
		if _, err = db.Collection("user").UpdateOne(ctx, bson.M{"_id": userPrefix + "a"}, bson.M{"$set": bson.M{"username": "改名后的昵称"}}); err != nil {
			t.Fatal(err)
		}
		contributors, err = ca.EntityContributors(ctx, "course", active.ID, "")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, contributor := range contributors {
			if contributor.UserID == userPrefix+"a" {
				found = contributor.Username == "改名后的昵称"
			}
		}
		if !found {
			t.Fatal("contributor name did not follow current profile", contributors)
		}
	})
	t.Run("deleted audit and legacy history", func(t *testing.T) {
		discarded := create("d", course("待删除提案"))
		_, err := s.DeleteProposal(as("e"), &dto.DeleteProposalReq{ProposalID: discarded.ProposalID})
		assertWorkflowCode(t, err, errno.ErrUserNotOwner)
		if _, err = s.DeleteProposal(as("d"), &dto.DeleteProposalReq{ProposalID: discarded.ProposalID}); err != nil {
			t.Fatal(err)
		}
		var event model.ChangeLog
		if err = db.Collection("changelog").FindOne(ctx, bson.M{"proposalId": discarded.ProposalID, "action": consts.ActionTypeDeleteProposal}).Decode(&event); err != nil || event.Snapshot == nil || !event.Snapshot.Deleted {
			t.Fatal("missing immutable delete snapshot", event, err)
		}
		legacy := &model.Proposal{ID: "legacy-source", UserID: userPrefix + "a", Status: 2, ShowUsername: true, Course: copyAs[model.ProposalCourse](course("历史课程")), CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now().Add(-time.Minute)}
		if _, err = db.Collection("proposal").InsertOne(ctx, legacy); err != nil {
			t.Fatal(err)
		}
		legacyCourse := &model.Course{ID: "legacy-course", Name: "历史课程", Code: "CS101", Category: 1, Department: 1, Campuses: []int32{1}, TeacherIDs: []string{teacherID}, ProposalID: legacy.ID}
		if _, err = db.Collection("course").InsertOne(ctx, legacyCourse); err != nil {
			t.Fatal(err)
		}
		history, err := s.EntityHistory(as("a"), &dto.EntityHistoryReq{TargetType: "course", TargetID: legacyCourse.ID})
		if err != nil || history.Total != 1 || !history.History[0].Legacy || history.History[0].Final != nil {
			t.Fatal("legacy history invented final state", history, err)
		}
		for i, name := range []string{"去重课程", "去重课程", "去重课程进阶"} {
			if _, err = db.Collection("course").InsertOne(ctx, &model.Course{ID: fmt.Sprintf("suggestion-%d", i), Name: name}); err != nil {
				t.Fatal(err)
			}
		}
		first, total, err := courses.GetSuggestionsByName(ctx, "去重课程", &dto.PageParam{Page: 1, PageSize: 1})
		if err != nil || total != 2 || len(first) != 1 || first[0].Name != "去重课程" {
			t.Fatal("name dedup before pagination failed", first, total, err)
		}
		second, total, err := courses.GetSuggestionsByName(ctx, "去重课程", &dto.PageParam{Page: 2, PageSize: 1})
		if err != nil || total != 2 || len(second) != 1 || second[0].Name != "去重课程进阶" {
			t.Fatal("unique pagination failed", second, total, err)
		}
	})
	t.Run("private feedback", func(t *testing.T) {
		created, err := feedback.Create(as("a"), &dto.CreateFeedbackReq{Text: "反馈正文"})
		if err != nil {
			t.Fatal(err)
		}
		id := created.Feedback.ID
		_, err = feedback.Detail(as("b"), id, nil, false)
		assertWorkflowCode(t, err, errno.ErrFeedbackNotFound)
		_, err = feedback.Detail(as("b"), id, nil, true)
		assertWorkflowCode(t, err, errno.ErrUserNotAdmin)
		adminDetail, err := feedback.Detail(as("admin"), id, nil, true)
		if err != nil || adminDetail.Feedback.UnreadCount != 1 {
			t.Fatalf("admin unread %#v %v", adminDetail, err)
		}
		if _, err = feedback.Read(as("admin"), id, &dto.FeedbackReadReq{Sequence: 1}, true); err != nil {
			t.Fatal(err)
		}
		otherAdmin, err := feedback.Detail(as("admin2"), id, nil, true)
		if err != nil || otherAdmin.Feedback.UnreadCount != 1 {
			t.Fatal("admin read states shared")
		}
		reply, err := feedback.Send(as("admin"), id, &dto.FeedbackMessageReq{Text: "管理员回复"}, true)
		if err != nil || reply.Feedback.Status != "answered" {
			t.Fatalf("reply %#v %v", reply, err)
		}
		unread, err := feedback.Unread(as("a"), false)
		if err != nil || unread.UnreadCount != 1 {
			t.Fatal("author unread missing")
		}
		if _, err = feedback.Close(as("a"), id, false); err != nil {
			t.Fatal(err)
		}
		continued, err := feedback.Send(as("a"), id, &dto.FeedbackMessageReq{Text: "继续反馈"}, false)
		if err != nil || continued.Feedback.Status != "pending" {
			t.Fatal("closed feedback not reopened")
		}
		_, err = feedback.Read(as("a"), id, &dto.FeedbackReadReq{Sequence: 999}, false)
		assertWorkflowCode(t, err, errno.ErrFeedbackInvalid)

		// Reading an older page must not move the cursor back or consume new messages.
		if _, err = feedback.Read(as("admin2"), id, &dto.FeedbackReadReq{Sequence: 1}, true); err != nil {
			t.Fatal(err)
		}
		detail, err := feedback.Detail(as("admin2"), id, nil, true)
		if err != nil || detail.Feedback.UnreadCount != 1 {
			t.Fatal("new reply consumed by old read", detail, err)
		}
		for i := 0; i < 3; i++ {
			if _, err = feedback.Send(as("a"), id, &dto.FeedbackMessageReq{Text: "补充消息"}, false); err != nil {
				t.Fatal(err)
			}
		}
		_, err = feedback.Send(as("a"), id, &dto.FeedbackMessageReq{Text: "超限消息"}, false)
		assertWorkflowCode(t, err, errno.ErrFeedbackRateLimit)
		if detail, err = feedback.Detail(as("a"), id, nil, false); err != nil || detail.Total != 6 {
			t.Fatal("rate failure wrote partial message", detail, err)
		}
	})
	exerciseWorkflowEdges(t, ctx, s, feedback, db, active.ID, userPrefix)
	exerciseWorkflowCrossOps(t, ctx, s, cfg, userPrefix)
	exerciseWorkflowResilience(t, ctx, s, feedback, cfg, userPrefix)
}
