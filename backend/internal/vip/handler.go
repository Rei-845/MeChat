package vip

import (
	"strconv"

	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

// 创建 VIP 处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// 套餐列表
func (h *Handler) GetPlans(c *gin.Context) {
	response.OK(c, gin.H{"list": h.svc.GetPlans()})
}

// 创建订单
func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	order, err := h.svc.CreateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, order)
}

// 支付
func (h *Handler) Pay(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单 ID")
		return
	}
	userID := middleware.CurrentUserID(c)
	status, err := h.svc.Pay(c.Request.Context(), userID, orderID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, status)
}

// VIP 状态
func (h *Handler) GetStatus(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	status, err := h.svc.GetStatus(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, status)
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	vip := r.Group("/vip", authMiddleware)
	{
		vip.GET("/plans", h.GetPlans)
		vip.POST("/orders", h.CreateOrder)
		vip.POST("/orders/:id/pay", h.Pay)
		vip.GET("/status", h.GetStatus)
	}
}
