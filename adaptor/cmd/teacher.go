package cmd

type GetTeachersReq struct {
	TeacherID string `form:"teacherId"`
	*PageParam
}

type GetTeachersResp struct {
	*Resp
	*PaginatedCourses
}
