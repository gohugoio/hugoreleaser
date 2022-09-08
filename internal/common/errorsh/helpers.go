package errorsh

import (
	"context"
	"errors"

	"github.com/bep/execrpc"
)

// IsShutdownError returns true if the error is a shutdown error which we normally don't report to the user.
func IsShutdownError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, execrpc.ErrShutdown)
}
