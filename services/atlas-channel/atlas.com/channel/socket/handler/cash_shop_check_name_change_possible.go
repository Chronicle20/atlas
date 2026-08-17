package handler

import (
	"atlas-channel/account"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"net"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// checkPossibleAccountGetByIdFunc and checkPossibleRecordPicAttemptFunc are the
// seams both check handlers (this file and
// cash_shop_check_transfer_world_possible.go) call the account package
// through, so tests can swap them the way TestCouponCode's
// couponRedemptionRequestFunc does (cash_shop_coupon_code.go) without a live
// atlas-account round trip.
var checkPossibleAccountGetByIdFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32) (account.Model, error) {
	return account.NewProcessor(l, ctx).GetById(accountId)
}

var checkPossibleRecordPicAttemptFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32, success bool, ipAddress string) (int, bool, error) {
	return account.NewProcessor(l, ctx).RecordPicAttempt(accountId, success, ipAddress, "")
}

// CashShopCheckNameChangePossibleHandleFunc handles the standalone serverbound
// NAME_TRANSFER op — the cash shop's "may this character be renamed?" request,
// sent when the player buys the 5400000 name-change item. It is NOT an arm of
// CashShopOperationHandle: the request has its own opcode and no leading mode
// byte, so it is registered by name in main.go like any other standalone
// handler.
//
// The body carries only characterId and the account's second-password
// credential (an 8-digit birthday code pre-v95, a string SPW on v95+ — see
// cashsb.CheckNameChangePossible's doc comment). This op does NOT carry a
// candidate name: name availability travels on the separate
// CASHSHOP_CHECK_NAME_CHANGE op, and destination/eligibility gates do not
// apply here at all (name changes are not world-scoped). So the only thing
// this handler validates is the credential itself, against the account's
// stored PIC (v95+) or stored BirthDate (pre-v95, task-227 Task 26 ruling 3),
// with the PIC-attempt lockout counter behind it (ruling 4).
//
// The four wire arms this op's clientbound result can express
// (cashcb.CheckNameChangePossible*) are ALLOWED, ALREADY_SUBMITTED,
// REQUEST_LIMIT_RECENT, REQUEST_LIMIT_REQUESTED and the UNKNOWN_ERROR default.
// A credential mismatch that has not yet tripped the lockout has no dedicated
// arm — the client's own switch has no "wrong password" text on this op (see
// the result codec's doc comment) — so it answers UNKNOWN_ERROR, the client's
// own catch-all. A mismatch that DOES trip the lockout answers
// REQUEST_LIMIT_RECENT, the closer of the two rate-limit arms to "you just got
// locked out".
//
// Evaluating whether a NAME_CHANGE is already pending (ALREADY_SUBMITTED) is
// NOT wired here: it needs a pending-changes lookup this task's brief does not
// list a REST client for (see the task-227 Task 26 report for the file-list
// evidence). Left as a documented gap rather than invented.
func CashShopCheckNameChangePossibleHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CheckNameChangePossible{}
		p.Decode(l, ctx)(r, readerOptions)
		// p.String() REDACTS the credential the body carries (the account
		// second password / birthday code). Never log p.Spw() or
		// p.BirthDate().
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		a, err := checkPossibleAccountGetByIdFunc(l, ctx, s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve account [%d] for name-change credential validation.", s.AccountId())
			announceNameChangePossible(l, ctx, wp, s, cashcb.CheckNameChangePossibleResultBody(p.CharacterId(), cashcb.CheckNameChangePossibleUnknownError, 0))
			return
		}

		ipAddress := remoteIpAddress(s)

		if !nameChangeCredentialMatches(ctx, p, a) {
			l.Debugf("Incorrect name-change credential for account [%d].", s.AccountId())
			_, limitReached, rErr := checkPossibleRecordPicAttemptFunc(l, ctx, s.AccountId(), false, ipAddress)
			if rErr != nil {
				l.WithError(rErr).Errorf("Unable to record PIC attempt for account [%d].", s.AccountId())
			}
			if limitReached {
				announceNameChangePossible(l, ctx, wp, s, cashcb.CheckNameChangePossibleResultBody(p.CharacterId(), cashcb.CheckNameChangePossibleRequestLimitRecent, 0))
				return
			}
			announceNameChangePossible(l, ctx, wp, s, cashcb.CheckNameChangePossibleResultBody(p.CharacterId(), cashcb.CheckNameChangePossibleUnknownError, 0))
			return
		}

		if _, _, rErr := checkPossibleRecordPicAttemptFunc(l, ctx, s.AccountId(), true, ipAddress); rErr != nil {
			l.WithError(rErr).Errorf("Unable to record PIC attempt for account [%d].", s.AccountId())
		}

		announceNameChangePossible(l, ctx, wp, s, cashcb.CheckNameChangePossibleResultAllowedBody(p.CharacterId(), a.BirthDate()))
	}
}

// nameChangeCredentialMatches implements task-227 Task 26 ruling 3: on v95+
// (cashsb.CredentialIsString — the SAME predicate the codec's own Decode used,
// per ruling 2) the credential is the account's PIC; pre-v95 it is the
// account's stored BirthDate, and a stored BirthDate of 0 means UNSET and
// FAILS the check regardless of what the client sent — it is never populated
// from the wire, and a 0-vs-0 comparison would be a trust-on-first-use
// authentication bypass for every account that has not had a birth date
// provisioned (which today is every account).
func nameChangeCredentialMatches(ctx context.Context, p cashsb.CheckNameChangePossible, a account.Model) bool {
	if cashsb.CredentialIsString(ctx) {
		return p.Spw() == a.PIC()
	}
	if a.BirthDate() == 0 {
		return false
	}
	return p.BirthDate() == a.BirthDate()
}

// announceNameChangePossible writes CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT.
func announceNameChangePossible(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body packet.Encode) {
	if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopCheckNameChangePossibleResultWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to write name-change-possible result for character [%d].", s.CharacterId())
	}
}

// remoteIpAddress mirrors atlas-login's character_selected_pic.go IP
// extraction — used as the ipAddress attribute recorded against the
// PIC-attempt lockout counter.
func remoteIpAddress(s session.Model) string {
	addr := s.GetRemoteAddress()
	if addr == nil {
		return ""
	}
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return host
	}
	return ""
}
