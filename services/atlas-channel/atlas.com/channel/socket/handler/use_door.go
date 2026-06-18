package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func UseDoorHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := fieldsb.UseDoor{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
	}
}
