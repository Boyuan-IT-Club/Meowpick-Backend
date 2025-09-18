package cmd

type CourseInLinkVO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CourseVO 传递给前端的课程类型 模糊搜索和精确搜索结果都可用此类型
type CourseVO struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Code       string            `json:"code"` // 暂未使用
	Category   string            `json:"category"`
	Campuses   []string          `json:"campuses"`
	Department string            `json:"department"`
	Link       []*CourseInLinkVO `json:"link"`
	Teachers   []string          `json:"teachers"`
	TagCount   map[string]int    `json:"tagCount"` // TODO 修改前端字段
}

type ListCoursesReq struct {
	Keyword string `form:"keyword"`
	*PageParam
}

type ListCoursesResp struct {
	*Resp
	*PaginatedCourses
}

type GetOneCourseResp struct {
	*Resp
	Course *CourseVO `json:"course"`
}

type GetCoursesDepartmentsReq struct {
	Keyword string `form:"keyword"`
}

type GetCoursesDepartmentsResp struct {
	*Resp
	Departments []string `json:"departments"`
}

type GetCourseCategoriesReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseCategoriesResp struct {
	*Resp
	Categories []string `json:"categories"`
}

type GetCourseCampusesReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseCampusesResp struct {
	*Resp
	Campuses []string `json:"campuses"`
}

type PaginatedCourses struct {
	Courses []*CourseVO `json:"courses"` // 当前页的课程列表
	Total   int64       `json:"total"`   // 符合条件的总记录数
	*PageParam
}
