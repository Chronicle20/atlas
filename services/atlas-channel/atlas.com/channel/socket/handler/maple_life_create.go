package handler

import (
	"atlas-channel/account"
	character2 "atlas-channel/character"
	"atlas-channel/character/factory"
	"atlas-channel/maplelife"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	mlcb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// accountSlotsFunc is the test seam for the slot-limit gate's account lookup
// (package-var injection precedent: cashItemInSlotFunc in
// character_cash_item_use.go). Tests substitute this rather than requiring a
// live REST round trip to atlas-account.
var accountSlotsFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32) (int16, error) {
	a, err := account.NewProcessor(l, ctx).GetById(accountId)
	if err != nil {
		return 0, err
	}
	return a.CharacterSlots(), nil
}

// charactersInWorldFunc is the test seam for the slot-limit gate's character
// count. character.Processor.GetForAccountInWorld is already a paged,
// drained provider (character/processor.go:248-258), so no further paging
// belongs here.
var charactersInWorldFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32, worldId world.Id) ([]character2.Model, error) {
	return character2.NewProcessor(l, ctx).GetForAccountInWorld(accountId, worldId)
}

// newFactoryProcessorFunc is the test seam over factory.NewProcessor itself,
// one level below createMapleLifeFunc (package-var injection precedent:
// cashItemInSlotFunc in character_cash_item_use.go). design.md §11 A8's
// finding was that the prior placeholder could ship a wrong wire mapping
// undetected because every test swapped that placeholder wholesale -- so this
// task puts the seam here instead, letting createMapleLifeFunc's own
// al0..al3/gender/currentClass/sp mapping run for real in tests, and letting
// a test observe the mapped ints it actually forwards to the factory.
var newFactoryProcessorFunc = func(l logrus.FieldLogger, ctx context.Context) factory.Processor {
	return factory.NewProcessor(l, ctx)
}

// createMapleLifeFunc is the test seam over the atlas-character-factory
// Maple Life creation call (package-var injection precedent:
// cashItemInSlotFunc in character_cash_item_use.go). It replaces Task 13's
// prior placeholder, which forwarded al0..al3 positionally into
// face/hair/hairColor/skinColor and nCurrentClass into the seed contract's
// class index -- wrong on all three counts (design.md §11 A8). The channel
// now forwards only what the player chose; the factory owns what a Maple
// Life character is.
var createMapleLifeFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32, worldId world.Id, sub cashsb.ItemUseMapleLife) (string, error) {
	return newFactoryProcessorFunc(l, ctx).CreateMapleLife(
		accountId, worldId, sub.Name(),
		uint32(sub.CurrentClass()), byte(sub.Gender()),
		uint32(sub.AL0()),
		uint32(sub.AL1()/10)*10,
		uint32(sub.AL2()%10),
		byte(sub.AL3()),
		byte(sub.SP()),
	)
}

