package repository

import "errors"

// ErrNotFound marks a single-row lookup that matched nothing. Only *-by-id
// lookups produce it; list queries return empty slices instead. Handlers map
// it to 404 NOT_FOUND (docs/conventions.md §Error envelope rules).
var ErrNotFound = errors.New("not found")
