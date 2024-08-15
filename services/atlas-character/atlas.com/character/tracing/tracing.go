package tracing

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"os"
)

func InitTracer(l logrus.FieldLogger) func(serviceName string) (io.Closer, error) {
	return func(serviceName string) (io.Closer, error) {
		jaegerHostPort := os.Getenv("JAEGER_HOST_PORT")
		cfg := &config.Configuration{
			ServiceName: serviceName,
			Sampler:     &config.SamplerConfig{Type: "const", Param: 1},
			Reporter:    &config.ReporterConfig{LogSpans: true, LocalAgentHostPort: jaegerHostPort},
		}
		tracer, closer, err := cfg.NewTracer(config.Logger(LogrusAdapter{logger: l}))
		if err != nil {
			return nil, err
		}
		opentracing.SetGlobalTracer(tracer)
		return closer, nil
	}
}

func Teardown(l logrus.FieldLogger) func(tc io.Closer) func() {
	return func(tc io.Closer) func() {
		return func() {
			err := tc.Close()
			if err != nil {
				l.WithError(err).Errorf("Unable to close tracer.")
			}
		}
	}
}

type LogrusAdapter struct {
	logger logrus.FieldLogger
}

func (l LogrusAdapter) Error(msg string) {
	l.logger.Error(msg)
}

func (l LogrusAdapter) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args)
}

func StartSpan(l logrus.FieldLogger, name string, opts ...opentracing.StartSpanOption) (logrus.FieldLogger, opentracing.Span) {
	span := opentracing.StartSpan(name, opts...)
	sl := l.WithField("span.id", fmt.Sprintf("%v", span))
	return sl, span
}
