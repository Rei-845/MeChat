package chat

import (
	"path/filepath"
	"strconv"

	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

// 创建会话处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// 会话列表
func (h *Handler) GetConversations(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	list, err := h.svc.GetConversations(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, gin.H{"list": list})
}

// 取或建私聊
func (h *Handler) CreatePrivateConv(c *gin.Context) {
	var req CreatePrivateConvReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	info, err := h.svc.GetOrCreatePrivateConvInfo(c.Request.Context(), userID, req.TargetUserID)
	if err != nil {
		response.ServerError(c, "创建会话失败")
		return
	}
	response.OK(c, info)
}

// 建群
func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	info, err := h.svc.CreateGroupInfo(c.Request.Context(), userID, &req)
	if err != nil {
		response.ServerError(c, "创建群聊失败")
		return
	}
	response.OK(c, info)
}

// 消息列表
func (h *Handler) GetMessages(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会话 ID")
		return
	}

	var req GetMessagesReq
	c.ShouldBindQuery(&req)

	userID := middleware.CurrentUserID(c)
	msgs, hasMore, err := h.svc.GetMessages(c.Request.Context(), userID, convID, req.BeforeID, req.Limit)
	if err != nil {
		if err == ErrNotMember {
			response.Forbidden(c, err.Error())
		} else {
			response.ServerError(c, "查询消息失败")
		}
		return
	}

	var nextCursor string
	if hasMore && len(msgs) > 0 {
		nextCursor = msgs[len(msgs)-1].MsgID
	}
	response.OK(c, gin.H{
		"list":        msgs,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	})
}

// 标记已读
func (h *Handler) MarkRead(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会话 ID")
		return
	}
	var body struct {
		MsgID uint64 `json:"msg_id"`
	}
	c.ShouldBindJSON(&body)
	userID := middleware.CurrentUserID(c)
	h.svc.MarkRead(c.Request.Context(), userID, convID, body.MsgID)
	response.OK(c, nil)
}

// 撤回消息
func (h *Handler) RecallMessage(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的消息 ID")
		return
	}
	userID := middleware.CurrentUserID(c)
	if err := h.svc.RecallMessage(c.Request.Context(), userID, msgID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 群详情
func (h *Handler) GetGroupDetail(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的群 ID")
		return
	}
	group, err := h.svc.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		response.NotFound(c, "群不存在")
		return
	}
	response.OK(c, group)
}

// 加群成员
func (h *Handler) AddGroupMembers(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会话 ID")
		return
	}
	var body struct {
		UserIDs []uint64 `json:"user_ids" binding:"required"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	userID := middleware.CurrentUserID(c)
	if err := h.svc.AddGroupMembers(c.Request.Context(), userID, convID, body.UserIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 移除群成员
func (h *Handler) RemoveGroupMember(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	targetID, _ := strconv.ParseUint(c.Param("uid"), 10, 64)
	userID := middleware.CurrentUserID(c)
	if err := h.svc.RemoveGroupMember(c.Request.Context(), userID, convID, targetID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 退群
func (h *Handler) LeaveGroup(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := middleware.CurrentUserID(c)
	if err := h.svc.RemoveGroupMember(c.Request.Context(), userID, convID, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 解散群
func (h *Handler) DisbandGroup(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := middleware.CurrentUserID(c)
	if err := h.svc.DisbandGroup(c.Request.Context(), userID, convID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 上传群头像
func (h *Handler) UploadGroupAvatar(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的群 ID")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "缺少文件")
		return
	}
	defer file.Close()
	userID := middleware.CurrentUserID(c)
	url, err := h.svc.UpdateGroupAvatar(c.Request.Context(), userID, groupID, file, header.Size, filepath.Base(header.Filename))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url})
}

// 群成员列表
func (h *Handler) GetGroupMembers(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的群 ID")
		return
	}
	userID := middleware.CurrentUserID(c)
	members, err := h.svc.GetGroupMembersInfo(c.Request.Context(), userID, groupID)
	if err != nil {
		response.ServerError(c, "获取成员失败")
		return
	}
	response.OK(c, gin.H{"list": members})
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	convs := r.Group("/conversations", authMiddleware)
	{
		convs.GET("", h.GetConversations)
		convs.POST("", h.CreatePrivateConv)
		convs.POST("/group", h.CreateGroup)
		convs.GET("/:id/messages", h.GetMessages)
		convs.PUT("/:id/read", h.MarkRead)
		convs.POST("/:id/members", h.AddGroupMembers)
		convs.DELETE("/:id/members/:uid", h.RemoveGroupMember)
		convs.DELETE("/:id/members/me", h.LeaveGroup)
		convs.POST("/:id/disband", h.DisbandGroup)
	}

	msgs := r.Group("/messages", authMiddleware)
	{
		msgs.POST("/:id/recall", h.RecallMessage)
	}

	groups := r.Group("/groups", authMiddleware)
	{
		groups.GET("/:id", h.GetGroupDetail)
		groups.POST("/:id/avatar", h.UploadGroupAvatar)
		groups.GET("/:id/members", h.GetGroupMembers)
	}
}
