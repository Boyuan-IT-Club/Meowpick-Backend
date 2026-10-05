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

package errno

import "github.com/Boyuan-IT-Club/go-kit/errorx/code"

const (
	ErrFeedbackNotFound   = 110000001
	ErrFeedbackInvalid    = 110000002
	ErrFeedbackRateLimit  = 110000003
	ErrFeedbackDailyLimit = 110000004
)

func init() {
	for value, message := range map[int32]string{
		ErrFeedbackNotFound:   "feedback not found",
		ErrFeedbackInvalid:    "invalid feedback input",
		ErrFeedbackRateLimit:  "at most 5 feedback messages per minute",
		ErrFeedbackDailyLimit: "at most 10 new feedback entries per day",
	} {
		code.Register(value, message, code.WithAffectStability(false))
	}
}
