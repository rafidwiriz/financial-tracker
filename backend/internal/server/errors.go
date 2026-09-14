package server

// AppError is the internal representation of an API error. Handlers and
// services return it; the global handler serializes it via the error envelope
// (docs/api-design.md §Error codes).
type AppError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *AppError) Error() string { return e.Message }

// NewAppError is a small constructor for handlers.
func NewAppError(status int, code, message string, details any) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}