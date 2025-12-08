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

type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func Success() *Resp {
	return &Resp{
		Code: 0,
		Msg:  "success",
	}
}

type PageParam struct {
	Page     int64 `form:"page" json:"page"`
	PageSize int64 `form:"pageSize" json:"pageSize"`
}

type IPageParam interface {
	UnWrap() (int64, int64)
}

func (p *PageParam) UnWrap() (int64, int64) {
	if p.Page < 0 {
		p.Page = 0
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 10
	}

	return p.Page, p.PageSize
}
