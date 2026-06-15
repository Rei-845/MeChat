package feed

import (
	"strconv"
	"strings"

	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

// 创建 Feed 处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// 发帖
func (h *Handler) CreatePost(c *gin.Context) {
	var req CreatePostReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	post, err := h.svc.CreatePost(c.Request.Context(), userID, &req, c.ClientIP())
	if err != nil {
		response.ServerError(c, "发帖失败")
		return
	}
	response.OK(c, post)
}

// 删帖
func (h *Handler) DeletePost(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := middleware.CurrentUserID(c)
	if err := h.svc.DeletePost(c.Request.Context(), userID, postID); err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}

// 主页 Feed
func (h *Handler) GetFeed(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	userID := middleware.CurrentUserID(c)
	posts, hasMore, err := h.svc.GetFeed(c.Request.Context(), userID, page, pageSize, c.Query("sort"))
	if err != nil {
		response.ServerError(c, "获取 Feed 失败")
		return
	}
	response.OK(c, gin.H{"list": posts, "has_more": hasMore})
}

// 搜索帖子
func (h *Handler) SearchPosts(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		response.OK(c, gin.H{"list": []any{}, "has_more": false})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	userID := middleware.CurrentUserID(c)
	posts, hasMore, err := h.svc.SearchPosts(c.Request.Context(), userID, keyword, c.Query("sort"), page, pageSize)
	if err != nil {
		response.ServerError(c, "搜索失败")
		return
	}
	response.OK(c, gin.H{"list": posts, "has_more": hasMore})
}

// 热门帖子
func (h *Handler) GetHotPosts(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	posts, err := h.svc.GetHotPosts(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "获取热门失败")
		return
	}
	response.OK(c, gin.H{"list": posts})
}

// 单帖
func (h *Handler) GetPost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的帖子 ID")
		return
	}
	userID := middleware.CurrentUserID(c)
	post, err := h.svc.GetPost(c.Request.Context(), postID, userID)
	if err != nil {
		response.NotFound(c, "帖子不存在")
		return
	}
	response.OK(c, post)
}

// 点赞
func (h *Handler) LikePost(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := middleware.CurrentUserID(c)
	h.svc.LikePost(c.Request.Context(), userID, postID)
	response.OK(c, nil)
}

// 取消点赞
func (h *Handler) UnlikePost(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := middleware.CurrentUserID(c)
	h.svc.UnlikePost(c.Request.Context(), userID, postID)
	response.OK(c, nil)
}

// 评论列表
func (h *Handler) GetComments(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := middleware.CurrentUserID(c)
	comments, hasMore, err := h.svc.GetComments(c.Request.Context(), userID, postID, page, pageSize, c.Query("sort"))
	if err != nil {
		response.ServerError(c, "查询评论失败")
		return
	}
	response.OK(c, gin.H{"list": comments, "has_more": hasMore})
}

// 回复列表
func (h *Handler) GetCommentReplies(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	commentID, _ := strconv.ParseUint(c.Param("cid"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	userID := middleware.CurrentUserID(c)
	replies, hasMore, err := h.svc.GetCommentReplies(c.Request.Context(), userID, postID, commentID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取回复失败")
		return
	}
	response.OK(c, gin.H{"list": replies, "has_more": hasMore})
}

// 点赞评论
func (h *Handler) LikeComment(c *gin.Context) {
	commentID, _ := strconv.ParseUint(c.Param("cid"), 10, 64)
	userID := middleware.CurrentUserID(c)
	h.svc.LikeComment(c.Request.Context(), userID, commentID)
	response.OK(c, nil)
}

// 取消评论点赞
func (h *Handler) UnlikeComment(c *gin.Context) {
	commentID, _ := strconv.ParseUint(c.Param("cid"), 10, 64)
	userID := middleware.CurrentUserID(c)
	h.svc.UnlikeComment(c.Request.Context(), userID, commentID)
	response.OK(c, nil)
}

// 用户帖子
func (h *Handler) GetUserPosts(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	viewerID := middleware.CurrentUserID(c)
	posts, hasMore, err := h.svc.GetUserPosts(c.Request.Context(), viewerID, targetID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取失败")
		return
	}
	response.OK(c, gin.H{"list": posts, "has_more": hasMore})
}

// 我的帖子
func (h *Handler) GetMyPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	userID := middleware.CurrentUserID(c)
	posts, hasMore, err := h.svc.GetMyPosts(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取失败")
		return
	}
	response.OK(c, gin.H{"list": posts, "has_more": hasMore})
}

// 发评论
func (h *Handler) CreateComment(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req CreateCommentReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	comment, xpGained, err := h.svc.CreateComment(c.Request.Context(), userID, postID, &req)
	if err != nil {
		response.ServerError(c, "评论失败")
		return
	}
	response.OK(c, gin.H{"comment": comment, "xp_gained": xpGained})
}

// 删评论
func (h *Handler) DeleteComment(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	commentID, _ := strconv.ParseUint(c.Param("cid"), 10, 64)
	userID := middleware.CurrentUserID(c)
	h.svc.DeleteComment(c.Request.Context(), userID, postID, commentID)
	response.OK(c, nil)
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	posts := r.Group("/posts", authMiddleware)
	{
		posts.POST("", h.CreatePost)
		posts.GET("/feed", h.GetFeed)
		posts.GET("/search", h.SearchPosts)
		posts.GET("/hot", h.GetHotPosts)
		posts.GET("/mine", h.GetMyPosts)
		posts.GET("/user/:uid", h.GetUserPosts)
		posts.GET("/:id", h.GetPost)
		posts.DELETE("/:id", h.DeletePost)
		posts.POST("/:id/like", h.LikePost)
		posts.DELETE("/:id/like", h.UnlikePost)
		posts.GET("/:id/comments", h.GetComments)
		posts.POST("/:id/comments", h.CreateComment)
		posts.DELETE("/:id/comments/:cid", h.DeleteComment)
		posts.GET("/:id/comments/:cid/replies", h.GetCommentReplies)
		posts.POST("/:id/comments/:cid/like", h.LikeComment)
		posts.DELETE("/:id/comments/:cid/like", h.UnlikeComment)
	}
}
