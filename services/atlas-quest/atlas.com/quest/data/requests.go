package data

import (
	"context"
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

func MakePostRequest[A any](url string, body any) requests.Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := requests.AddHeaderDecorator(requests.SpanHeaderDecorator(ctx))
		td := requests.AddHeaderDecorator(requests.TenantHeaderDecorator(ctx))
		return requests.MakePostRequest[A](url, body, sd, td)(l, ctx)
	}
}
