package cmd

type TeacherVO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Department string `json:"department"`
}
type GetTeachersReq struct {
	TeacherID string `form:"teacherId"`
	*PageParam
}

type GetTeachersResp struct {
	*Resp
	*PaginatedCourses
}

type AddNewTeacherReq struct {
	Name       string `json:"name" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Department string `json:"department" binding:"required"`
}

type AddNewTeacherResp struct {
	*Resp
	*TeacherVO
}
