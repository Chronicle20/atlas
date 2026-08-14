package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// CashShopCheckTransferWorldPossibleHandleFunc handles the standalone
// serverbound WORLD_TRANSFER op — the cash shop's "may this character change
// worlds?" request, sent when the player buys the 5401000 world-transfer item
// and the client-side CCashShop::CheckTransferWorldPossible gate passes. It is
// NOT an arm of CashShopOperationHandle: the request has its own opcode and no
// leading mode byte, so it is registered by name in main.go like any other
// standalone handler.
//
// This handler decodes and logs. Answering the request means emitting
// CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT, whose codec does not exist yet
// (that clientbound op is its own packet-phase task); until then there is no
// reply this channel could send that the client would understand. Registering
// the decode now is what stops the client's request from landing as an
// "unhandled message op" and lets the route be verified on every version.
func CashShopCheckTransferWorldPossibleHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CheckTransferWorldPossible{}
		p.Decode(l, ctx)(r, readerOptions)
		// p.String() REDACTS the credential the body carries (the account
		// second password / birthday code). Never log p.Spw() or
		// p.BirthDate().
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
	}
}