// handleMapleLifeCreate is the submit flow for Cash/0543
// (CUICharacterSaleDlg::SendCreateNewCharacter, task-246 Task 1/derivation.md
// §2). It replaces Task 11's beginMapleLife, which wired this arm to the
// wrong signal (bug-543-is-the-submit-not-the-open.md): there is no
// open-time packet at all, so the pending maplelife registry entry is now
// created here, directly in PhaseSubmitted, on a successful factory call --
// never before.
//
// Runs design §5.2's gates 2-5 (gate 1, "a live pending PhaseOpen record,"
// has no subject under this rewire and is dropped -- the record this arm
// would have checked no longer exists before this very packet arrives):
//
//  1. ownership + classification (FR-5.3)
//  2. slot limit
//  3. name re-check (FR-4.5) -- the only duplicate gate, since design C2 found
//     the factory's seed path performs no duplicate check of its own
//  4. class ordinal and SP, wire-range only (design.md §11 A7)
//
// Nothing here consumes the item (FR-5.1): every gate below leaves the cash
// slot untouched, and consumption belongs to Task 14's CREATED path alone.
// TOCTOU on gates 2 (slot limit) and 3 (name re-check) is accepted: an
// account has exactly one channel session, so the residual race (two
// requests interleaving between the read and the eventual write) surfaces as
// a saga FAILED -> MapleLifeErrorUnknownError -> item retained, not a
// duplicate character or a lost item.
func handleMapleLifeCreate(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, source slot.Position, sub cashsb.ItemUseMapleLife) {
	return func(s session.Model, itemId item.Id, source slot.Position, sub cashsb.ItemUseMapleLife) {
		t := tenant.MustFromContext(ctx)
		accountId := s.AccountId()
		worldId := s.WorldId()

		fail := func(reason string, args ...interface{}) {
			l.Warnf(reason, args...)
			if err := session.Announce(l)(ctx)(wp)(mlcb.MapleLifeErrorWriter)(mlcb.MapleLifeErrorBody(mlcb.MapleLifeErrorUnknownError))(s); err != nil {
				l.WithError(err).Errorf("Unable to write Maple Life error for account [%d].", accountId)
			}
		}

		// Gate 2 -- ownership + classification. This re-derives ownership of
		// the SAME item/slot the enclosing arm (character_cash_item_use.go:
		// 61-66) already validated the common ItemUse prefix against, rather
		// than reusing that call's result. At the moment this runs
		// synchronously, in the same call as that upstream check with no I/O
		// between them, the template-id half is genuinely redundant with it,
		// and the classification half is also currently redundant --
		// item.GetClassification(itemId) == ClassificationCharacterCreation
		// is exactly the test that routed this packet into this arm
		// (character_cash_item_use.go:800). We keep the re-check anyway as
		// defense-in-depth: it is cheap, and gates 3/4 below make two live
		// REST calls, so a future edit that reorders this gate after either
		// of them would silently reopen a real TOCTOU window.
		templateId, err := cashItemInSlotFunc(l, ctx, s.CharacterId(), int16(source))
		if err != nil || item.Id(templateId) != itemId {
			fail("Character [%d] submitted Maple Life creation for item [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), itemId, source)
			return
		}
		if item.GetClassification(item.Id(templateId)) != item.ClassificationCharacterCreation {
			fail("Character [%d] submitted Maple Life creation for item [%d] in slot [%d], but its classification is no longer Maple Life.", s.CharacterId(), itemId, source)
			return
		}

		// Gate 3 -- slot limit.
		slots, err := accountSlotsFunc(l, ctx, accountId)
		if err != nil {
			fail("Unable to resolve character slots for account [%d]: %v.", accountId, err)
			return
		}
		chars, err := charactersInWorldFunc(l, ctx, accountId, worldId)
		if err != nil {
			fail("Unable to resolve existing characters for account [%d] in world [%d]: %v.", accountId, worldId, err)
			return
		}
		if len(chars) >= int(slots) {
			fail("Account [%d] attempted Maple Life creation in world [%d] with [%d] of [%d] slots already used.", accountId, worldId, len(chars), slots)
			return
		}

		// Gate 4 -- name re-check (FR-4.5). NameScopeWorld, matching the
		// earlier duplicate-name probe (Task 12) -- the only collision that
		// matters for a creation is within the world the character will
		// actually be created in.
		res, err := mapleLifeNameValidityFunc(l, ctx, sub.Name(), worldId, character2.NameScopeWorld)
		if err != nil {
			fail("Unable to check name validity of [%s] for account [%d]: %v.", sub.Name(), accountId, err)
			return
		}
		if !res.Valid {
			l.Infof("Account [%d] submitted Maple Life creation with name [%s], but it is no longer available: [%s].", accountId, sub.Name(), res.Reason)
			if err := session.Announce(l)(ctx)(wp)(mlcb.MapleLifeErrorWriter)(mlcb.MapleLifeErrorBody(mlcb.MapleLifeErrorNameTakenAtSubmit))(s); err != nil {
				l.WithError(err).Errorf("Unable to write Maple Life name-taken error for account [%d].", accountId)
			}
			return
		}

		// Note: account and world are sourced from the session, never the
		// packet (FR-4.2): the same reasoning Task 11's beginMapleLife doc
		// comment gave for the (now-removed) open path applies unchanged to
		// the submit -- the session is the only trustworthy source of
		// "which account is this" for an entry point that exists precisely
		// because the account has no character yet to authenticate through.
		// Unlike the brief's original description, ItemUseMapleLife (Task 6)
		// carries no accountId/worldId field on the wire at all (derivation.md
		// §2's field list: sName, al[0..3], nGender, nCurrentClass, nSP,
		// update_time) -- so there is no packet-carried value to compare
		// against or log a mismatch for. This is a defect in the brief the
		// controller addendum did not resolve; see task-13-report.md.

		// Gate 5 -- class ordinal and SP (design.md §11 A7). The channel
		// checks only what the WIRE constrains: m_nCurrentClass cycles % 5
		// and m_nSP cycles % 11 in the client's own OnButtonClicked
		// (gms_v95 @0x77edc0), so anything outside those ranges did not
		// come from the dialog. WHICH ordinals are actually configured, and
		// whether the class offers an SP step at all, is the factory's call
		// -- the channel does not read the mapleLife block.
		//
		// Rejected, never clamped: A7 supersedes A4's earlier
		// clamp-and-continue, because a clamped submit silently creates a
		// character the player did not ask for.
		if sub.CurrentClass() < 0 || sub.CurrentClass() > 4 {
			fail("Character [%d] submitted Maple Life creation with class ordinal [%d], outside the client's 0..4 range.", s.CharacterId(), sub.CurrentClass())
			return
		}
		if sub.SP() < 0 || sub.SP() > 10 {
			fail("Character [%d] submitted Maple Life creation with sp [%d], outside the client's 0..10 range.", s.CharacterId(), sub.SP())
			return
		}

		transactionId, err := createMapleLifeFunc(l, ctx, accountId, worldId, sub)
		if err != nil {
			arm := logCreateFailure(l, accountId, err)
			l.Warnf("Maple Life character creation for account [%d] was rejected by the factory.", accountId)
			if announceErr := session.Announce(l)(ctx)(wp)(mlcb.MapleLifeErrorWriter)(mlcb.MapleLifeErrorBody(arm))(s); announceErr != nil {
				l.WithError(announceErr).Errorf("Unable to write Maple Life error for account [%d].", accountId)
			}
			return
		}

		// Success: write nothing to the client -- the outcome arrives later
		// on the seed topic (Task 14). Record the PhaseSubmitted entry so
		// that consumer can correlate it back to this account.
		maplelife.GetRegistry().Put(t, accountId, maplelife.Entry{
			CharacterId:   s.CharacterId(),
			WorldId:       worldId,
			ItemId:        itemId,
			Slot:          source,
			Phase:         maplelife.PhaseSubmitted,
			TransactionId: transactionId,
			CandidateName: sub.Name(),
			At:            time.Now(),
		})
	}
}

// logCreateFailure classifies a createMapleLifeFunc error by HTTP status for
// the log line, and returns the MAPLELIFE_ERROR arm the caller should write.
// requests.ErrConflict -- the factory's own name-collision check -- maps onto
// the same NAME_TAKEN_AT_SUBMIT arm gate 4's re-check writes; every other
// error (bad request, transport, unknown) maps onto the generic
// MapleLifeErrorUnknownError, since there are only three MAPLELIFE_ERROR arms
// (libs/atlas-packet/maplelife/clientbound/error.go:51,55,60).
func logCreateFailure(l logrus.FieldLogger, accountId uint32, err error) string {
	if errors.Is(err, requests.ErrConflict) {
		l.WithError(err).Warnf("Account [%d]'s Maple Life submission was rejected because the name was taken by the time the character factory processed it.", accountId)
		return mlcb.MapleLifeErrorNameTakenAtSubmit
	}
	if errors.Is(err, requests.ErrBadRequest) {
		l.WithError(err).Warnf("Account [%d]'s Maple Life submission was rejected as invalid (look or name) by the character factory.", accountId)
		return mlcb.MapleLifeErrorUnknownError
	}
	l.WithError(err).Errorf("Account [%d]'s Maple Life submission failed calling the character factory.", accountId)
	return mlcb.MapleLifeErrorUnknownError
}
