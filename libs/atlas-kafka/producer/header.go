package producer

import "github.com/opentracing/opentracing-go"

type HeaderDecorator func() (map[string]string, error)

//goland:noinspection GoUnusedExportedFunction
func SpanHeaderDecorator(span opentracing.Span) HeaderDecorator {
	return func() (map[string]string, error) {
		headers := make(map[string]string)
		err := opentracing.GlobalTracer().Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(headers))
		return headers, err
	}
}
