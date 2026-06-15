package user

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"mechat/pkg/middleware"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

// FriendIDGetter 防循环依赖的最小接口
type FriendIDGetter interface {
	GetFriendIDs(ctx context.Context, userID uint64) ([]uint64, error)
}

// Handler 用户处理器
type Handler struct {
	svc       *Service
	friendSvc FriendIDGetter
}

// 创建用户处理器
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// 注入 FriendIDGetter
func (h *Handler) SetFriendSvc(f FriendIDGetter) {
	h.friendSvc = f
}

// 发送验证码
func (h *Handler) SendCode(c *gin.Context) {
	var req SendCodeReq
	if !response.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SendCode(c.Request.Context(), &req); err != nil {
		if err == ErrCodeSendTooFast {
			response.BadRequest(c, err.Error())
		} else {
			response.ServerError(c, "发送失败")
		}
		return
	}
	response.OK(c, nil)
}

// 注册
func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if !response.BindJSON(c, &req) {
		return
	}
	info, token, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrCodeInvalid:
			response.BadRequest(c, err.Error())
		case ErrEmailExists:
			response.BadRequest(c, err.Error())
		default:
			response.ServerError(c, "注册失败")
		}
		return
	}
	response.OK(c, gin.H{"token": token, "user": info})
}

// 登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if !response.BindJSON(c, &req) {
		return
	}
	info, token, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrCodeInvalid:
			response.BadRequest(c, err.Error())
		case ErrUserNotFound:
			response.BadRequest(c, "用户不存在，请先注册")
		default:
			response.ServerError(c, "登录失败")
		}
		return
	}
	response.OK(c, gin.H{
		"token":      token,
		"expires_at": time.Now().Add(720 * time.Hour),
		"user":       info,
	})
}

// 登出
func (h *Handler) Logout(c *gin.Context) {
	tokenStr, _ := c.Get("tokenStr")
	expireAt, _ := c.Get("tokenExpireAt")
	h.svc.Logout(c.Request.Context(), tokenStr.(string), expireAt.(time.Time))
	response.OK(c, nil)
}

// 当前用户
func (h *Handler) Me(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	u, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}
	response.OK(c, u.ToInfo())
}

// 更新资料
func (h *Handler) UpdateMe(c *gin.Context) {
	var req UpdateProfileReq
	if !response.BindJSON(c, &req) {
		return
	}
	userID := middleware.CurrentUserID(c)
	info, err := h.svc.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		if err == ErrNicknameExists {
			response.BadRequest(c, err.Error())
		} else {
			response.ServerError(c, "更新失败："+err.Error())
		}
		return
	}
	response.OK(c, info)
}

// 上传头像
func (h *Handler) UploadAvatar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传文件")
		return
	}
	defer file.Close()

	if header.Size > 10<<20 { // 10MB
		response.BadRequest(c, "文件不能超过 10MB")
		return
	}

	userID := middleware.CurrentUserID(c)
	url, err := h.svc.UploadAvatar(c.Request.Context(), userID, file, header.Size, header.Filename)
	if err != nil {
		response.ServerError(c, "上传失败: "+err.Error())
		return
	}
	response.OK(c, gin.H{"url": url})
}

// 用户公开信息
func (h *Handler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}
	info, err := h.svc.GetPublicInfo(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}
	response.OK(c, info)
}

// 搜索用户
func (h *Handler) SearchUsers(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		response.BadRequest(c, "请输入搜索关键词")
		return
	}
	users, err := h.svc.Search(c.Request.Context(), keyword)
	if err != nil {
		response.ServerError(c, "搜索失败")
		return
	}
	response.OK(c, gin.H{"list": users, "total": len(users)})
}

// 通用图片上传
func (h *Handler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传文件")
		return
	}
	defer file.Close()

	if header.Size > 10<<20 {
		response.BadRequest(c, "文件不能超过 10MB")
		return
	}

	url, err := h.svc.UploadImage(file, header.Size, header.Filename)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url})
}

// 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	auth := r.Group("/auth")
	{
		auth.POST("/send-code", h.SendCode)
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/logout", authMiddleware, h.Logout)
	}

	users := r.Group("/users", authMiddleware)
	{
		users.GET("/me", h.Me)
		users.PUT("/me", h.UpdateMe)
		users.PUT("/me/avatar", h.UploadAvatar)
		users.GET("/search", h.SearchUsers)
		users.GET("/recommend", h.RecommendUsers)
		users.GET("/:id", h.GetUser)
	}

	r.POST("/upload/image", authMiddleware, h.UploadImage)
}

// 推荐用户 排除自己和好友
func (h *Handler) RecommendUsers(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	excludeIDs := []uint64{userID}
	if h.friendSvc != nil {
		if ids, err := h.friendSvc.GetFriendIDs(c.Request.Context(), userID); err == nil {
			excludeIDs = append(excludeIDs, ids...)
		}
	}
	list, err := h.svc.Recommend(c.Request.Context(), excludeIDs, 10)
	if err != nil {
		response.ServerError(c, "获取推荐失败")
		return
	}
	response.OK(c, gin.H{"list": list})
}

const _ = http.StatusOK
