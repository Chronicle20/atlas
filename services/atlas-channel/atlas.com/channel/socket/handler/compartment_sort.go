package handler

import (
	"atlas-channel/compartment"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CompartmentSortHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.CompartmentSortRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		compartmentType := inventory.Type(p.CompartmentType())

		isValid := false
		for _, validType := range inventory.Types {
			if compartmentType == validType {
				isValid = true
				break
			}
		}

		if !isValid {
			l.Warnf("Character [%d] issued compartment sort with invalid compartment type [%d].", s.CharacterId(), compartmentType)
			return
		}

		err := compartment.NewProcessor(l, ctx).Sort(s.CharacterId(), compartmentType, p.UpdateTime())
		if err != nil {
			l.WithError(err).Errorf("Failed to send compartment sort command for character [%d].", s.CharacterId())
		}
	}
}
