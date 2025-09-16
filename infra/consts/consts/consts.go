package consts

var PageSize int64 = 10

// 数据库相关
const (
	ID         = "_id"
	Status     = "status"
	CreatedAt  = "createdAt"
	UpdatedAt  = "updatedAt"
	UserId     = "userId"
	Query      = "query"
	Deleted    = "deleted"
	TargetId   = "targetId"
	Active     = "active"
	CourseId   = "courseId"
	OpenId     = "openId"
	TeacherIds = "teacherIds"
	Categories = "categories"
	Department = "department"
	Campuses   = "campuses"
	Code       = "code"
	Name       = "name"
)

// 元素类别相关（如课程、评论、老师）
const (
	CourseType int32 = 101 + iota
	CommentType
)

// 业务相关
const (
	ContextUserID = "userID"
	ContextTarget = "targetID"
	ContextToken  = "token"
)

// 限制相关
const (
	SearchHistoryLimit = 15
)
