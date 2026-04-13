package errorsx

import (
	stderrs "errors"

	"shop/api/gen/go/common"

	kratosErrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-sql-driver/mysql"
)

const (
	// MetadataKeyConflictType 标识冲突类型。
	MetadataKeyConflictType = "conflict_type"
	// MetadataKeyResource 标识资源名称。
	MetadataKeyResource = "resource"
	// MetadataKeyField 标识字段名称。
	MetadataKeyField = "field"
	// MetadataKeyConstraint 标识数据库约束名称。
	MetadataKeyConstraint = "constraint"
	// MetadataKeyChildResource 标识子资源名称。
	MetadataKeyChildResource = "child_resource"
	// MetadataKeyCurrentState 标识当前状态。
	MetadataKeyCurrentState = "current_state"
	// MetadataKeyExpectedState 标识期望状态。
	MetadataKeyExpectedState = "expected_state"

	// ConflictTypeUniqueViolation 表示唯一约束冲突。
	ConflictTypeUniqueViolation = "unique_violation"
	// ConflictTypeHasChildren 表示仍存在子资源。
	ConflictTypeHasChildren = "has_children"
	// ConflictTypeStateConflict 表示状态冲突。
	ConflictTypeStateConflict = "state_conflict"
	// ConflictTypeProtectedResource 表示受保护资源。
	ConflictTypeProtectedResource = "protected_resource"
)

// InvalidArgument 构造请求参数错误。
func InvalidArgument(message string) *kratosErrors.Error {
	return kratosErrors.New(400, common.ErrorReason_INVALID_ARGUMENT.String(), message)
}

// Unauthenticated 构造认证失败错误。
func Unauthenticated(message string) *kratosErrors.Error {
	return kratosErrors.New(401, common.ErrorReason_UNAUTHENTICATED.String(), message)
}

// PermissionDenied 构造权限不足错误。
func PermissionDenied(message string) *kratosErrors.Error {
	return kratosErrors.New(403, common.ErrorReason_PERMISSION_DENIED.String(), message)
}

// ResourceNotFound 构造资源不存在错误。
func ResourceNotFound(message string) *kratosErrors.Error {
	return kratosErrors.New(404, common.ErrorReason_RESOURCE_NOT_FOUND.String(), message)
}

// Conflict 构造状态冲突错误。
func Conflict(message string) *kratosErrors.Error {
	return kratosErrors.New(409, common.ErrorReason_CONFLICT.String(), message)
}

// Internal 构造内部错误。
func Internal(message string) *kratosErrors.Error {
	return kratosErrors.New(500, common.ErrorReason_INTERNAL_ERROR.String(), message)
}

// UniqueConflict 构造唯一约束冲突错误。
func UniqueConflict(message, resource, field, constraint string) *kratosErrors.Error {
	metadata := map[string]string{
		MetadataKeyConflictType: ConflictTypeUniqueViolation,
		MetadataKeyResource:     resource,
		MetadataKeyField:        field,
	}
	// 提供了约束名时，再补充到错误元数据中。
	if constraint != "" {
		metadata[MetadataKeyConstraint] = constraint
	}
	return Conflict(message).WithMetadata(metadata)
}

// HasChildrenConflict 构造存在子资源的冲突错误。
func HasChildrenConflict(message, resource, childResource string) *kratosErrors.Error {
	metadata := map[string]string{
		MetadataKeyConflictType: ConflictTypeHasChildren,
		MetadataKeyResource:     resource,
	}
	// 已知子资源名称时，再补充到错误元数据中。
	if childResource != "" {
		metadata[MetadataKeyChildResource] = childResource
	}
	return Conflict(message).WithMetadata(metadata)
}

// StateConflict 构造状态冲突错误。
func StateConflict(message, resource, currentState, expectedState string) *kratosErrors.Error {
	metadata := map[string]string{
		MetadataKeyConflictType: ConflictTypeStateConflict,
		MetadataKeyResource:     resource,
	}
	// 提供了当前状态时，再补充到错误元数据中。
	if currentState != "" {
		metadata[MetadataKeyCurrentState] = currentState
	}
	// 提供了期望状态时，再补充到错误元数据中。
	if expectedState != "" {
		metadata[MetadataKeyExpectedState] = expectedState
	}
	return Conflict(message).WithMetadata(metadata)
}

// ProtectedResourceConflict 构造受保护资源冲突错误。
func ProtectedResourceConflict(message, resource string) *kratosErrors.Error {
	metadata := map[string]string{
		MetadataKeyConflictType: ConflictTypeProtectedResource,
	}
	// 提供了资源名称时，再补充到错误元数据中。
	if resource != "" {
		metadata[MetadataKeyResource] = resource
	}
	return Conflict(message).WithMetadata(metadata)
}

// IsStructuredError 判断错误是否已经携带稳定 reason。
func IsStructuredError(err error) bool {
	var kratosErr *kratosErrors.Error
	// 已经是 Kratos 错误且 reason 非空时，视为已完成分类。
	return stderrs.As(err, &kratosErr) && kratosErr.Reason != ""
}

// WrapIfNeeded 在错误尚未完成分类时，使用兜底错误包装。
func WrapIfNeeded(err error, fallback *kratosErrors.Error) error {
	if err == nil {
		return nil
	}
	// 已经完成分类的错误直接透传，避免覆盖原始语义。
	if IsStructuredError(err) {
		return err
	}
	if fallback == nil {
		return err
	}
	return fallback.WithCause(err)
}

// WrapInternal 在错误尚未完成分类时，包装成内部错误。
func WrapInternal(err error, message string) error {
	return WrapIfNeeded(err, Internal(message))
}

// IsMySQLDuplicateKey 判断是否命中了 MySQL 唯一键冲突。
func IsMySQLDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return stderrs.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
