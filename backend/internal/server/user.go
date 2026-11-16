package server

import "net/http"

// UserResolver returns the acting user id for a request (D-027). In
// single-user mode it always returns the configured default user; when auth
// lands (D-013) only this implementation changes.
type UserResolver func(r *http.Request) (string, error)

func DefaultUserResolver(defaultUserID string) UserResolver {
	return func(r *http.Request) (string, error) {
		return defaultUserID, nil
	}
}
