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

// auth: 100 000 000 ~ 100 999 999

const (
	ErrUserNotLogin = 100000001
	ErrUserNotAdmin = 100000002
)

func init() {
	code.Register(
		ErrUserNotLogin,
		"user not login",
		code.WithAffectStability(false),
	)
}
