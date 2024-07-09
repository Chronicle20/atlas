package handler

import (
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Handler func(l logrus.FieldLogger, span opentracing.Span, msg kafka.Message) error
