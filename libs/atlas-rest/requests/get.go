package requests

import (
	"github.com/Chronicle20/atlas-rest/retry"
	"github.com/sirupsen/logrus"
	"net/http"
)

type Request[A any] func(l logrus.FieldLogger) (A, error)

func get[A any](l logrus.FieldLogger) func(url string, resp *A, configurators ...Configurator) error {
	return func(url string, resp *A, configurators ...Configurator) error {
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

			r, err = http.DefaultClient.Do(req)
			if err != nil {
				l.Warnf("Failed calling %s on %s, will retry.", http.MethodGet, url)
				return true, err
			}
			return false, nil
		}
		err := retry.Try(get, c.retries)
		if err != nil {
			l.WithError(err).Errorf("Unable to successfully call GET on %s.", url)
			return err
		}
		err = processResponse(r, resp)

		l.WithFields(logrus.Fields{"method": http.MethodGet, "status": r.Status, "path": url, "response": resp}).Debugf("Printing request.")

		return err
	}
}

func MakeGetRequest[A any](url string, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger) (A, error) {
		var r A
		err := get[A](l)(url, &r, configurators...)
		return r, err
	}
}
