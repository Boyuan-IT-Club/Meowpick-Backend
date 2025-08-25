package cmd

type CourseQueryCmd struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=10"`
}

type PaginatedCourses struct {
	List  []CourseInList `json:"list"`  // 当前页的课程列表
	Total int64          `json:"total"` // 符合条件的总记录数
	Page  int            `json:"page"`  // 当前页码
	Size  int            `json:"size"`  // 每页数量
}

type PaginatedCoursesResp struct {
	Code int               `json:"-"`
	Msg  string            `json:"-"`
	Page *PaginatedCourses `json:"page"`
}

type CourseInList struct {
	ID             string   `json:"_id"`
	Name           string   `json:"name"`
	Code           string   `json:"code"`
	DepartmentName string   `json:"department_name"`
	CategoriesName string   `json:"categories_name"`
	CampusesName   []string `json:"campuses_name"`
} //只包含要传前端展示的字段
