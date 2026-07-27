package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func AdminCommandHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := fieldsb.AdminCommand{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
	}
}
