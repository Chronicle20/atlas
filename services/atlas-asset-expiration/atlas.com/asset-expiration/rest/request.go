package rest

import (
	"context"
	"net/http"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func MakeGetRequest[A any](url string) requests.Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := requests.AddHeaderDecorator(requests.SpanHeaderDecorator(ctx))
		td := requests.AddHeaderDecorator(requests.TenantHeaderDecorator(ctx))
		return requests.MakeGetRequest[A](url, sd, td)(l, ctx)
	}
}

func MakeGetRequestWithHeader[A any](url string, headerKey, headerValue string) requests.Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := requests.AddHeaderDecorator(requests.SpanHeaderDecorator(ctx))
		td := requests.AddHeaderDecorator(requests.TenantHeaderDecorator(ctx))
		hd := requests.AddHeaderDecorator(func(h http.Header) {
			h.Set(headerKey, headerValue)
		})
		return requests.MakeGetRequest[A](url, sd, td, hd)(l, ctx)
	}
}
