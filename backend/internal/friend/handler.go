package friend

import (
	"strconv"

	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

// 创建好友处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// 发送好友请求
func (h *Handler) SendRequest(c *gin.Context) {
	var req SendRequestReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	if req.ToUserID == userID {
		response.BadRequest(c, "不能添加自己为好友")
		return
	}
	if err := h.svc.SendRequest(c.Request.Context(), userID, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 待处理请求
func (h *Handler) GetRequests(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	list, err := h.svc.GetPendingRequests(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, gin.H{"list": list})
}

// 处理好友请求
func (h *Handler) HandleRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的申请 ID")
		return
	}
	var req HandleRequestReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	if err := h.svc.HandleRequest(c.Request.Context(), userID, id, req.Action); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// 好友列表
func (h *Handler) GetFriends(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	list, err := h.svc.GetFriendList(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

// 删好友
func (h *Handler) DeleteFriend(c *gin.Context) {
	friendID, err := strconv.ParseUint(c.Param("friend_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的好友 ID")
		return
	}
	userID := middleware.CurrentUserID(c)
	if err := h.svc.DeleteFriend(c.Request.Context(), userID, friendID); err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	friends := r.Group("/friends", authMiddleware)
	{
		friends.POST("/requests", h.SendRequest)
		friends.GET("/requests", h.GetRequests)
		friends.PUT("/requests/:id", h.HandleRequest)
		friends.GET("", h.GetFriends)
		friends.DELETE("/:friend_id", h.DeleteFriend)
	}
}
