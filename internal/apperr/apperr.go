package apperr

// AppError represents an application error that can be mapped to the
// normalized JSON error envelope.
type AppError struct {
	Code         string
	Message      string
	Retryable    bool
	HTTPStatus   int
	UpstreamCode string
}

func (e *AppError) Error() string {
	return e.Message
}

// New creates a basic application error.
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WithRetryable marks whether the error is retryable.
func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// WithHTTPStatus sets the related HTTP status code.
func (e *AppError) WithHTTPStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

// WithUpstreamCode sets the upstream provider error code.
func (e *AppError) WithUpstreamCode(code string) *AppError {
	e.UpstreamCode = code
	return e
}
