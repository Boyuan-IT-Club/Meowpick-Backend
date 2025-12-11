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

import (
	"time"
)

type ChangeLog struct {
	ID           string    `bson:"_id,omitempty"         json:"-"`
	TargetID     string    `bson:"targetId"              json:"-"`
	TargetType   int32     `bson:"targetType"            json:"-"`
	Like         int32     `bson:"action"                json:"-"`
	Content      string    `bson:"content"               json:"-"`
	UpdateSource int32     `bson:"updateSource"          json:"-"`
	ProposalID   string    `bson:"proposalId,omitempty"  json:"-"`
	UpdatedAt    time.Time `bson:"updatedAt"             json:"-"`
}
