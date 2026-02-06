package handler

import (
	"atlas-channel/chalkboard"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ChalkboardCloseHandle = "ChalkboardCloseHandle"

func ChalkboardCloseHandleHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		_ = chalkboard.NewProcessor(l, ctx).Close(s.Field(), s.CharacterId())
	}
}
