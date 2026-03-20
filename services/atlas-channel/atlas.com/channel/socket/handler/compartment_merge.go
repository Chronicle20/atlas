package handler

import (
	"atlas-channel/compartment"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	inventory2 "github.com/Chronicle20/atlas-packet/inventory/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CompartmentMergeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.CompartmentMergeRequest{}
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
			l.Warnf("Character [%d] issued compartment merge with invalid compartment type [%d].", s.CharacterId(), compartmentType)
			return
		}

		err := compartment.NewProcessor(l, ctx).Merge(s.CharacterId(), compartmentType, p.UpdateTime())
		if err != nil {
			l.WithError(err).Errorf("Failed to send compartment merge command for character [%d].", s.CharacterId())
		}
	}
}
