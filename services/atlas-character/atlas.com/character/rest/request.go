package rest

import (
	"atlas-character/tenant"
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"net/http"
	"strconv"
)

func headerDecorator(ctx context.Context, tenant tenant.Model) requests.HeaderDecorator {
	return func(h http.Header) {
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set(ID, tenant.Id.String())
		h.Set(Region, tenant.Region)
		h.Set(MajorVersion, strconv.Itoa(int(tenant.MajorVersion)))
		h.Set(MinorVersion, strconv.Itoa(int(tenant.MinorVersion)))

		propagator := otel.GetTextMapPropagator()
		propagator.Inject(ctx, propagation.HeaderCarrier(h))
	}
}

func MakeGetRequest[A any](ctx context.Context, tenant tenant.Model) func(url string) requests.Request[A] {
	hd := requests.SetHeaderDecorator(headerDecorator(ctx, tenant))
	return func(url string) requests.Request[A] {
		return requests.MakeGetRequest[A](url, hd)
	}
}

func MakePostRequest[A any](ctx context.Context, tenant tenant.Model) func(url string, i interface{}) requests.Request[A] {
	hd := requests.SetHeaderDecorator(headerDecorator(ctx, tenant))
	return func(url string, i interface{}) requests.Request[A] {
		return requests.MakePostRequest[A](url, i, hd)
	}
}

func MakePatchRequest[A any](ctx context.Context, tenant tenant.Model) func(url string, i interface{}) requests.Request[A] {
	hd := requests.SetHeaderDecorator(headerDecorator(ctx, tenant))
	return func(url string, i interface{}) requests.Request[A] {
		return requests.MakePatchRequest[A](url, i, hd)
	}
}

func MakeDeleteRequest(ctx context.Context, tenant tenant.Model) func(url string) requests.EmptyBodyRequest {
	hd := requests.SetHeaderDecorator(headerDecorator(ctx, tenant))
	return func(url string) requests.EmptyBodyRequest {
		return requests.MakeDeleteRequest(url, hd)
	}
}
