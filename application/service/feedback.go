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
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/page"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/types/errno"
	"github.com/Boyuan-IT-Club/go-kit/errorx"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const feedbackCollection = "feedback"
const feedbackMessageCollection = "feedback_message"
const feedbackReadCollection = "feedback_read"
const feedbackTargetType int32 = 5
const feedbackReplyAction int32 = 110
const feedbackCloseAction int32 = 111

type FeedbackService struct {
	ProposalRepo  *repo.ProposalRepo
	UserRepo      *repo.UserRepo
	ChangeLogRepo *repo.ChangeLogRepo
}

var FeedbackServiceSet = wire.NewSet(wire.Struct(new(FeedbackService), "*"))

func (s *FeedbackService) actor(ctx context.Context, admin bool) (string, error) {
	return (&ProposalService{UserRepo: s.UserRepo}).actor(ctx, admin)
}
func feedbackText(text string) (string, error) {
	text = strings.TrimSpace(text)
	length := utf8.RuneCountInString(text)
	if length < 1 || length > 2000 {
		return "", errorx.New(errno.ErrFeedbackInvalid)
	}
	return text, nil
}
func (s *FeedbackService) find(ctx context.Context, id, actor string, admin bool) (*model.Feedback, error) {
	filter := bson.M{"_id": id}
	if !admin {
		filter["userId"] = actor
	}
	var feedback model.Feedback
	err := s.ProposalRepo.Database().Collection(feedbackCollection).FindOne(ctx, filter).Decode(&feedback)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errorx.New(errno.ErrFeedbackNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}
func (s *FeedbackService) audit(ctx context.Context, feedback *model.Feedback, actor, content string, action int32) error {
	return s.ChangeLogRepo.Insert(ctx, &model.ChangeLog{ID: primitive.NewObjectID().Hex(), TargetID: feedback.ID, TargetType: feedbackTargetType, EntityType: "feedback", EntityID: feedback.ID, UserID: actor, Action: action, Content: content, UpdateSource: consts.UpdateSourceAdmin, UpdatedAt: time.Now().UTC()})
}
func (s *FeedbackService) Create(ctx context.Context, req *dto.CreateFeedbackReq) (*dto.FeedbackResp, error) {
	actor, err := s.actor(ctx, false)
	if err != nil {
		return nil, err
	}
	text, err := feedbackText(req.Text)
	if err != nil {
		return nil, err
	}
	category := req.Category
	if category == "" {
		category = "other"
	}
	if category != "bug" && category != "feature" && category != "other" {
		return nil, errorx.New(errno.ErrFeedbackInvalid)
	}
	return s.send(ctx, "", actor, false, text, category)
}
func (s *FeedbackService) Send(ctx context.Context, id string, req *dto.FeedbackMessageReq, admin bool) (*dto.FeedbackResp, error) {
	actor, err := s.actor(ctx, admin)
	if err != nil {
		return nil, err
	}
	text, err := feedbackText(req.Text)
	if err != nil {
		return nil, err
	}
	return s.send(ctx, id, actor, admin, text, "")
}
func (s *FeedbackService) send(ctx context.Context, id, actor string, admin bool, text, category string) (*dto.FeedbackResp, error) {
	if err := s.ProposalRepo.AcquireCreateGuards(ctx, "feedback:"+actor, "feedback:"+actor); err != nil {
		return nil, err
	}
	var feedback *model.Feedback
	var message *model.FeedbackMessage
	err := s.ProposalRepo.WithTransaction(ctx, func(tx mongo.SessionContext) error {
		if err := s.ProposalRepo.AcquireCreateGuards(tx, "feedback:"+actor, "feedback:"+actor); err != nil {
			return err
		}
		now := time.Now().UTC()
		count, err := s.ProposalRepo.Database().Collection(feedbackMessageCollection).CountDocuments(tx, bson.M{"userId": actor, "createdAt": bson.M{"$gt": now.Add(-time.Minute)}})
		if err != nil {
			return err
		}
		if count >= 5 {
			return errorx.New(errno.ErrFeedbackRateLimit)
		}
		if id == "" {
			local := now.In(time.FixedZone("Asia/Shanghai", 8*3600))
			start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
			count, err := s.ProposalRepo.Database().Collection(feedbackCollection).CountDocuments(tx, bson.M{"userId": actor, "createdAt": bson.M{"$gte": start, "$lt": start.Add(24 * time.Hour)}})
			if err != nil {
				return err
			}
			if count >= 10 {
				return errorx.New(errno.ErrFeedbackDailyLimit)
			}
			feedback = &model.Feedback{ID: primitive.NewObjectID().Hex(), UserID: actor, Category: category, Status: "pending", CreatedAt: now, UpdatedAt: now, Sequence: 1, Summary: text}
			if _, err = s.ProposalRepo.Database().Collection(feedbackCollection).InsertOne(tx, feedback); err != nil {
				return err
			}
		} else {
			feedback, err = s.find(tx, id, actor, admin)
			if err != nil {
				return err
			}
			feedback.Sequence++
			feedback.UpdatedAt = now
			feedback.Summary = text
			feedback.Status = "pending"
			if admin {
				feedback.Status = "answered"
			}
			if _, err = s.ProposalRepo.Database().Collection(feedbackCollection).ReplaceOne(tx, bson.M{"_id": feedback.ID}, feedback); err != nil {
				return err
			}
		}
		role := "author"
		if admin {
			role = "admin"
		}
		message = &model.FeedbackMessage{ID: primitive.NewObjectID().Hex(), FeedbackID: feedback.ID, UserID: actor, Role: role, Sequence: feedback.Sequence, Text: text, CreatedAt: now}
		if _, err = s.ProposalRepo.Database().Collection(feedbackMessageCollection).InsertOne(tx, message); err != nil {
			return err
		}
		if admin {
			return s.audit(tx, feedback, actor, "管理员回复反馈", feedbackReplyAction)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	unread, err := s.unreadFor(ctx, feedback.ID, actor, admin)
	if err != nil {
		return nil, err
	}
	return &dto.FeedbackResp{Resp: dto.Success(), Feedback: &dto.FeedbackVO{Feedback: feedback, UnreadCount: unread}, Messages: []*model.FeedbackMessage{message}, Total: feedback.Sequence}, nil
}
func feedbackReaderID(id, actor string, admin bool) string {
	role := "author"
	if admin {
		role = "admin"
	}
	return role + ":" + actor + ":" + id
}
func (s *FeedbackService) unreadFor(ctx context.Context, id, actor string, admin bool) (int64, error) {
	var read struct {
		Sequence int64 `bson:"sequence"`
	}
	err := s.ProposalRepo.Database().Collection(feedbackReadCollection).FindOne(ctx, bson.M{"_id": feedbackReaderID(id, actor, admin)}).Decode(&read)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return 0, err
	}
	opposite := "admin"
	if admin {
		opposite = "author"
	}
	return s.ProposalRepo.Database().Collection(feedbackMessageCollection).CountDocuments(ctx, bson.M{"feedbackId": id, "role": opposite, "sequence": bson.M{"$gt": read.Sequence}})
}
func (s *FeedbackService) Detail(ctx context.Context, id string, param *dto.PageParam, admin bool) (*dto.FeedbackResp, error) {
	actor, err := s.actor(ctx, admin)
	if err != nil {
		return nil, err
	}
	feedback, err := s.find(ctx, id, actor, admin)
	if err != nil {
		return nil, err
	}
	messages := []*model.FeedbackMessage{}
	cursor, err := s.ProposalRepo.Database().Collection(feedbackMessageCollection).Find(ctx, bson.M{"feedbackId": id}, page.FindPageOption(param).SetSort(bson.D{{Key: "sequence", Value: -1}}))
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &messages)
	cursor.Close(ctx)
	if err != nil {
		return nil, err
	}
	unread, err := s.unreadFor(ctx, id, actor, admin)
	if err != nil {
		return nil, err
	}
	return &dto.FeedbackResp{Resp: dto.Success(), Feedback: &dto.FeedbackVO{Feedback: feedback, UnreadCount: unread}, Messages: messages, Total: feedback.Sequence}, nil
}
func (s *FeedbackService) Read(ctx context.Context, id string, req *dto.FeedbackReadReq, admin bool) (*dto.Resp, error) {
	actor, err := s.actor(ctx, admin)
	if err != nil {
		return nil, err
	}
	feedback, err := s.find(ctx, id, actor, admin)
	if err != nil {
		return nil, err
	}
	if req.Sequence < 0 || req.Sequence > feedback.Sequence {
		return nil, errorx.New(errno.ErrFeedbackInvalid)
	}
	_, err = s.ProposalRepo.Database().Collection(feedbackReadCollection).UpdateOne(ctx, bson.M{"_id": feedbackReaderID(id, actor, admin)}, bson.M{"$max": bson.M{"sequence": req.Sequence}, "$set": bson.M{"feedbackId": id, "userId": actor, "admin": admin}}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	return dto.Success(), nil
}
func (s *FeedbackService) Close(ctx context.Context, id string, admin bool) (*dto.Resp, error) {
	actor, err := s.actor(ctx, admin)
	if err != nil {
		return nil, err
	}
	err = s.ProposalRepo.WithTransaction(ctx, func(tx mongo.SessionContext) error {
		feedback, err := s.find(tx, id, actor, admin)
		if err != nil {
			return err
		}
		if feedback.Status == "closed" {
			return nil
		}
		feedback.Status = "closed"
		feedback.UpdatedAt = time.Now().UTC()
		if _, err = s.ProposalRepo.Database().Collection(feedbackCollection).ReplaceOne(tx, bson.M{"_id": id}, feedback); err != nil {
			return err
		}
		if admin {
			return s.audit(tx, feedback, actor, "关闭反馈", feedbackCloseAction)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dto.Success(), nil
}
func (s *FeedbackService) List(ctx context.Context, req *dto.ListFeedbackReq, admin bool) (*dto.ListFeedbackResp, error) {
	actor, err := s.actor(ctx, admin)
	if err != nil {
		return nil, err
	}
	filter := bson.M{}
	if !admin {
		filter["userId"] = actor
	}
	if req.Status != "" {
		if req.Status != "pending" && req.Status != "answered" && req.Status != "closed" {
			return nil, errorx.New(errno.ErrFeedbackInvalid)
		}
		filter["status"] = req.Status
	}
	if req.Category != "" {
		if req.Category != "bug" && req.Category != "feature" && req.Category != "other" {
			return nil, errorx.New(errno.ErrFeedbackInvalid)
		}
		filter["category"] = req.Category
	}
	if req.Keyword != "" && admin {
		ids, err := s.ProposalRepo.Database().Collection(feedbackMessageCollection).Distinct(ctx, "feedbackId", bson.M{"text": primitive.Regex{Pattern: regexp.QuoteMeta(req.Keyword), Options: "i"}})
		if err != nil {
			return nil, err
		}
		filter["_id"] = bson.M{"$in": ids}
	}
	collection := s.ProposalRepo.Database().Collection(feedbackCollection)
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}
	cursor, err := collection.Find(ctx, filter, page.FindPageOption(req.PageParam).SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "_id", Value: -1}}))
	if err != nil {
		return nil, err
	}
	feedbacks := []*model.Feedback{}
	err = cursor.All(ctx, &feedbacks)
	cursor.Close(ctx)
	if err != nil {
		return nil, err
	}
	result := []*dto.FeedbackVO{}
	for _, feedback := range feedbacks {
		unread, err := s.unreadFor(ctx, feedback.ID, actor, admin)
		if err != nil {
			return nil, err
		}
		result = append(result, &dto.FeedbackVO{Feedback: feedback, UnreadCount: unread})
	}
	return &dto.ListFeedbackResp{Resp: dto.Success(), Total: total, Feedbacks: result}, nil
}
func (s *FeedbackService) Unread(ctx context.Context, admin bool) (*dto.FeedbackUnreadResp, error) {
	actor, err := s.actor(ctx, admin)
	if err != nil {
		return nil, err
	}
	filter := bson.M{}
	if !admin {
		filter["userId"] = actor
	}
	ids, err := s.ProposalRepo.Database().Collection(feedbackCollection).Distinct(ctx, "_id", filter)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, raw := range ids {
		id, ok := raw.(string)
		if !ok {
			continue
		}
		unread, err := s.unreadFor(ctx, id, actor, admin)
		if err != nil {
			return nil, err
		}
		total += unread
	}
	return &dto.FeedbackUnreadResp{Resp: dto.Success(), UnreadCount: total}, nil
}
