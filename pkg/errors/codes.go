// Package errors 提供增强的错误处理
package errors

// ErrNotFound 预定义错误
var (
	// ErrInvalidInput 输入无效
	ErrInvalidInput = New(40000, "invalid input")
	// ErrUnauthorized 未授权
	ErrUnauthorized = New(40100, "unauthorized")
	// ErrForbidden 禁止访问
	ErrForbidden = New(40300, "forbidden")
	// ErrNotFound 未找到
	ErrNotFound = New(40400, "not found")
	// ErrConflict 冲突
	ErrConflict = New(40900, "conflict")
	// ErrInternal 内部错误
	ErrInternal = New(50000, "internal error")
	// ErrTimeout 超时
	ErrTimeout = New(50400, "timeout")
	// ErrRateLimited 速率限制
	ErrRateLimited = New(42900, "rate limited")
	// ErrValidation 校验错误
	ErrValidation = New(40001, "validation error")
	// ErrDatabase 数据库错误
	ErrDatabase = New(50001, "database error")
)

// ErrorCodeMap 错误码映射
var ErrorCodeMap = map[int]string{
	40000: "Bad Request",
	40001: "Validation Error",
	40100: "Unauthorized",
	40300: "Forbidden",
	40400: "Not Found",
	40900: "Conflict",
	42900: "Too Many Requests",
	50000: "Internal Server Error",
	50001: "Database Error",
	50400: "Gateway Timeout",
}

// GetMessage 获取错误码对应的消息
func GetMessage(code int) string {
	if msg, ok := ErrorCodeMap[code]; ok {
		return msg
	}
	return "Unknown Error"
}

// NewNotFoundError 创建未找到错误
func NewNotFoundError(resource, id string) *Error {
	return New(40400, resource+" not found: "+id)
}

// NewValidationError 创建校验错误
func NewValidationError(field, message string) *Error {
	return New(40001, "validation error: "+field+" - "+message)
}

// NewConflictError 创建冲突错误
func NewConflictError(resource, message string) *Error {
	return New(40900, "conflict: "+resource+" - "+message)
}

// IsNotFound 是否为未找到错误
func IsNotFound(err error) bool {
	return IsError(err, 40400)
}

// IsValidationError 是否为校验错误
func IsValidationError(err error) bool {
	return IsError(err, 40001)
}

// IsConflictError 是否为冲突错误
func IsConflictError(err error) bool {
	return IsError(err, 40900)
}

// IsInternalError 是否为内部错误
func IsInternalError(err error) bool {
	return IsError(err, 50000)
}

// IsRateLimited 是否为速率限制
func IsRateLimited(err error) bool {
	return IsError(err, 42900)
}
