package rest

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"net/http"
)

type SpanHandler func(logrus.FieldLogger, opentracing.Span) http.HandlerFunc

func RetrieveSpan(l logrus.FieldLogger, name string, next SpanHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wireCtx, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		serverSpan := opentracing.StartSpan(name, ext.RPCServerOption(wireCtx))
		sl := l.WithField("span.id", fmt.Sprintf("%v", serverSpan))
		defer serverSpan.Finish()
		next(sl, serverSpan)(w, r)
	}
}
