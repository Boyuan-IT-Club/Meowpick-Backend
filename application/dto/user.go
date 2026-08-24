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

package dto

// GetUserProfileResp 获取当前登录用户的个人资料。
type GetUserProfileResp struct {
	*Resp
	Username        string `json:"username"`
	Avatar          string `json:"avatar"`
	Contribution    int64  `json:"contribution"`
	DailyQuota      int64  `json:"dailyQuota"`
	DailyQuotaLimit int64  `json:"dailyQuotaLimit"`
	CanEditUsername bool   `json:"canEditUsername"`
}

// GetUsernameByUserIDResp 返回指定用户的昵称。
type GetUsernameByUserIDResp struct {
	*Resp
	Username string `json:"username"`
}

// UpdateUserProfileReq 更新当前登录用户的个人资料。
// 指针用于区分字段未传/null（保持原值）与空字符串（清空）。
type UpdateUserProfileReq struct {
	Username *string `json:"username"`
	Avatar   *string `json:"avatar"`
}

// UpdateUserProfileResp 返回更新后的个人资料字段。
type UpdateUserProfileResp struct {
	*Resp
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}
