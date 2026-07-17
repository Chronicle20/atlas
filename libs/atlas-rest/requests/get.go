package requests

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	retry "github.com/Chronicle20/atlas/libs/atlas-retry"
)

var (
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("not found")
)

// ErrServiceUnavailable is returned when a GET exhausted its attempts and the
// final response was 503 Service Unavailable — the dependency is saturated
// but not broken. Callers distinguish it from ErrBadRequest/ErrNotFound via
// errors.Is.
var ErrServiceUnavailable = errors.New("service unavailable")

// errServiceUnavailableAttempt marks a single 503 attempt inside the retry
// loop; it is translated to ErrServiceUnavailable after exhaustion.
var errServiceUnavailableAttempt = errors.New("received 503 response")

type Request[A any] func(l logrus.FieldLogger, ctx context.Context) (A, error)

// parseRetryAfter parses an integer-seconds Retry-After header value.
func parseRetryAfter(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if s, err := strconv.Atoi(v); err == nil && s >= 0 {
		return time.Duration(s) * time.Second, true
	}
	return 0, false
}

// getBody issues the GET, drives the retry loop (transport errors and 503
// responses are retryable), and maps the final status to a sentinel error,
// returning the raw response body on success. get[A] wraps it with the
// JSON:API unmarshal; paged.go reuses it for envelope-aware decoding.
func getBody(l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) ([]byte, error) {
	return func(url string, configurators ...Configurator) ([]byte, error) {
		// GETs are idempotent reads of JSON:API resources: default to 3
		// attempts (transport errors and 503 responses are retryable).
		c := &configuration{retries: 3, timeout: DefaultTimeout}
		for _, configurator := range configurators {
			configurator(c)
		}

		var statusCode int
		var body []byte
		get := func(attempt int) (bool, error) {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				l.WithError(err).Errorf("Error creating request.")
				return true, err
			}

			for _, hd := range c.headerDecorators {
				hd(req.Header)
			}

			reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()
			req = req.WithContext(reqCtx)

			l.Debugf("Issuing [%s] request to [%s].", req.Method, req.URL)
			r, err := client.Do(req)
			if err != nil {
				l.WithError(err).Warnf("Failed calling [%s] on [%s], will retry.", http.MethodGet, url)
				return true, err
			}
			defer r.Body.Close()

			statusCode = r.StatusCode
			body, err = io.ReadAll(r.Body)
			if err != nil {
				l.WithError(err).Warnf("Failed reading response from [%s] on [%s], will retry.", http.MethodGet, url)
				return true, err
			}
			if statusCode == http.StatusServiceUnavailable {
				if attempt < c.retries {
					clientRetriesTotal.WithLabelValues("503").Inc()
					l.Warnf("Received [503] from [%s] on [%s], will retry.", http.MethodGet, url)
				}
				if d, ok := parseRetryAfter(r.Header.Get("Retry-After")); ok {
					return true, retry.WithDelayHint(errServiceUnavailableAttempt, d)
				}
				return true, errServiceUnavailableAttempt
			}
			return false, nil
		}
		cfg := retry.DefaultConfig().WithMaxRetries(c.retries).WithInitialDelay(200 * time.Millisecond).WithMaxDelay(2 * time.Second)
		err := retry.Try(ctx, cfg, get)

		if err != nil {
			if errors.Is(err, errServiceUnavailableAttempt) {
				l.WithError(err).Errorf("Service unavailable after retries calling [%s] on [%s].", http.MethodGet, url)
				return nil, ErrServiceUnavailable
			}
			l.WithError(err).Errorf("Unable to successfully call [%s] on [%s].", http.MethodGet, url)
			return nil, err
		}
		if statusCode == http.StatusOK || statusCode == http.StatusAccepted {
			return body, nil
		}
		if statusCode == http.StatusBadRequest {
			return nil, ErrBadRequest
		}
		if statusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		l.Debugf("Unable to successfully call [%s] on [%s], returned status code [%d].", http.MethodGet, url, statusCode)
		return nil, errors.New("unknown error")
	}
}

func get[A any](l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) (A, error) {
	return func(url string, configurators ...Configurator) (A, error) {
		var resp A
		body, err := getBody(l, ctx)(url, configurators...)
		if err != nil {
			return resp, err
		}
		resp, err = unmarshalResponse[A](body)
		l.WithFields(logrus.Fields{"method": http.MethodGet, "path": url, "response": resp}).Debugf("Printing request.")
		return resp, err
	}
}

//goland:noinspection GoUnusedExportedFunction
func MakeGetRequest[A any](url string, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		return get[A](l, ctx)(url, configurators...)
	}
}
