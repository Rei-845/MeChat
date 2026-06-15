package level

import (
	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

// 创建等级处理器
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// 我的等级
func (h *Handler) GetMyLevel(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	info, err := h.svc.GetLevelInfo(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "获取等级失败")
		return
	}
	response.OK(c, info)
}

// 签到
func (h *Handler) Checkin(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	u, err := h.svc.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "获取用户信息失败")
		return
	}
	xp, done := h.svc.Checkin(c.Request.Context(), userID, u.IsVIP())
	if done {
		response.OK(c, gin.H{"already_done": true, "xp_gained": 0})
		return
	}
	response.OK(c, gin.H{"already_done": false, "xp_gained": xp})
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	g := r.Group("/level", authMiddleware)
	g.GET("/me", h.GetMyLevel)
	g.POST("/checkin", h.Checkin)
}
