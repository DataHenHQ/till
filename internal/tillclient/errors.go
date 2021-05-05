package tillclient

import "fmt"

type CustomError struct {
	StatusCode int

	Err error
}

func (r *CustomError) Error() string {
	return fmt.Sprintf("status %v", r.Err)
}
