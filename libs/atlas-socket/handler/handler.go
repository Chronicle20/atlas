package handler

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	"github.com/sirupsen/logrus"
)

type MessageValidator[S any] func(l logrus.FieldLogger, ctx context.Context) func(s S) bool

type MessageHandler[S any] func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s S, r *request.Reader, readerOptions map[string]interface{})

type Adapter[S any] func(name string, v MessageValidator[S], h MessageHandler[S], readerOptions map[string]interface{}) request.Handler
