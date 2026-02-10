package retry

import (
	"fmt"
	"time"
)

func Try(f func(attempt int) (bool, error), maxRetries int) error {
	var err error
	var cont bool
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cont, err = f(attempt)
		if !cont || err == nil {
			return err
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return fmt.Errorf("after %d attempts, last error: %s", maxRetries, err)
}
