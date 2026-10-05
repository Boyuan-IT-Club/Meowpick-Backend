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

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/token"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/provider"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestWorkflowHTTPIntegration exercises real JSON, JWTs, routes, providers and
// transactions. Only the local, isolated MongoDB/Redis supplied by the runner
// are used; WeChat credentials below are fake and no login API is called.
func TestWorkflowHTTPIntegration(t *testing.T) {
	uri, redisAddr := os.Getenv("MEOWPICK_TEST_MONGO_URI"), os.Getenv("MEOWPICK_TEST_REDIS_ADDR")
	if uri == "" || redisAddr == "" {
		t.Skip("isolated replica-set MongoDB and Redis required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	dbName := fmt.Sprintf("workflow_http_test_%d", time.Now().UnixNano())
	db := client.Database(dbName)
	defer db.Drop(context.Background())
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	yaml := fmt.Sprintf(`Name: meowpick-http-test
Mode: test
ListenOn: 127.0.0.1:0
State: test
Log:
  Level: error
Auth:
  SecretKey: workflow-test-only-secret
  PublicKey: workflow-test-only-public
  AccessExpire: 3600
Mongo:
  URL: %q
  DB: %q
Redis:
  Host: %q
  Type: node
Cache:
  - Host: %q
    Type: node
    Weight: 100
WeApp:
  AppID: test-only
  AppSecret: test-only
DebugLogin:
  Enabled: false
  VerifyCode: test-unused
  OpenID: test-unused
AdminGrantKey: test-unused
`, uri, dbName, redisAddr, redisAddr)
	if err = os.WriteFile(configPath, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", configPath)
	if _, err = db.Collection("mapping").InsertOne(ctx, &model.Mapping{Type: model.MappingTypeCampus, Name: "普陀校区", Code: 1, Canonical: true}); err != nil {
		t.Fatal(err)
	}
	provider.Init()
	tokens := map[string]string{}
	for _, id := range []string{"http-a", "http-b", "http-admin"} {
		user := &model.User{ID: dbName + "_" + id, Username: id, Admin: id == "http-admin", Contribution: 100}
		if _, err = db.Collection("user").InsertOne(ctx, user); err != nil {
			t.Fatal(err)
		}
		tokens[id], err = token.NewAuthorizedToken(user)
		if err != nil {
			t.Fatal(err)
		}
	}
	routes := SetupRoutes()
	call := func(t *testing.T, user, method, path string, body any, wantCode int) map[string]any {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		if user != "" {
			req.Header.Set("Authorization", "Bearer "+tokens[user])
		}
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, req)
		var response struct {
			Code int            `json:"code"`
			Data map[string]any `json:"data"`
		}
		if err = json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("%s %s returned invalid JSON: %s", method, path, recorder.Body.String())
		}
		if recorder.Code != http.StatusOK || response.Code != wantCode {
			t.Fatalf("%s %s: HTTP %d, body %s; want business code %d", method, path, recorder.Code, recorder.Body.String(), wantCode)
		}
		if wantCode != 0 && response.Data != nil {
			t.Fatalf("business failure exposed data: %s", recorder.Body.String())
		}
		return response.Data
	}
	admin := "http-admin"
	approve := func(t *testing.T, id string, body map[string]any) map[string]any {
		t.Helper()
		preview := call(t, admin, "POST", "/api/proposal/"+id+"/preview", body, 0)
		if preview["canApprove"] != true {
			t.Fatalf("unexpected approval block: %#v", preview)
		}
		body["previewToken"] = preview["previewToken"]
		return call(t, admin, "POST", "/api/proposal/"+id+"/approve", body, 0)
	}
	courseBody := map[string]any{
		"type": "create_course", "showUsername": true,
		"course": map[string]any{"name": "HTTP课程", "code": "HTTP1", "category": "接口分类", "department": "接口学院", "campuses": []string{"普陀校区"}, "teachers": []map[string]any{{"name": "HTTP教师", "title": "讲师", "department": "接口学院"}}},
	}
	var proposalA, courseID, teacherID string
	t.Run("JSON and authorization", func(t *testing.T) {
		call(t, "", "POST", "/api/feedback", map[string]any{"text": "未登录"}, errno.ErrUserNotLogin)
		call(t, "http-a", "POST", "/api/feedback", map[string]any{"text": 123}, errno.ErrRequestInvalid)
		data := call(t, "http-a", "POST", "/api/proposal/add", courseBody, 0)
		proposalA = data["proposalId"].(string)
		if _, exists := data["proposal"].(map[string]any)["title"]; exists {
			t.Fatal("removed proposal title leaked into JSON")
		}
		call(t, "http-a", "POST", "/api/proposal/"+proposalA+"/preview", map[string]any{}, errno.ErrUserNotAdmin)
		call(t, admin, "POST", "/api/proposal/"+proposalA+"/approve", map[string]any{}, errno.ErrProposalPreviewRequired)
	})
	t.Run("shared approval and public course", func(t *testing.T) {
		duplicate := call(t, "http-b", "POST", "/api/proposal/add", courseBody, 0)
		if len(duplicate["pendingDuplicateIds"].([]any)) != 1 {
			t.Fatal("duplicate notice missing")
		}
		approved := approve(t, proposalA, map[string]any{})
		if len(approved["proposalIds"].([]any)) != 2 {
			t.Fatal("HTTP approval did not process exact group")
		}
		courseID = approved["targetId"].(string)
		course := call(t, "http-a", "GET", "/api/course/"+courseID, nil, 0)["course"].(map[string]any)
		if len(course["contributors"].([]any)) != 2 {
			t.Fatal("contributors array missing")
		}
		if _, exists := course["contributor"]; exists {
			t.Fatal("old singular contributor exposed")
		}
		teacherID = course["teachers"].([]any)[0].(map[string]any)["id"].(string)
	})
	t.Run("course and teacher edits", func(t *testing.T) {
		edit := call(t, "http-a", "POST", "/api/proposal/add", map[string]any{"type": "update_course", "targetId": courseID, "suggested": map[string]any{"code": "HTTP2"}}, 0)
		approve(t, edit["proposalId"].(string), map[string]any{})
		call(t, "http-b", "POST", "/api/proposal/add", map[string]any{"type": "update_course", "targetId": courseID, "suggested": map[string]any{"code": "HTTP2"}}, errno.ErrProposalNoChanges)
		teacherEdit := call(t, "http-b", "POST", "/api/proposal/add", map[string]any{"type": "update_teacher", "targetId": teacherID, "suggested": map[string]any{"title": "教授", "department": ""}}, 0)
		approve(t, teacherEdit["proposalId"].(string), map[string]any{})
		teacher := call(t, "http-a", "GET", "/api/teacher/"+teacherID, nil, 0)["teacher"].(map[string]any)
		if teacher["department"] != "" || teacher["title"] != "教授" {
			t.Fatalf("teacher edit/clearing JSON wrong: %#v", teacher)
		}
		course := call(t, "http-a", "GET", "/api/course/"+courseID, nil, 0)["course"].(map[string]any)
		if course["code"] != "HTTP2" || course["teachers"].([]any)[0].(map[string]any)["title"] != "教授" {
			t.Fatal("formal course did not reflect edits")
		}
		for _, path := range []string{"/api/course/" + courseID + "/history", "/api/teacher/" + teacherID + "/history"} {
			if data := call(t, "http-a", "GET", path, nil, 0); data["total"] != float64(2) {
				t.Fatalf("incorrect history: %s %#v", path, data)
			}
		}
	})
	t.Run("administrator logs", func(t *testing.T) {
		call(t, "http-a", "POST", "/api/changelog/list", map[string]any{}, errno.ErrUserNotAdmin)
		data := call(t, admin, "POST", "/api/changelog/list", map[string]any{"page": 1, "pageSize": 100}, 0)
		found := false
		for _, item := range data["changeLogs"].([]any) {
			log := item.(map[string]any)
			if log["proposalType"] == "update_teacher" && log["entityType"] == "teacher" && log["decisionBatchId"] != "" {
				found = true
			}
		}
		if !found {
			t.Fatal("new teacher operation missing from logs")
		}
		call(t, admin, "GET", "/api/changelog/proposal/grouped?pageSize=100", nil, 0)
		call(t, admin, "GET", "/api/changelog/proposal/timeline?pageSize=100", nil, 0)
	})
	t.Run("private feedback round trip", func(t *testing.T) {
		created := call(t, "http-a", "POST", "/api/feedback", map[string]any{"text": "接口反馈"}, 0)
		id := created["feedback"].(map[string]any)["id"].(string)
		call(t, "http-b", "GET", "/api/feedback/"+id, nil, errno.ErrFeedbackNotFound)
		call(t, "http-b", "GET", "/api/admin/feedback", nil, errno.ErrUserNotAdmin)
		call(t, admin, "GET", "/api/admin/feedback?keyword=接口反馈", nil, 0)
		call(t, admin, "POST", "/api/admin/feedback/"+id+"/reply", map[string]any{"text": "收到反馈"}, 0)
		unread := call(t, "http-a", "GET", "/api/feedback/unread", nil, 0)
		if unread["unreadCount"] != float64(1) {
			t.Fatalf("wrong static unread route/response: %#v", unread)
		}
		detail := call(t, "http-a", "GET", "/api/feedback/"+id, nil, 0)
		if detail["total"] != float64(2) || detail["feedback"].(map[string]any)["status"] != "answered" {
			t.Fatal("feedback reply/status did not persist")
		}
		call(t, "http-a", "POST", "/api/feedback/"+id+"/read", map[string]any{"sequence": 2}, 0)
		if data := call(t, "http-a", "GET", "/api/feedback/unread", nil, 0); data["unreadCount"] != float64(0) {
			t.Fatal("read cursor not reflected")
		}
		call(t, "http-a", "POST", "/api/feedback/"+id+"/close", map[string]any{}, 0)
		reopened := call(t, "http-a", "POST", "/api/feedback/"+id+"/messages", map[string]any{"text": "补充信息"}, 0)
		if reopened["feedback"].(map[string]any)["status"] != "pending" {
			t.Fatal("feedback did not reopen")
		}
	})
}
