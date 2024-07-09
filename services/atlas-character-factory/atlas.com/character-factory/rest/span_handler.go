package rest

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"net/http"
)

type SpanHandler func(opentracing.Span) http.HandlerFunc

func RetrieveSpan(name string, next SpanHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wireCtx, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		serverSpan := opentracing.StartSpan(name, ext.RPCServerOption(wireCtx))
		defer serverSpan.Finish()
		next(serverSpan)(w, r)
	}
}
