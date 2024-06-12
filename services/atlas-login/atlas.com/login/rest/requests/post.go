package requests

import (
	"atlas-login/tenant"
	"bytes"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type PostRequest[A any] func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) (A, error)

func post[A any](l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(url string, input interface{}, resp *A) error {
	return func(url string, input interface{}, resp *A) error {
		jsonReq, err := jsonapi.Marshal(input)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonReq))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set(ID, tenant.Id().String())
		req.Header.Set(Region, tenant.Region())
		req.Header.Set(MajorVersion, strconv.Itoa(int(tenant.MajorVersion())))
		req.Header.Set(MinorVersion, strconv.Itoa(int(tenant.MinorVersion())))
		err = opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header))
		if err != nil {
			l.WithError(err).Errorf("Unable to decorate request headers with OpenTracing information.")
		}
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
	return func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) (A, error) {
		c := &configuration{}
		for _, configurator := range configurators {
			configurator(c)
		}

		var r A
		err := post[A](l, span, tenant)(url, i, &r)
		return r, err
	}
}
