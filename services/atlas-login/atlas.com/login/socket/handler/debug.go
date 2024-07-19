package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const DebugHandle = "DebugHandle"

func DebugHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, r *request.Reader) {
		l.Warnf("[%s] in use. Read [%s].", DebugHandle, r.GetRestAsBytes())
	}
}
