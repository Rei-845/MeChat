package ai

import (
	"encoding/json"

	"mechat/internal/ai/agent"
	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

// 创建 AI 处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// 映射 AI 错误
func agentErr(c *gin.Context, err error) {
	switch err {
	case ErrVIPRequired:
		response.Forbidden(c, err.Error())
	default:
		response.ServerError(c, "AI 处理失败: "+err.Error())
	}
}

// Summarize 提交总结任务 异步返回 async 同步返回 result
func (h *Handler) Summarize(c *gin.Context) {
	var req SummarizeReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	queued, result, err := h.svc.SubmitSummarize(c.Request.Context(), userID, &req)
	if err != nil {
		agentErr(c, err)
		return
	}
	if queued {
		response.OK(c, gin.H{"async": true})
		return
	}
	response.OK(c, gin.H{"result": result})
}

// sseEmit 写一条 SSE 并 flush
func sseEmit(c *gin.Context, ev agent.Event) {
	b, _ := json.Marshal(ev)
	c.Writer.Write([]byte("data: "))
	c.Writer.Write(b)
	c.Writer.Write([]byte("\n\n"))
	if f, ok := c.Writer.(interface{ Flush() }); ok {
		f.Flush()
	}
}

// setupSSE 设置 SSE 响应头
func setupSSE(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(200)
}

// StreamDraftMessage SSE 帮写消息 仅 VIP
func (h *Handler) StreamDraftMessage(c *gin.Context) {
	var req DraftMessageReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	if err := h.svc.checkVIP(c.Request.Context(), userID); err != nil {
		agentErr(c, err)
		return
	}
	setupSSE(c)
	emit := func(ev agent.Event) { sseEmit(c, ev) }
	h.svc.StreamDraftMessage(c.Request.Context(), userID, &req, emit)
	emit(agent.Event{Type: agent.EventDone})
}

// StreamDraftPost SSE 帮写帖子 仅 VIP
func (h *Handler) StreamDraftPost(c *gin.Context) {
	var req DraftPostReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	if err := h.svc.checkVIP(c.Request.Context(), userID); err != nil {
		agentErr(c, err)
		return
	}
	setupSSE(c)
	emit := func(ev agent.Event) { sseEmit(c, ev) }
	h.svc.StreamDraftPost(c.Request.Context(), userID, &req, emit)
	emit(agent.Event{Type: agent.EventDone})
}

// StreamChat SSE 流式 Agent 对话
func (h *Handler) StreamChat(c *gin.Context) {
	var req ChatReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	_, events, err := h.svc.Chat(userID, req.Text)
	if err != nil {
		agentErr(c, err)
		return
	}
	setupSSE(c)
	done := c.Request.Context().Done()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				sseEmit(c, agent.Event{Type: agent.EventDone})
				return
			}
			sseEmit(c, ev)
		case <-done:
			go func() { // 客户端断开 排空通道让后台生成跑完落库
				for range events {
				}
			}()
			return
		}
	}
}

// GetHistory 返回当前用户的 AI 对话历史
func (h *Handler) GetHistory(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	msgs, err := h.svc.History(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "加载失败")
		return
	}
	response.OK(c, gin.H{"messages": msgs})
}

// ClearHistory 清空当前用户的 AI 对话
func (h *Handler) ClearHistory(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	if err := h.svc.ClearHistory(c.Request.Context(), userID); err != nil {
		response.ServerError(c, "清空失败")
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// CancelAction 取消待确认写操作 清掉落库的确认框
func (h *Handler) CancelAction(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	if err := h.svc.CancelAction(c.Request.Context(), userID); err != nil {
		response.ServerError(c, "取消失败")
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// ConfirmAction 执行确认后的写操作
func (h *Handler) ConfirmAction(c *gin.Context) {
	var req ConfirmActionReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	result, err := h.svc.ConfirmAction(c.Request.Context(), userID, req.Tool, req.Args)
	if err != nil {
		agentErr(c, err)
		return
	}
	response.OK(c, ConfirmActionResp{Result: result})
}

// 查 VIP 状态
func (h *Handler) GetQuota(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	quota, err := h.svc.GetQuota(c.Request.Context(), userID)
	if err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, quota)
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	ai := r.Group("/ai", authMiddleware)
	{
		ai.POST("/chat/stream", h.StreamChat)
		ai.GET("/history", h.GetHistory)
		ai.DELETE("/history", h.ClearHistory)
		ai.POST("/action/confirm", h.ConfirmAction)
		ai.POST("/action/cancel", h.CancelAction)
		ai.POST("/summarize", h.Summarize)
		ai.POST("/draft-message/stream", h.StreamDraftMessage)
		ai.POST("/draft-post/stream", h.StreamDraftPost)
		ai.GET("/quota", h.GetQuota)
	}
}
