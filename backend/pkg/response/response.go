package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK           = 0
	CodeUnauthorized = 4001
	CodeForbidden    = 4003
	CodeNotFound     = 4004
	CodeBadRequest   = 4000
	CodeServerError  = 5000
)

// 统一响应格式
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 通用成功响应
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 通用失败响应
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Response{Code: code, Message: msg, Data: nil})
}

// BadRequest 请求参数错误
func BadRequest(c *gin.Context, msg string) {
	Fail(c, http.StatusBadRequest, CodeBadRequest, msg)
}

// BindJSON 绑定 JSON 请求体 失败时写出 400
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		BadRequest(c, err.Error())
		return false
	}
	return true
}

// Unauthorized 未认证
func Unauthorized(c *gin.Context, msg string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}

// Forbidden 权限不足
func Forbidden(c *gin.Context, msg string) {
	Fail(c, http.StatusForbidden, CodeForbidden, msg)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, msg string) {
	Fail(c, http.StatusNotFound, CodeNotFound, msg)
}

// ServerError 服务器错误
func ServerError(c *gin.Context, msg string) {
	Fail(c, http.StatusInternalServerError, CodeServerError, msg)
}
