package errors

// Error 是框架标准错误结构体，实现了 error 接口。
//
// 错误码结构：[严重程度(1位)][服务ID(3位)][序号(4位)]，共 8 位。
// 详见 tiles/errors/README.md。
type Error struct {
	Code    uint
	Message string // 面向用户的消息
	errMsg  string // 面向开发者的内部诊断信息，不对外暴露
	cause   error  // 原始错误，支持 errors.Is / errors.As 链式查找
}

// New 创建一个新的 Error。
func New(code uint, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Error 实现 error 接口，只返回面向用户的消息。
func (e *Error) Error() string { return e.Message }

// Unwrap 实现错误链，支持 errors.Is / errors.As 向下查找原始错误。
func (e *Error) Unwrap() error { return e.cause }

// Internal 返回内部诊断信息，仅在日志记录时使用，不得写入 HTTP 响应。
func (e *Error) Internal() string { return e.errMsg }

// Is 支持 errors.Is 按错误码匹配，无论是否经过 WithErrMsg / Wrap 包装。
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithErrMsg 附加内部诊断信息，返回新实例，不修改原哨兵错误。
func (e *Error) WithErrMsg(msg string) *Error {
	clone := *e
	clone.errMsg = msg
	return &clone
}

// Wrap 包装底层错误（如数据库错误），返回新实例，不修改原哨兵错误。
func (e *Error) Wrap(cause error) *Error {
	clone := *e
	clone.cause = cause
	return &clone
}

// tiles 框架通用哨兵错误，服务ID = 000，错误码段：20000001 - 20009999
var (
	ErrNotFound     = New(20000001, "资源不存在")
	ErrUnauthorized = New(20000002, "未授权")
	ErrForbidden    = New(20000003, "权限不足")
	ErrConflict     = New(20000004, "资源已存在")
	ErrBadRequest   = New(20000005, "请求参数有误")
	ErrInternal     = New(20000006, "服务内部错误")
)
