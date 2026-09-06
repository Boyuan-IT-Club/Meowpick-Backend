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
	Username        string `json:"username"`        // 当前昵称，未设置时为空字符串
	Avatar          string `json:"avatar"`          // 当前头像引用，未设置时为空字符串
	Contribution    int64  `json:"contribution"`    // 用户累计贡献值，用于计算每日提案上限
	DailyQuota      int64  `json:"dailyQuota"`      // 当前中国时区自然日内已创建的提案数量
	DailyQuotaLimit int64  `json:"dailyQuotaLimit"` // 按当前贡献值等级计算的每日提案上限
	CanEditUsername bool   `json:"canEditUsername"` // 当前是否允许设置或修改非空昵称；清空昵称不受30天冷却限制
}

// GetUsernameByUserIDResp 返回指定用户的昵称。
type GetUsernameByUserIDResp struct {
	*Resp
	Username string `json:"username"` // 目标用户昵称，尚未设置时为空字符串
}

// GetUsernameByUserIDReq 限定跨用户昵称查询所关联的公开提案。
type GetUsernameByUserIDReq struct {
	ProposalID string `form:"proposalId"` // 普通用户跨用户查询时必填，且必须属于目标用户并允许展示昵称
}

// UpdateUserProfileReq 更新当前登录用户的个人资料。
// 指针用于区分字段未传/null（保持原值）与空字符串（清空）。
type UpdateUserProfileReq struct {
	Username *string `json:"username"` // 省略或 null 保持不变；空字符串清空；非空昵称会去除首尾空白并校验长度、控制字符、唯一性和30天冷却
	Avatar   *string `json:"avatar"`   // 省略或 null 保持不变；空字符串清空；服务端仅保存引用字符串，不上传或校验 URL
}

// UpdateUserProfileResp 返回更新后的个人资料字段。
type UpdateUserProfileResp struct {
	*Resp
	Username string `json:"username"` // 更新后的昵称
	Avatar   string `json:"avatar"`   // 更新后的头像引用
}
