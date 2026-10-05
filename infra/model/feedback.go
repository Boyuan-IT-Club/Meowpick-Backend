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

package model

import "time"

type Feedback struct {
	ID        string    `bson:"_id" json:"id"`
	UserID    string    `bson:"userId" json:"userId"`
	Category  string    `bson:"category" json:"category"`
	Status    string    `bson:"status" json:"status"`
	Sequence  int64     `bson:"sequence" json:"sequence"`
	Summary   string    `bson:"summary" json:"summary"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type FeedbackMessage struct {
	ID         string    `bson:"_id" json:"id"`
	FeedbackID string    `bson:"feedbackId" json:"feedbackId"`
	UserID     string    `bson:"userId" json:"userId"`
	Role       string    `bson:"role" json:"role"` // author/admin
	Sequence   int64     `bson:"sequence" json:"sequence"`
	Text       string    `bson:"text" json:"text"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
}
