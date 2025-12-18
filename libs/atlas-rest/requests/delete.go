package requests

import (
	"context"
	"github.com/Chronicle20/atlas-rest/retry"
	"github.com/sirupsen/logrus"
	"net/http"
)

type EmptyBodyRequest func(l logrus.FieldLogger, ctx context.Context) error

func delete(l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) error {
	return func(url string, configurators ...Configurator) error {
		c := &configuration{retries: 1}
		for _, configurator := range configurators {
			configurator(c)
		}

		var r *http.Response
		get := func(attempt int) (bool, error) {
			var err error

			req, err := http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				l.WithError(err).Errorf("Error creating request.")
				return true, err
			}

			for _, hd := range c.headerDecorators {
				hd(req.Header)
			}

			req = req.WithContext(ctx)

			l.Debugf("Issuing [%s] request to [%s].", req.Method, req.URL)
			r, err = http.DefaultClient.Do(req)
			if err != nil {
				l.WithError(err).Warnf("Failed calling [%s] on [%s], will retry.", http.MethodDelete, url)
				return true, err
			}
			return false, nil
		}
		err := retry.Try(get, c.retries)
		if err != nil {
			l.WithError(err).Errorf("Unable to successfully call [%s] on [%s].", http.MethodDelete, url)
			return err
		}
		l.WithFields(logrus.Fields{"method": http.MethodDelete, "status": r.Status, "path": url}).Debugf("Printing request.")

		return err
	}
}

//goland:noinspection GoUnusedExportedFunction
func MakeDeleteRequest(url string, configurators ...Configurator) EmptyBodyRequest {
	return func(l logrus.FieldLogger, ctx context.Context) error {
		return delete(l, ctx)(url, configurators...)
	}
}
