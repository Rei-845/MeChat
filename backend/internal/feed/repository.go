package feed

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// 创建帖子仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// 创建帖子
func (r *Repository) CreatePost(ctx context.Context, p *Post) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// 查帖子
func (r *Repository) GetPost(ctx context.Context, id uint64) (*Post, error) {
	var p Post
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// 软删帖子 gorm.DeletedAt 自动写 deleted_at 并让后续查询自动过滤
func (r *Repository) DeletePost(ctx context.Context, id, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&Post{}).Error
}

// 多用户近期帖子
func (r *Repository) GetPostsByUserIDs(ctx context.Context, userIDs []uint64, since time.Time, limit int) ([]*Post, error) {
	var posts []*Post
	err := r.db.WithContext(ctx).
		Where("user_id IN ? AND created_at > ?", userIDs, since).
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

// 按 ID 批量取帖
func (r *Repository) GetPostsByIDs(ctx context.Context, ids []uint64) ([]*Post, error) {
	var posts []*Post
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&posts).Error
	return posts, err
}

// 全局最新帖子
func (r *Repository) GetRecentPosts(ctx context.Context, page, pageSize int) ([]*Post, error) {
	var posts []*Post
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	return posts, err
}

// Like

// 点赞帖子
func (r *Repository) LikePost(ctx context.Context, postID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&PostLike{PostID: postID, UserID: userID}).Error; err != nil {
			return err // 唯一索引冲突即已点赞
		}
		return tx.Model(&Post{}).Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	})
}

// 取消点赞
func (r *Repository) UnlikePost(ctx context.Context, postID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&PostLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return tx.Model(&Post{}).Where("id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		}
		return nil
	})
}

// 是否点赞
func (r *Repository) IsLiked(ctx context.Context, postID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

// 批量查点赞
func (r *Repository) BatchIsLiked(ctx context.Context, postIDs []uint64, userID uint64) (map[uint64]bool, error) {
	var likes []PostLike
	err := r.db.WithContext(ctx).
		Where("post_id IN ? AND user_id = ?", postIDs, userID).
		Find(&likes).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint64]bool, len(likes))
	for _, l := range likes {
		m[l.PostID] = true
	}
	return m, nil
}

// Comment

// 创建评论
func (r *Repository) CreateComment(ctx context.Context, c *PostComment) error {
	if err := r.db.WithContext(ctx).Create(c).Error; err != nil {
		return err
	}
	r.db.WithContext(ctx).Model(&Post{}).Where("id = ?", c.PostID).UpdateColumn("comment_count", gorm.Expr("comment_count + 1"))
	return nil
}

// 帖子评论分页
func (r *Repository) GetComments(ctx context.Context, postID uint64, page, pageSize int, sort string) ([]*PostComment, error) {
	order := "like_count DESC, created_at DESC" // 默认热度优先
	if sort == "time" {
		order = "created_at DESC" // 时间优先
	}
	var comments []*PostComment
	err := r.db.WithContext(ctx).
		Where("post_id = ? AND parent_id = 0", postID).
		Order(order).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error
	return comments, err
}

// 按标题搜索帖子
func (r *Repository) SearchPostsByTitle(ctx context.Context, keyword, sort string, page, pageSize int) ([]*Post, error) {
	order := "like_count DESC, comment_count DESC, created_at DESC"
	if sort == "time" {
		order = "created_at DESC"
	}
	var posts []*Post
	err := r.db.WithContext(ctx).
		Where("title LIKE ?", "%"+keyword+"%").
		Order(order).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	return posts, err
}

// 取根评论的回复
func (r *Repository) GetReplies(ctx context.Context, parentID uint64, limit int) ([]*PostComment, error) {
	var replies []*PostComment
	q := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("like_count DESC, created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return replies, q.Find(&replies).Error
}

// 分页取回复
func (r *Repository) GetRepliesPaged(ctx context.Context, parentID uint64, offset, limit int) ([]*PostComment, error) {
	var replies []*PostComment
	return replies, r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("like_count DESC, created_at ASC").
		Offset(offset).Limit(limit).
		Find(&replies).Error
}

// 软删评论
func (r *Repository) DeleteComment(ctx context.Context, commentID, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", commentID, userID).
		Delete(&PostComment{}).Error
}

// Comment Like

// 点赞评论
func (r *Repository) LikeComment(ctx context.Context, commentID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&CommentLike{CommentID: commentID, UserID: userID}).Error; err != nil {
			return err // 唯一索引冲突即已点赞
		}
		return tx.Model(&PostComment{}).Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	})
}

// 取消评论点赞
func (r *Repository) UnlikeComment(ctx context.Context, commentID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).Delete(&CommentLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return tx.Model(&PostComment{}).Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		}
		return nil
	})
}

// 批量查评论点赞
func (r *Repository) BatchCommentLiked(ctx context.Context, commentIDs []uint64, userID uint64) (map[uint64]bool, error) {
	if len(commentIDs) == 0 {
		return map[uint64]bool{}, nil
	}
	var likes []CommentLike
	err := r.db.WithContext(ctx).
		Where("comment_id IN ? AND user_id = ?", commentIDs, userID).
		Find(&likes).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint64]bool, len(likes))
	for _, l := range likes {
		m[l.CommentID] = true
	}
	return m, nil
}

// 某用户的帖子 limit 多取 1 条用于 hasMore
func (r *Repository) GetPostsByUser(ctx context.Context, userID uint64, page, pageSize, limit int) ([]*Post, error) {
	var posts []*Post
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(limit).
		Find(&posts).Error
	return posts, err
}
