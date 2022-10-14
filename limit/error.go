package limit

import (
	"errors"
)

// universal error
var (
	// ErrUnknownCode is an error that represents unknown status code.
	ErrUnknownCode = errors.New("limit: unknown status code")
	// ErrDuplicateDriver is an error that driver duplicate.
	ErrDuplicateDriver = errors.New("limit: duplicate driver")
	// ErrUnsupportedDriver is an error that driver unsupported.
	ErrUnsupportedDriver = errors.New("limit: unsupported driver")
)
