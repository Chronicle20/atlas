package handler

import (
	"atlas-channel/account"
	"atlas-channel/session"
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrCredentialMismatch is the sentinel every gated arm maps to its own
// *_FAILED body with the errors-table key "INVALID_BIRTHDAY".
var ErrCredentialMismatch = errors.New("secondary credential mismatch")

// credentialMatches decides the gate. It PASSES when the account has no
// credential of the applicable kind set: a server that never collected the
// value cannot meaningfully check it, and failing closed would make every
// gift, ring, and rebate unusable on a fresh tenant (design section 2 step 4).
func credentialMatches(usesPIC bool, storedPIC string, storedBirthDate uint32, spw string, birthday uint32) bool {
	if usesPIC {
		if storedPIC == "" {
			return true
		}
		return storedPIC == spw
	}
	if storedBirthDate == 0 {
		return true
	}
	return storedBirthDate == birthday
}

// verifySecondaryCredential resolves the session's account, applies
// credentialMatches, records the attempt, and returns ErrCredentialMismatch
// on failure.
//
//nolint:unused // First consumers land in plan Tasks 12/14/17/20 (gift, ring, rebate arms); remove this directive when Task 12 wires the first caller.
func verifySecondaryCredential(l logrus.FieldLogger, ctx context.Context) func(s session.Model, spw string, birthday uint32) error {
	return func(s session.Model, spw string, birthday uint32) error {
		a, err := account.NewProcessor(l, ctx).GetById(s.AccountId())
		if err != nil {
			return err
		}

		t := tenant.MustFromContext(ctx)
		usesPIC := t.IsRegion("GMS") && t.MajorAtLeast(95)

		if credentialMatches(usesPIC, a.PIC(), a.BirthDate(), spw, birthday) {
			if (usesPIC && a.PIC() == "") || (!usesPIC && a.BirthDate() == 0) {
				l.Debugf("Secondary credential gate is inert for account [%d]: no credential of the applicable kind is set.", s.AccountId())
			}
			return nil
		}

		if _, _, rErr := account.NewProcessor(l, ctx).RecordPicAttempt(s.AccountId(), false, remoteIpAddress(s), ""); rErr != nil {
			l.WithError(rErr).Errorf("Unable to record PIC attempt for account [%d].", s.AccountId())
		}
		return ErrCredentialMismatch
	}
}
