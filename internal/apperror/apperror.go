package apperror

import "net/http"

// AppError represents a structured, machine-readable application error.
// Frontend can switch on Code (e.g., "AUTH_INVALID_CREDENTIALS") instead of
// string-matching Message.
type AppError struct {
	Code    string `json:"code"`    // Machine-readable: "AUTH_INVALID_CREDENTIALS"
	Message string `json:"message"` // Human-readable: "invalid email or password"
	Status  int    `json:"-"`       // HTTP status code (not serialized)
}

func (e *AppError) Error() string {
	return e.Message
}

// New creates an AppError with the given code, message, and HTTP status.
func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

// --- Error Catalog ---
// Define all application errors here so they are consistent across the codebase.

var (
	ErrInvalidCredentials         = New("AUTH_INVALID_CREDENTIALS", "invalid email or password", http.StatusUnauthorized)
	ErrAccountInactive            = New("AUTH_ACCOUNT_INACTIVE", "account is inactive", http.StatusForbidden)
	ErrTokenExpired               = New("AUTH_TOKEN_EXPIRED", "token has expired", http.StatusUnauthorized)
	ErrInvalidToken               = New("AUTH_INVALID_TOKEN", "invalid or malformed token", http.StatusUnauthorized)
	ErrRefreshTokenInvalid        = New("AUTH_REFRESH_INVALID", "invalid or expired refresh token", http.StatusUnauthorized)
	ErrRateLimited                = New("RATE_LIMITED", "too many requests", http.StatusTooManyRequests)
	ErrNotFound                   = New("NOT_FOUND", "resource not found", http.StatusNotFound)
	ErrForbidden                  = New("FORBIDDEN", "insufficient permissions", http.StatusForbidden)
	ErrConflict                   = New("CONFLICT", "resource already exists", http.StatusConflict)
	ErrValidationFailed           = New("VALIDATION_FAILED", "validation failed", http.StatusBadRequest)
	ErrInternal                   = New("INTERNAL_ERROR", "an unexpected error occurred", http.StatusInternalServerError)
	ErrPublicRegistrationDisabled = New("AUTH_REGISTRATION_DISABLED", "public registration is disabled", http.StatusForbidden)
	ErrTenantRequired             = New("TENANT_REQUIRED", "tenant context is required", http.StatusForbidden)
	ErrTenantAccessDenied         = New("TENANT_ACCESS_DENIED", "tenant access denied", http.StatusForbidden)

	// --- ADD YOUR ERROR CODES HERE ---
)
