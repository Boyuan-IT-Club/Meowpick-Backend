package assembler

import (
	"context"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/application/dto"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/consts"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/consts/mapping"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/comment"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/course"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/like"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/repo/teacher"
	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/util/log"
	"github.com/google/wire"
)

var _ ICommentDTO = (*CommentDTO)(nil)

type ICommentDTO interface {
	// ToCommentVO 单个Comment转CommentVO (DB to VO) 包含点赞信息查询
	ToCommentVO(ctx context.Context, c *comment.Comment, userID string) (*dto.CommentVO, error)
	// ToComment 单个CommentVO转Comment (VO to DB)
	ToComment(ctx context.Context, vo *dto.CommentVO) (*comment.Comment, error)
	// TOMyCommentVO 单个Comment转MyCommentVO(with 4 Extra fields) (DB to VO)
	TOMyCommentVO(ctx context.Context, c *comment.Comment, userID string) (*dto.CommentVO, error)
	// ToMyCommentVOList Comment数组转MyCommentVO数组(with 4 extra fields) (DB Array to VO Array)
	ToMyCommentVOList(ctx context.Context, comments []*comment.Comment, userID string) ([]*dto.CommentVO, error)
	// ToCommentVOList Comment数组转CommentVO数组 (DB Array to VO Array)
	ToCommentVOList(ctx context.Context, comments []*comment.Comment, userID string) ([]*dto.CommentVO, error)
	// ToCommentList CommentVO数组转Comment数组 (VO Array to DB Array)
	ToCommentList(ctx context.Context, vos []*dto.CommentVO) ([]*comment.Comment, error)
}

type CommentDTO struct {
	LikeMapper    *like.MongoRepo
	CourseMapper  *course.MongoRepo
	TeacherMapper *teacher.MongoRepo
	StaticData    *mapping.StaticData
}

var CommentDTOSet = wire.NewSet(
	wire.Struct(new(CommentDTO), "*"),
	wire.Bind(new(ICommentDTO), new(*CommentDTO)),
)

// 单个Comment转CommentVO (DB to VO) 包含点赞信息查询
func (d *CommentDTO) ToCommentVO(ctx context.Context, c *comment.Comment, userID string) (*dto.CommentVO, error) {
	// 获取点赞信息
	likeCnt, err := d.LikeMapper.GetLikeCount(ctx, c.ID, consts.CommentType)
	if err != nil {
		log.CtxError(ctx, "GetLikeCount failed for commentID=%s: %v", c.ID, err)
		return nil, err
	}

	// 这里的userID是查看评论的用户，而非评论作者
	active, err := d.LikeMapper.GetLikeStatus(ctx, userID, c.ID, consts.CommentType)
	if err != nil {
		log.CtxError(ctx, "GetLikeStatus failed for userID=%s, commentID=%s: %v", userID, c.ID, err)
		return nil, err
	}

	return &dto.CommentVO{
		ID:       c.ID,
		Content:  c.Content,
		Tags:     c.Tags,
		UserID:   c.UserID,
		CourseID: c.CourseID,
		LikeVO: &dto.LikeVO{
			Like:    active,
			LikeCnt: likeCnt,
		},
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

// 单个CommentVO转Comment (VO to DB)
func (d *CommentDTO) ToComment(ctx context.Context, vo *dto.CommentVO) (*comment.Comment, error) {
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
func (d *CommentDTO) TOMyCommentVO(ctx context.Context, c *comment.Comment, userID string) (*dto.CommentVO, error) {
	// 先获取除了Extra以外的字段
	myCommentVO, err := d.ToCommentVO(ctx, c, userID)
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

// Comment数组转MyCommentVO数组(with 4 extra fields) (DB Array to VO Array)
func (d *CommentDTO) ToMyCommentVOList(ctx context.Context, comments []*comment.Comment, userID string) ([]*dto.CommentVO, error) {
	if len(comments) == 0 {
		log.CtxError(ctx, "ToMyCommentVOList: comments is empty")
		return []*dto.CommentVO{}, nil
	}
	commentVOs := make([]*dto.CommentVO, 0, len(comments))

	for _, c := range comments {
		commentVO, err := d.TOMyCommentVO(ctx, c, userID)
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
func (d *CommentDTO) ToCommentList(ctx context.Context, vos []*dto.CommentVO) ([]*comment.Comment, error) {
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

// Comment数组转CommentVO数组 (DB Array to VO Array)
func (d *CommentDTO) ToCommentVOList(ctx context.Context, comments []*comment.Comment, userID string) ([]*dto.CommentVO, error) {
	if len(comments) == 0 {
		log.CtxInfo(ctx, "ToCommentVOList: comments is empty")
		return []*dto.CommentVO{}, nil
	}

	// 提取所有 commentID
	commentIDs := make([]string, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}

	// 批量获取点赞数
	likeCountMap, err := d.LikeMapper.GetBatchLikeCount(ctx, userID, commentIDs, consts.CommentType)
	if err != nil {
		log.CtxError(ctx, "GetBatchLikeCount failed: %v", err)
		return nil, err
	}

	// 批量获取点赞状态
	likeStatusMap, err := d.LikeMapper.GetBatchLikeStatus(ctx, userID, commentIDs, consts.CommentType)
	if err != nil {
		log.CtxError(ctx, "GetBatchLikeStatus failed: %v", err)
		return nil, err
	}

	// 构建结果
	commentVOs := make([]*dto.CommentVO, 0, len(comments))
	for _, c := range comments {
		// 从批量查询结果中获取点赞信息
		likeCnt := likeCountMap[c.ID] // 如果不存在则为0
		active := likeStatusMap[c.ID] // 如果不存在则为false

		commentVO := &dto.CommentVO{
			ID:       c.ID,
			Content:  c.Content,
			Tags:     c.Tags,
			UserID:   c.UserID,
			CourseID: c.CourseID,
			LikeVO: &dto.LikeVO{
				Like:    active,
				LikeCnt: likeCnt,
			},
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
		commentVOs = append(commentVOs, commentVO)
	}

	return commentVOs, nil
}
