package requests

import (
	"errors"
	"github.com/Chronicle20/atlas-rest/retry"
	"github.com/sirupsen/logrus"
	"net/http"
)

var ErrBadRequest = errors.New("bad request")
var ErrNotFound = errors.New("not found")

type Request[A any] func(l logrus.FieldLogger) (A, error)

func get[A any](l logrus.FieldLogger) func(url string, configurators ...Configurator) (A, error) {
	return func(url string, configurators ...Configurator) (A, error) {
		c := &configuration{retries: 1}
		for _, configurator := range configurators {
			configurator(c)
		}

		var r *http.Response
		get := func(attempt int) (bool, error) {
			var err error

			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				l.WithError(err).Errorf("Error creating request.")
				return true, err
			}

			c.headerDecorator(req.Header)

			l.Debugf("Issuing [%s] request to [%s].", req.Method, req.URL)
			r, err = http.DefaultClient.Do(req)
			if err != nil {
				l.Warnf("Failed calling [%s] on [%s], will retry.", http.MethodGet, url)
				return true, err
			}
			return false, nil
		}
		err := retry.Try(get, c.retries)

		var resp A
		if err != nil {
			l.WithError(err).Errorf("Unable to successfully call [%s] on [%s].", http.MethodGet, url)
			return resp, err
		}
		if r.StatusCode == http.StatusOK || r.StatusCode == http.StatusAccepted {
			resp, err = processResponse[A](r)
			l.WithFields(logrus.Fields{"method": http.MethodGet, "status": r.Status, "path": url, "response": resp}).Debugf("Printing request.")
			return resp, err
		}
		if r.StatusCode == http.StatusBadRequest {
			return resp, ErrBadRequest
		}
		if r.StatusCode == http.StatusNotFound {
			return resp, ErrNotFound
		}
		l.Debugf("Unable to successfully call [%s] on [%s], returned status code [%d].", http.MethodGet, url, r.StatusCode)
		return resp, errors.New("unknown error")
	}
}

//goland:noinspection GoUnusedExportedFunction
func MakeGetRequest[A any](url string, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger) (A, error) {
		return get[A](l)(url, configurators...)
	}
}
