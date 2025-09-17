package dto

import (
	"context"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/teacher"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/adaptor/cmd"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/mapper/like"
	"github.com/google/wire"
)

type ICommentDTO interface {
	// ToCommentVO 单个Comment转CommentVO (DB to VO) 包含点赞信息查询
	ToCommentVO(ctx context.Context, c *comment.Comment) (*cmd.CommentVO, error)
	// ToComment 单个CommentVO转Comment (VO to DB)
	ToComment(ctx context.Context, vo *cmd.CommentVO) (*comment.Comment, error)
	// TOMyCommentVO 单个Comment转MyCommentVO(with 4 Extra fields) (DB to VO)
	TOMyCommentVO(ctx context.Context, c *comment.Comment) (*cmd.CommentVO, error)
	// ToMyCommentVOList Comment数组转MyCommentVO数组(with 4 extra fields) (DB Array to VO Array)
	ToMyCommentVOList(ctx context.Context, comments []*comment.Comment) ([]*cmd.CommentVO, error)
	// ToCommentVOList Comment数组转CommentVO数组 (DB Array to VO Array)
	ToCommentVOList(ctx context.Context, comments []*comment.Comment) ([]*cmd.CommentVO, error)
	// ToCommentList CommentVO数组转Comment数组 (VO Array to DB Array)
	ToCommentList(ctx context.Context, vos []*cmd.CommentVO) ([]*comment.Comment, error)
}

type CommentDTO struct {
	LikeMapper    *like.MongoMapper
	CourseMapper  *course.MongoMapper
	TeacherMapper *teacher.MongoMapper
	StaticData    *consts.StaticData
}

var CommentDTOSet = wire.NewSet(
	wire.Struct(new(CommentDTO), "*"),
	wire.Bind(new(ICommentDTO), new(*CommentDTO)),
)

// 单个Comment转CommentVO (DB to VO) 包含点赞信息查询
func (d *CommentDTO) ToCommentVO(ctx context.Context, c *comment.Comment) (*cmd.CommentVO, error) {
	userID := c.UserID
	// 获取点赞信息
	likeCnt, err := d.LikeMapper.GetLikeCount(ctx, c.ID, consts.CommentType)
	if err != nil {
		logx.Error(ctx, "GetLikeCount failed for commentID=%s: %v", c.ID, err)
		return nil, err
	}

	active, err := d.LikeMapper.GetLikeStatus(ctx, userID, c.ID, consts.CommentType)
	if err != nil {
		logx.Error(ctx, "GetLikeStatus failed for userID=%s, commentID=%s: %v", userID, c.ID, err)
		return nil, err
	}

	return &cmd.CommentVO{
		ID:       c.ID,
		Content:  c.Content,
		Tags:     c.Tags,
		UserID:   c.UserID,
		CourseID: c.CourseID,
		LikeVO: &cmd.LikeVO{
			Like:    active,
			LikeCnt: likeCnt,
		},
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

// 单个CommentVO转Comment (VO to DB)
func (d *CommentDTO) ToComment(ctx context.Context, vo *cmd.CommentVO) (*comment.Comment, error) {
	if vo == nil {
		return nil, nil
	}

	return &comment.Comment{
		ID:        vo.ID,
		Content:   vo.Content,
		Tags:      vo.Tags,
		UserID:    vo.UserID,
		CourseID:  vo.CourseID,
		CreatedAt: vo.CreatedAt,
		UpdatedAt: vo.UpdatedAt,
		Deleted:   false, // 默认为未删除
	}, nil
}

// 单个Comment转MyCommentVO(with 4 Extra fields) (DB to VO)
func (d *CommentDTO) TOMyCommentVO(ctx context.Context, c *comment.Comment) (*cmd.CommentVO, error) {
	// 先获取除了Extra以外的字段
	myCommentVO, err := d.ToCommentVO(ctx, c)
	if err != nil {
		return nil, err
	}

	// 获取Extra course相关
	var cou *course.Course
	cou, err = d.CourseMapper.FindOneByID(ctx, c.CourseID)
	if err != nil {
		return nil, err
	}

	// 获取Extra teacher相关
	var teachersNameAndTitle []string
	for _, teacherID := range cou.TeacherIDs {
		t, err := d.TeacherMapper.FindOneTeacherByID(ctx, teacherID)
		if err != nil {
			continue
		}
		teachersNameAndTitle = append(teachersNameAndTitle, t.Name+t.Title)
	}
	// 组合rawVO和Extra得到MyCommentVO
	myCommentVO.Name = cou.Name
	myCommentVO.Category = d.StaticData.GetCategoryNameByID(cou.Category)
	myCommentVO.Department = d.StaticData.GetDepartmentNameByID(cou.Department)
	myCommentVO.Teachers = teachersNameAndTitle

	return myCommentVO, nil
}

// Comment数组转CommentVO数组 (DB Array to VO Array)
func (d *CommentDTO) ToCommentVOList(ctx context.Context, comments []*comment.Comment) ([]*cmd.CommentVO, error) {
	if len(comments) == 0 {
		logx.Error(ctx, "ToCommentVOList: comments is empty")
		return []*cmd.CommentVO{}, nil
	}

	commentVOs := make([]*cmd.CommentVO, 0, len(comments))

	for _, c := range comments {
		commentVO, err := d.ToCommentVO(ctx, c)
		if err != nil {
			return nil, err
		}
		if commentVO != nil {
			commentVOs = append(commentVOs, commentVO)
		}
	}

	return commentVOs, nil
}

// Comment数组转MyCommentVO数组(with 4 extra fields) (DB Array to VO Array)
func (d *CommentDTO) ToMyCommentVOList(ctx context.Context, comments []*comment.Comment) ([]*cmd.CommentVO, error) {
	if len(comments) == 0 {
		logx.Error(ctx, "ToCommentVOList: comments is empty")
		return []*cmd.CommentVO{}, nil
	}
	commentVOs := make([]*cmd.CommentVO, 0, len(comments))

	for _, c := range comments {
		commentVO, err := d.TOMyCommentVO(ctx, c)
		if err != nil {
			return nil, err
		}
		if commentVO != nil {
			commentVOs = append(commentVOs, commentVO)
		}
	}

	return commentVOs, nil
}

// CommentVO数组转Comment数组 (VO Array to DB Array)
func (d *CommentDTO) ToCommentList(ctx context.Context, vos []*cmd.CommentVO) ([]*comment.Comment, error) {
	if len(vos) == 0 {
		return []*comment.Comment{}, nil
	}

	comments := make([]*comment.Comment, 0, len(vos))

	for _, vo := range vos {
		dbComment, err := d.ToComment(ctx, vo)
		if err != nil {
			return nil, err
		}
		if dbComment != nil {
			comments = append(comments, dbComment)
		}
	}

	return comments, nil
}
