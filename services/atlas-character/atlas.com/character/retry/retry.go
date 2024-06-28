package retry

import (
	"errors"
	"time"
)

type TryFunc func(attempt int) (retry bool, err error)

func Try(fn TryFunc, retries int) error {
	attempt := 1
	for {
		cont, err := fn(attempt)
		if !cont || err == nil {
			return nil
		}
		attempt++
		if attempt > retries {
			return errors.New("max retry reached")
		}
		time.Sleep(1 * time.Second)
	}
}