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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/assembler"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	appcache "github.com/Boyuan-IT-Club/Meowpick-Backend/infra/cache"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/config"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Run against disposable local MongoDB (replica set) and Redis instances.
// Only the uniquely named database created by this test is dropped.
func TestPendingAndCommentPermissionsIntegration(t *testing.T) {
	uri, redisHost := os.Getenv("MEOWPICK_TEST_MONGO_URI"), os.Getenv("MEOWPICK_TEST_REDIS_ADDR")
	if uri == "" || redisHost == "" {
		t.Skip("set MEOWPICK_TEST_MONGO_URI and MEOWPICK_TEST_REDIS_ADDR for isolated integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	cfg := &config.Config{Redis: &redis.RedisConf{Host: redisHost, Type: "node"}}
	cfg.Mongo.URL = uri
	cfg.Mongo.DB = fmt.Sprintf("permissions_test_%d", time.Now().UnixNano())
	cfg.Cache = cache.CacheConf{{RedisConf: *cfg.Redis, Weight: 100}}
	db := client.Database(cfg.Mongo.DB)
	defer db.Drop(context.Background())
	users, err := repo.NewUserRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	comments, err := repo.NewCommentRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proposals, err := repo.NewProposalRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	courses, err := repo.NewCourseRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	likes := repo.NewLikeRepo(cfg)
	prefix := cfg.Mongo.DB + "_"
	owner, other, admin := prefix+"owner", prefix+"other", prefix+"admin"
	for _, u := range []*model.User{{ID: owner}, {ID: other}, {ID: admin, Admin: true}} {
		if _, err := db.Collection("user").InsertOne(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	userContext := func(id string) context.Context { return context.WithValue(ctx, consts.CtxUserID, id) }
	commentService := &CommentService{UserRepo: users, CommentRepo: comments, LikeRepo: likes, CommentCache: appcache.NewCommentCache(cfg)}
	for _, tt := range []struct {
		name, actor string
		want        bool
	}{{"owner", owner, true}, {"other", other, false}, {"admin", admin, true}, {"anonymous", "", false}} {
		t.Run("delete_"+tt.name, func(t *testing.T) {
			id := prefix + tt.name
			if _, err := db.Collection("comment").InsertOne(ctx, &model.Comment{ID: id, UserID: owner}); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Collection("like").InsertOne(ctx, bson.M{"_id": id, "targetId": id, "targetType": 2, "userId": other, "active": true}); err != nil {
				t.Fatal(err)
			}
			if err := commentService.CommentCache.SetCount(ctx, 99, time.Minute); err != nil {
				t.Fatal(err)
			}
			_, err := commentService.DeleteComment(userContext(tt.actor), &dto.DeleteCommentReq{CommentID: id})
			if (err == nil) != tt.want {
				t.Fatalf("delete error=%v, want success=%v", err, tt.want)
			}
			var c model.Comment
			if err := db.Collection("comment").FindOne(ctx, bson.M{"_id": id}).Decode(&c); err != nil {
				t.Fatal(err)
			}
			if c.Deleted != tt.want {
				t.Fatalf("deleted=%v, want %v", c.Deleted, tt.want)
			}
			count, err := db.Collection("like").CountDocuments(ctx, bson.M{"targetId": id})
			if err != nil {
				t.Fatal(err)
			}
			if (count == 0) != tt.want {
				t.Fatalf("likes remaining=%d, want deleted=%v", count, tt.want)
			}
			if tt.want {
				if _, hit, err := commentService.CommentCache.GetCount(ctx); err != nil || hit {
					t.Fatalf("comment count cache not invalidated: hit=%v err=%v", hit, err)
				}
				if _, err := commentService.DeleteComment(userContext(tt.actor), &dto.DeleteCommentReq{CommentID: id}); err == nil {
					t.Fatal("second delete should fail")
				}
			}
		})
	}
	if _, err := commentService.DeleteComment(userContext(admin), &dto.DeleteCommentReq{CommentID: "missing"}); err == nil {
		t.Fatal("missing comment should fail")
	}
	proposalService := &ProposalService{UserRepo: users, ProposalRepo: proposals, CourseRepo: courses, ProposalAssembler: &assembler.ProposalAssembler{LikeRepo: likes, CourseAssembler: &assembler.CourseAssembler{}}}
	now := time.Now()
	for i, p := range []*model.Proposal{
		{ID: "pending-own", UserID: owner, Status: 1, Contribution: 7},
		{ID: "pending-other", UserID: other, Status: 1, Contribution: 8},
		{ID: "approved", UserID: other, Status: 2},
		{ID: "rejected", UserID: other, Status: 3},
		{ID: "deleted", UserID: other, Status: 1, Deleted: true},
	} {
		p.CreatedAt = now.Add(-time.Duration(i) * time.Minute)
		if _, err := db.Collection("proposal").InsertOne(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := proposalService.ListPendingProposals(ctx, &dto.PageParam{}); err == nil {
		t.Fatal("anonymous pending list should fail")
	}
	for _, actor := range []string{owner, admin} {
		result, err := proposalService.ListPendingProposals(userContext(actor), &dto.PageParam{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 2 || len(result.Proposals) != 1 || result.Proposals[0].ID != "pending-own" {
			t.Fatalf("unexpected pending first page: %+v", result)
		}
		wantContribution := int64(-1)
		if actor == owner {
			wantContribution = 7
		}
		if result.Proposals[0].Contribution != wantContribution {
			t.Fatal("contribution visibility violated")
		}
		result, err = proposalService.ListPendingProposals(userContext(actor), &dto.PageParam{Page: 2, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Proposals) != 1 || result.Proposals[0].ID != "pending-other" || result.Proposals[0].Contribution != -1 {
			t.Fatalf("unexpected pending second page: %+v", result)
		}
	}
	result, err := proposalService.ListProposals(userContext(owner), &dto.ListProposalReq{Status: "pending"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Proposals) != 1 || result.Proposals[0].ID != "approved" {
		t.Fatal("existing list permissions changed")
	}
	if _, err := db.Collection("proposal").DeleteMany(ctx, bson.M{"status": 1}); err != nil {
		t.Fatal(err)
	}
	result, err = proposalService.ListPendingProposals(userContext(owner), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || result.Proposals == nil || len(result.Proposals) != 0 {
		t.Fatal("empty pending list must be [] with total 0")
	}
}
