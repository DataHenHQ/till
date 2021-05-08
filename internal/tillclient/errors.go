package tillclient

import (
	"errors"
	"fmt"
)

type CustomError struct {
	StatusCode int

	Err error
}

func (r *CustomError) Error() string {
	return fmt.Sprintf("status %v", r.Err)
}

var ErrNotFound = errors.New("not found")
