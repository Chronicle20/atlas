package requests

import (
	"github.com/Chronicle20/atlas-rest/retry"
	"github.com/sirupsen/logrus"
	"net/http"
)

type DeleteRequest func(l logrus.FieldLogger) error

func delete(l logrus.FieldLogger) func(url string, configurators ...Configurator) error {
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

			c.headerDecorator(req.Header)

			r, err = http.DefaultClient.Do(req)
			if err != nil {
				l.Warnf("Failed calling %s on %s, will retry.", http.MethodDelete, url)
				return true, err
			}
			return false, nil
		}
		err := retry.Try(get, c.retries)
		if err != nil {
			l.WithError(err).Errorf("Unable to successfully call %s on %s.", http.MethodDelete, url)
			return err
		}
		l.WithFields(logrus.Fields{"method": http.MethodDelete, "status": r.Status, "path": url}).Debugf("Printing request.")

		return err
	}
}

func MakeDeleteRequest(url string, configurators ...Configurator) DeleteRequest {
	return func(l logrus.FieldLogger) error {
		return delete(l)(url, configurators...)
	}
}
