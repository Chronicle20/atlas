package requests

import (
	"context"

	"github.com/sirupsen/logrus"
)

func GetRequest[A any](url string) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := AddHeaderDecorator(SpanHeaderDecorator(ctx))
		td := AddHeaderDecorator(TenantHeaderDecorator(ctx))
		return MakeGetRequest[A](url, sd, td)(l, ctx)
	}
}

func PostRequest[A any](url string, i interface{}) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := AddHeaderDecorator(SpanHeaderDecorator(ctx))
		td := AddHeaderDecorator(TenantHeaderDecorator(ctx))
		return MakePostRequest[A](url, i, sd, td)(l, ctx)
	}
}

func PutRequest[A any](url string, i interface{}) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := AddHeaderDecorator(SpanHeaderDecorator(ctx))
		td := AddHeaderDecorator(TenantHeaderDecorator(ctx))
		return MakePutRequest[A](url, i, sd, td)(l, ctx)
	}
}

func PatchRequest[A any](url string, i interface{}) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		sd := AddHeaderDecorator(SpanHeaderDecorator(ctx))
		td := AddHeaderDecorator(TenantHeaderDecorator(ctx))
		return MakePatchRequest[A](url, i, sd, td)(l, ctx)
	}
}

func DeleteRequest(url string) EmptyBodyRequest {
	return func(l logrus.FieldLogger, ctx context.Context) error {
		sd := AddHeaderDecorator(SpanHeaderDecorator(ctx))
		td := AddHeaderDecorator(TenantHeaderDecorator(ctx))
		return MakeDeleteRequest(url, sd, td)(l, ctx)
	}
}
