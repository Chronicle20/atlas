package handler

import (
	"context"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Handler func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error)
