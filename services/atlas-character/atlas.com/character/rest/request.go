package rest

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

func headerDecorator(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) requests.HeaderDecorator {
	return func(h http.Header) {
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set(ID, tenant.Id.String())
		h.Set(Region, tenant.Region)
		h.Set(MajorVersion, strconv.Itoa(int(tenant.MajorVersion)))
		h.Set(MinorVersion, strconv.Itoa(int(tenant.MinorVersion)))

		err := opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(h))
		if err != nil {
			l.WithError(err).Errorf("Unable to decorate request headers with OpenTracing information.")
		}
	}
}

func MakeGetRequest[A any](l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(url string) requests.Request[A] {
	hd := requests.SetHeaderDecorator(headerDecorator(l, span, tenant))
	return func(url string) requests.Request[A] {
		return requests.MakeGetRequest[A](url, hd)
	}
}

func MakePostRequest[A any](l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(url string, i interface{}) requests.Request[A] {
	hd := requests.SetHeaderDecorator(headerDecorator(l, span, tenant))
	return func(url string, i interface{}) requests.Request[A] {
		return requests.MakePostRequest[A](url, i, hd)
	}
}

func MakeDeleteRequest(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(url string) requests.EmptyBodyRequest {
	hd := requests.SetHeaderDecorator(headerDecorator(l, span, tenant))
	return func(url string) requests.EmptyBodyRequest {
		return requests.MakeDeleteRequest(url, hd)
	}
}
