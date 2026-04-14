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

package mapping

import "github.com/Boyuan-IT-Club/Meowpick-Backend/types/consts"

var ChangeLogTargetTypeMap = map[int32]string{
	1: consts.ChangeLogTargetTypeCourse,
	2: consts.ChangeLogTargetTypeProposal,
	3: consts.ChangeLogTargetTypeTeacher,
	4: consts.ChangeLogTargetTypeUser,
}
