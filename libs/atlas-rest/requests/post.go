package requests

import (
	"bytes"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

type PostRequest[A any] func(l logrus.FieldLogger) (A, error)

func post[A any](l logrus.FieldLogger) func(url string, input interface{}, resp *A, configurators ...Configurator) error {
	return func(url string, input interface{}, resp *A, configurators ...Configurator) error {
		c := &configuration{retries: 1}
		for _, configurator := range configurators {
			configurator(c)
		}

		jsonReq, err := jsonapi.Marshal(input)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonReq))
		if err != nil {
			return err
		}

		c.headerDecorator(req.Header)

		r, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if r.ContentLength > 0 {
			err = processResponse(r, resp)
			if err != nil {
				return err
			}
			l.WithFields(logrus.Fields{"method": http.MethodPost, "status": r.Status, "path": url, "input": input, "response": resp}).Debugf("Printing request.")
		} else {
			l.WithFields(logrus.Fields{"method": http.MethodPost, "status": r.Status, "path": url, "input": input, "response": ""}).Debugf("Printing request.")
		}

		return nil
	}
}

func MakePostRequest[A any](url string, i interface{}, configurators ...Configurator) PostRequest[A] {
	return func(l logrus.FieldLogger) (A, error) {
		var r A
		err := post[A](l)(url, i, &r, configurators...)
		return r, err
	}
}
