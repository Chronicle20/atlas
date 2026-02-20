package requests

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Chronicle20/atlas-retry"
	"github.com/sirupsen/logrus"
)

var ErrBadRequest = errors.New("bad request")
var ErrNotFound = errors.New("not found")

type Request[A any] func(l logrus.FieldLogger, ctx context.Context) (A, error)

func get[A any](l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) (A, error) {
	return func(url string, configurators ...Configurator) (A, error) {
		c := &configuration{retries: 1, timeout: DefaultTimeout}
		for _, configurator := range configurators {
			configurator(c)
		}

		var statusCode int
		var status string
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
			status = r.Status
			body, err = io.ReadAll(r.Body)
			if err != nil {
				l.WithError(err).Warnf("Failed reading response from [%s] on [%s], will retry.", http.MethodGet, url)
				return true, err
			}
			return false, nil
		}
		cfg := retry.DefaultConfig().WithMaxRetries(c.retries).WithInitialDelay(200 * time.Millisecond).WithMaxDelay(5 * time.Second)
		err := retry.Try(ctx, cfg, get)

		var resp A
		if err != nil {
			l.WithError(err).Errorf("Unable to successfully call [%s] on [%s].", http.MethodGet, url)
			return resp, err
		}
		if statusCode == http.StatusOK || statusCode == http.StatusAccepted {
			resp, err = unmarshalResponse[A](body)
			l.WithFields(logrus.Fields{"method": http.MethodGet, "status": status, "path": url, "response": resp}).Debugf("Printing request.")
			return resp, err
		}
		if statusCode == http.StatusBadRequest {
			return resp, ErrBadRequest
		}
		if statusCode == http.StatusNotFound {
			return resp, ErrNotFound
		}
		l.Debugf("Unable to successfully call [%s] on [%s], returned status code [%d].", http.MethodGet, url, statusCode)
		return resp, errors.New("unknown error")
	}
}

//goland:noinspection GoUnusedExportedFunction
func MakeGetRequest[A any](url string, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		return get[A](l, ctx)(url, configurators...)
	}
}
