// Package httpx 提供统一响应封装 {code, message, data} 与错误处理（docs/admin/01 §5 约定）。
package httpx

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Envelope 是所有管理面接口的统一响应体。
type Envelope struct {
	Code    int    `json:"code"`              // 0 = 成功，非 0 = 业务错误码（沿用 HTTP 状态码语义）
	Message string `json:"message"`           // 人类可读信息
	Data    any    `json:"data,omitempty"`    // 业务数据
}

// OK 返回成功响应。
func OK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data})
}

// Created 返回 201。
func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, Envelope{Code: 0, Message: "created", Data: data})
}

// Fail 返回业务错误（HTTP 状态码 = code）。
func Fail(c echo.Context, code int, message string) error {
	return c.JSON(code, Envelope{Code: code, Message: message})
}

// BindError 表示请求体解析失败。
var BindError = errors.New("请求参数解析失败")

// HTTPErrorHandler 是 Echo 全局错误处理器，把 echo.HTTPError 与普通 error
// 都包成统一 Envelope，避免泄漏内部细节。
func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	code := http.StatusInternalServerError
	msg := "服务器内部错误"

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		if m, ok := he.Message.(string); ok {
			msg = m
		} else if he.Message != nil {
			msg = http.StatusText(he.Code)
		}
	}
	_ = c.JSON(code, Envelope{Code: code, Message: msg})
}
