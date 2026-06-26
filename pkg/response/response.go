package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func SuccessWithTotal(c *gin.Context, data interface{}, total int64) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data, Total: total})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Code: 0, Message: "created", Data: data})
}

func Error(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Response{Code: httpStatus, Message: message})
}

func BadRequest(c *gin.Context, message string) { Error(c, http.StatusBadRequest, message) }

func Unauthorized(c *gin.Context, message string) { Error(c, http.StatusUnauthorized, message) }

func Forbidden(c *gin.Context, message string) { Error(c, http.StatusForbidden, message) }

func NotFound(c *gin.Context, message string) { Error(c, http.StatusNotFound, message) }

func InternalError(c *gin.Context, message string) { Error(c, http.StatusInternalServerError, message) }

func OK(c *gin.Context) { Success(c, nil) }

type PageParams struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Keyword  string `form:"keyword"`
}

func DefaultPage() PageParams { return PageParams{Page: 1, PageSize: 20} }

func (p PageParams) Offset() int { return (p.Page - 1) * p.PageSize }

func (p PageParams) Limit() int { return p.PageSize }
