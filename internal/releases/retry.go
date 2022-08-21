package releases

import (
	"math/rand"
	"time"
)

const numRetries = 10

func withRetries(f func() (err error, shouldTryAgain bool)) error {
	var (
		lastErr      error
		nextInterval time.Duration = 77 * time.Millisecond
	)

	for i := 0; i < numRetries; i++ {
		err, shouldTryAgain := f()
		if err == nil || !shouldTryAgain {
			return err
		}

		lastErr = err

		time.Sleep(nextInterval)
		nextInterval += time.Duration(rand.Int63n(int64(nextInterval)))
	}

	return lastErr
}
