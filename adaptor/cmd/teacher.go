package cmd

type GetTeachersReq struct {
	TeacherID string `form:"teacherID"`
	*PageParam
}

type GetTeachersResp struct {
	*Resp
	*PaginatedCourses
}
