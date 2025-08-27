package cmd

type GetTeachersReq struct {
	TeacherID string `form:"teacherID"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"pageSize,default=10"`
}

type GetTeachersResp struct {
	*Resp
	Page *PaginatedCourses `json:"page"`
}
