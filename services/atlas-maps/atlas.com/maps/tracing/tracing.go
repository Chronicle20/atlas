package tracing

import (
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

type LogrusAdapter struct {
	logger logrus.FieldLogger
}

func (l LogrusAdapter) Error(msg string) {
	l.logger.Error(msg)
}

func (l LogrusAdapter) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args)
}
