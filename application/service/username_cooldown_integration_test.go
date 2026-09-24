package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

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

// Run only against disposable MongoDB and Redis instances.
func TestResetUsernameCooldownIntegration(t *testing.T) {
	uri, redisHost := os.Getenv("MEOWPICK_TEST_MONGO_URI"), os.Getenv("MEOWPICK_TEST_REDIS_ADDR")
	if uri == "" || redisHost == "" {
		t.Skip("set MEOWPICK_TEST_MONGO_URI and MEOWPICK_TEST_REDIS_ADDR for isolated integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	cfg := &config.Config{Redis: &redis.RedisConf{Host: redisHost, Type: "node"}}
	cfg.Mongo.URL = uri
	cfg.Mongo.DB = fmt.Sprintf("username_cooldown_test_%d", time.Now().UnixNano())
	cfg.Cache = cache.CacheConf{{RedisConf: *cfg.Redis, Weight: 100}}
	db := client.Database(cfg.Mongo.DB)
	defer db.Drop(context.Background())
	users, err := repo.NewUserRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, user := range []*model.User{
		{ID: "admin", Admin: true},
		{ID: "ordinary"},
		{ID: "target", Username: "Alice", UsernameUpdatedAt: now},
	} {
		if _, err := db.Collection("user").InsertOne(ctx, user); err != nil {
			t.Fatal(err)
		}
	}
	s := &UserService{UserRepo: users}
	actor := func(id string) context.Context { return context.WithValue(ctx, consts.CtxUserID, id) }
	// Warm the cache to catch stale profile reads after the reset.
	if user, err := users.FindByID(ctx, "target"); err != nil || canEditUsername(user.UsernameUpdatedAt, now) {
		t.Fatalf("target should initially be in cooldown: user=%+v err=%v", user, err)
	}
	for _, id := range []string{"", "ordinary"} {
		if _, err := s.ResetUsernameCooldown(actor(id), "target"); err == nil {
			t.Fatalf("actor %q should be rejected", id)
		}
	}
	if _, err := s.ResetUsernameCooldown(actor("admin"), "missing"); err == nil {
		t.Fatal("missing target should be rejected")
	}
	for i := 0; i < 2; i++ {
		result, err := s.ResetUsernameCooldown(actor("admin"), "target")
		if err != nil || result == nil || result.UserID != "target" || !result.CanEditUsername {
			t.Fatalf("reset %d: result=%+v err=%v", i, result, err)
		}
	}
	var stored model.User
	if err := db.Collection("user").FindOne(ctx, bson.M{consts.ID: "target"}).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if stored.Username != "Alice" || !stored.UsernameUpdatedAt.IsZero() {
		t.Fatalf("reset changed nickname or preserved cooldown: %+v", stored)
	}
	if user, err := users.FindByID(ctx, "target"); err != nil || user.Username != "Alice" || !canEditUsername(user.UsernameUpdatedAt, time.Now()) {
		t.Fatalf("cached target was not refreshed: user=%+v err=%v", user, err)
	}
}
