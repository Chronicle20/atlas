package requests

import (
	"bytes"
	"github.com/Chronicle20/atlas-rest/retry"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func createOrUpdate[A any](l logrus.FieldLogger) func(method string) func(url string, input interface{}, configurators ...Configurator) (A, error) {
	return func(method string) func(url string, input interface{}, configurators ...Configurator) (A, error) {
		return func(url string, input interface{}, configurators ...Configurator) (A, error) {
			c := &configuration{retries: 1}
			for _, configurator := range configurators {
				configurator(c)
			}

			var result A
			jsonReq, err := jsonapi.Marshal(input)
			if err != nil {
				return result, err
			}

			var r *http.Response
			post := func(attempt int) (bool, error) {
				var err error

				req, err := http.NewRequest(method, url, bytes.NewReader(jsonReq))
				if err != nil {
					l.WithError(err).Errorf("Error creating request.")
					return true, err
				}

				c.headerDecorator(req.Header)

				l.Debugf("Issuing [%s] request to [%s].", method, req.URL)
				r, err = http.DefaultClient.Do(req)
				if err != nil {
					l.Warnf("Failed calling [%s] on [%s], will retry.", method, url)
					return true, err
				}
				return false, nil
			}
			err = retry.Try(post, c.retries)
			if err != nil {
				l.WithError(err).Errorf("Unable to successfully call [%s] on [%s].", method, url)
				return result, err
			}

			if r.ContentLength == 0 {
				l.WithFields(logrus.Fields{"method": method, "status": r.Status, "path": url, "input": input, "response": ""}).Debugf("Printing request.")
			} else {
				result, err = processResponse[A](r)
				if err != nil {
					return result, err
				}
				l.WithFields(logrus.Fields{"method": method, "status": r.Status, "path": url, "input": input, "response": result}).Debugf("Printing request.")
			}

			return result, nil
		}
	}
}

//goland:noinspection GoUnusedExportedFunction
func MakePostRequest[A any](url string, i interface{}, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger) (A, error) {
		return createOrUpdate[A](l)(http.MethodPost)(url, i, configurators...)
	}
}
