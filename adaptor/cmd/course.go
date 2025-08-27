package cmd

type GetCoursesReq struct {
	Keyword string `form:"keyword"`
	*PageParam
}

type GetCoursesDepartsReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseCategoriesReq struct {
	Keyword string `form:"keyword"`
}

type GetCourseCampusesReq struct {
	Keyword string `form:"keyword"`
}

type GetCoursesResp struct {
	*Resp
	*PaginatedCourses
}

type GetCoursesDepartsResp struct {
	*Resp
	Departs []string `json:"departs"`
}

type GetCourseCategoriesResp struct {
	*Resp
	Categories []string `json:"categories"`
}

type GetCourseCampusesResp struct {
	*Resp
	Campuses []string `json:"campuses"`
}

type PaginatedCourses struct {
	List  []CourseInList `json:"list"`  // 当前页的课程列表
	Total int64          `json:"total"` // 符合条件的总记录数
	*PageParam
}

type CourseInList struct {
	ID             string   `json:"_id"`
	Name           string   `json:"name"`
	Code           string   `json:"code"`
	DepartmentName string   `json:"department_name"`
	CategoriesName string   `json:"categories_name"`
	CampusesName   []string `json:"campuses_name"`
	TeachersName   []string `json:"teachers_name"`
} //只包含要传前端展示的字段
