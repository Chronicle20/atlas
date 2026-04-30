package tracing

import "os"

func osUnsetenv(k string) error {
	return os.Unsetenv(k)
}
