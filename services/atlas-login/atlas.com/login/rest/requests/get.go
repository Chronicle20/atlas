package requests

import (
	"atlas-login/retry"
	"atlas-login/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

const (
	ID           = "TENANT_ID"
	Region       = "REGION"
	MajorVersion = "MAJOR_VERSION"
	MinorVersion = "MINOR_VERSION"
)

type Request[A any] func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) (A, error)

func get[A any](l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(url string, resp *A, configurators ...Configurator) error {
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
			r, err = http.DefaultClient.Do(req)
			if err != nil {
				l.Warnf("Failed calling GET on %s, will retry.", url)
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
	return func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) (A, error) {
		c := &configuration{}
		for _, configurator := range configurators {
			configurator(c)
		}

		var r A
		err := get[A](l, span, tenant)(url, &r, configurators...)
		return r, err
	}
}
