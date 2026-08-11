package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// CashItemGachaponHandleFunc handles the Cash Shop Surprise "Open" button
// (CUICashItemGachapon::OnButtonClicked). It performs NO validation: the
// edge does not own the cash locker, and every check — ownership, template,
// capacity — has to happen atomically with the grant, which only
// atlas-cashshop can do. The client self-gates re-clicks via m_nState and
// does not arm the excl-request gate, so nothing is unlocked here.
func CashItemGachaponHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CashItemGachaponButton{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := cashshop.NewProcessor(l, ctx).OpenSurprise(s.AccountId(), s.CharacterId(), p.CashId())
		if err != nil {
			l.WithError(err).Errorf("Unable to request surprise box open for character [%d].", s.CharacterId())
		}
	}
}
