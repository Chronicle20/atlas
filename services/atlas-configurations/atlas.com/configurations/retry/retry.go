package retry

import (
	"errors"
	"time"
)

type RepeatableFunc func(attempt int) (retry bool, err error)

type RepeatableFuncWithResponse func(attempt int) (bool, interface{}, error)

func Try(fn RepeatableFunc, retries int) error {
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

