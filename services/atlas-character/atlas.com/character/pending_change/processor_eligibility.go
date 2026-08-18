package pending_change

import (
	"atlas-character/character"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// guildMasterTitle is atlas-guilds' member.title value for the guild leader
// (services/atlas-guilds/atlas.com/guilds/guild/member/rest.go). Gate 7 is
// guild MASTERY specifically, not membership: a rank-3 member is severed by
// the world-transfer saga, not blocked at request time (design §3.6).
const guildMasterTitle = byte(1)

// gateDeps is the set of narrow remote lookups gates 3, 4, 6-11 of
// evaluateTransferEligibility need, injected as function fields exactly like
// WorldTransferStarterFunc — so a test stubs the one lookup it cares about
// without a network round trip, and production wires the real REST clients in
// requests.go via productionGateDeps.
type gateDeps struct {
	// worldStatus reports whether destinationWorldId exists (found) and, if
	// so, whether it has reached atlas-world's StatusFull capacity (full).
	worldStatus func(l logrus.FieldLogger, ctx context.Context, worldId world.Id) (found bool, full bool, err error)
	// accountSlots is the account's configured character-slot cap
	// (atlas-account's characterSlots). The count of characters the account
	// already holds in the destination world is a LOCAL lookup
	// (character.GetForAccountInWorld) and is not part of this seam.
	accountSlots func(l logrus.FieldLogger, ctx context.Context, accountId uint32) (slots int16, err error)
	// banned reports atlas-ban's account-scoped ban check.
	banned func(l logrus.FieldLogger, ctx context.Context, accountId uint32) (bool, error)
	// guildTitle reports the character's title in whichever guild it belongs
	// to. inGuild is false when the character is in no guild at all, in
	// which case title is meaningless.
	guildTitle func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (title byte, inGuild bool, err error)
	// inFamily reports whether atlas-families has a member row for the
	// character (i.e. GET /families/tree/{characterId} did not 404).
	inFamily func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error)
	// tradeOpen reports whether the character currently occupies an
	// atlas-trades room (either side).
	tradeOpen func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error)
	// merchantOpen reports whether the character has a hired-merchant shop
	// open in atlas-merchant.
	merchantOpen func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error)
	// mtsHolding reports whether the character has any live MTS holding
	// (unsold/expired listing proceeds or items awaiting take-home).
	mtsHolding func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error)
}

// productionGateDeps wires every gate to its real REST client
// (requests.go). NewProcessor uses this by default; tests override individual
// fields via withTransferEligibilityGates so no unit test needs a live
// dependency.
func productionGateDeps() gateDeps {
	return gateDeps{
		worldStatus:  worldStatus,
		accountSlots: accountSlots,
		banned:       banned,
		guildTitle:   guildTitle,
		inFamily:     inFamily,
		tradeOpen:    tradeOpen,
		merchantOpen: merchantOpen,
		mtsHolding:   mtsHoldingOpen,
	}
}

// withTransferEligibilityGates overrides the gate lookups. Unexported (like
// the struct it takes) because gateDeps is an internal wiring seam, not a
// contract atlas-channel or any other caller has any business constructing —
// only this package's own tests and NewProcessor's production default use it.
func (p *ProcessorImpl) withTransferEligibilityGates(g gateDeps) Processor {
	return &ProcessorImpl{
		l:                    p.l,
		ctx:                  p.ctx,
		db:                   p.db,
		t:                    p.t,
		expiry:               p.expiry,
		worldTransferStarter: p.worldTransferStarter,
		gates:                g,
	}
}

// CheckTransferEligibility resolves the character and runs the full gate
// table (both destination-independent and destination-dependent halves —
// see evaluateTransferEligibility), with no side effect. It backs the
// synchronous GET .../transfer-eligibility endpoint (design §3.5)
// atlas-channel calls before it even attempts a WORLD_TRANSFER pending-change
// request.
func (p *ProcessorImpl) CheckTransferEligibility(characterId uint32, destinationWorldId world.Id) (bool, string, error) {
	c, err := character.NewProcessor(p.l, p.ctx, p.db).GetById()(characterId)
	if err != nil {
		return false, "", err
	}
	reason, ok := p.evaluateTransferEligibility(c, destinationWorldId)
	return ok, reason, nil
}

// CheckTransferEligibilityIndependent resolves the character and runs ONLY
// the destination-independent half of the gate table (see
// evaluateTransferEligibilityIndependent), with no side effect. It backs the
// CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE handler in atlas-channel
// (services/atlas-channel/.../cash_shop_check_transfer_world_possible.go),
// which is asked before a destination world is chosen and so cannot supply
// one — closing OQ-7
// (docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-eligibility-reasons.md,
// "The better fix for 2c").
func (p *ProcessorImpl) CheckTransferEligibilityIndependent(characterId uint32) (bool, string, error) {
	c, err := character.NewProcessor(p.l, p.ctx, p.db).GetById()(characterId)
	if err != nil {
		return false, "", err
	}
	reason, ok := p.evaluateTransferEligibilityIndependent(c)
	return ok, reason, nil
}

// gateCheck is one entry of the gate table as data: a self-contained lookup
// (already bound to whatever destinationWorldId it needs, if any) that
// reports its affirmative reason, whether it rejected, and any dependency
// error. Both evaluateTransferEligibility and
// evaluateTransferEligibilityIndependent are nothing but an ordered slice of
// these plus a uniform reject/error handler, so the two entry points never
// duplicate a single gate's underlying logic — they only differ in which
// gates they include and in what order.
type gateCheck func() (reason string, rejected bool, err error)

// runGates evaluates gates in order, on the first rejection or dependency
// error calling reject (which itself performs the info-level log and returns
// (reason, false)). A dependency error always resolves to "check_unavailable"
// (design §6) — see checkWorldStatus etc. below for why: the server failed
// closed but must not assert an affirmative reason it does not actually know
// to hold.
func runGates(gates []gateCheck, reject func(reason string) (string, bool)) (string, bool) {
	for _, gate := range gates {
		reason, rejected, err := gate()
		if err != nil {
			return reject("check_unavailable")
		}
		if rejected {
			return reject(reason)
		}
	}
	return "", true
}

// evaluateTransferEligibility applies the design §3.6 / §1.6 gate table in the
// documented order — cheapest and most local first, so an obviously-invalid
// request never fans out to eight services — and returns on the first failing
// gate. Every rejection is logged at info with tenant, character, destination
// and reason (design §8). A remote dependency error still fails CLOSED — the
// transfer is refused either way — but is reported as the distinct
// "check_unavailable" reason (design §6), never as the gate's affirmative
// reason: the server does not know whether the condition holds, only that it
// could not find out, and the two must not be conflated in what reaches the
// player. The error itself, with the real dependency it came from, is always
// logged at error level before the reject.
//
// The table is split into two halves by whether a gate needs
// destinationWorldId (bug-world-transfer-eligibility-reasons.md, "The better
// fix for 2c"): gates 1, 3, 4, 5 are destination-DEPENDENT and evaluated only
// here, at BUY time, where a destination is finally known. Gates 2, 6-11 are
// destination-INDEPENDENT — evaluateTransferEligibilityIndependent below
// runs exactly that subset, in the same relative order, so CHECK time can
// answer them before a destination is ever chosen. The gate logic itself
// (the checkXxx methods) is shared; only the orchestration differs.
func (p *ProcessorImpl) evaluateTransferEligibility(c character.Model, destinationWorldId world.Id) (string, bool) {
	reject := func(reason string) (string, bool) {
		p.l.Infof("World-transfer eligibility check for character [%d] (tenant [%s]) to world [%d] rejected: %s.", c.Id(), p.t.Id(), destinationWorldId, reason)
		return reason, false
	}

	gates := []gateCheck{
		// Gate 1 (destination-DEPENDENT): a transfer to the world you are
		// already in is not a transfer.
		func() (string, bool, error) { return p.checkWorldSame(c, destinationWorldId) },
		// Gate 2 (destination-INDEPENDENT): the v83 client's own
		// CCashShop::CheckTransferWorldPossible refuses to send the request
		// for a GM, so a server that permits it produces a state the client
		// considers impossible.
		func() (string, bool, error) { return p.checkIsGM(c) },
		// Gate 3 (destination-DEPENDENT): the destination world must exist
		// and have room.
		func() (string, bool, error) { return p.checkWorldStatus(c, destinationWorldId) },
		// Gate 4 (destination-DEPENDENT): a free character slot in the
		// destination.
		func() (string, bool, error) { return p.checkCharacterSlot(c, destinationWorldId) },
		// Gate 5 (destination-DEPENDENT): the name must be free.
		func() (string, bool, error) { return p.checkNameTaken(c) },
		// Gate 6 (destination-INDEPENDENT): the account must not be banned.
		func() (string, bool, error) { return p.checkBanned(c) },
		// Gate 7 (destination-INDEPENDENT): guild MASTER specifically.
		func() (string, bool, error) { return p.checkGuildMaster(c) },
		// Gate 8 (destination-INDEPENDENT): family membership.
		func() (string, bool, error) { return p.checkInFamily(c) },
		// Gate 9 (destination-INDEPENDENT): an open trade.
		func() (string, bool, error) { return p.checkTradeOpen(c) },
		// Gate 10 (destination-INDEPENDENT): an open hired merchant.
		func() (string, bool, error) { return p.checkMerchantOpen(c) },
		// Gate 11 (destination-INDEPENDENT): live MTS listings or bids.
		func() (string, bool, error) { return p.checkMtsHolding(c) },
	}
	return runGates(gates, reject)
}

// evaluateTransferEligibilityIndependent applies ONLY the
// destination-independent gates (2, 6-11 of evaluateTransferEligibility's
// table, in the same relative order) — the subset the CHECK-time handler can
// answer without a destinationWorldId. It shares every gate's underlying
// logic with evaluateTransferEligibility via the same checkXxx methods; the
// destination-dependent gates (world_same, world_unknown, world_full,
// no_character_slot, name_taken) are deliberately NOT evaluated here and
// remain BUY-time only, via evaluateTransferEligibility.
func (p *ProcessorImpl) evaluateTransferEligibilityIndependent(c character.Model) (string, bool) {
	reject := func(reason string) (string, bool) {
		p.l.Infof("World-transfer eligibility check (destination-independent) for character [%d] (tenant [%s]) rejected: %s.", c.Id(), p.t.Id(), reason)
		return reason, false
	}

	gates := []gateCheck{
		func() (string, bool, error) { return p.checkIsGM(c) },
		func() (string, bool, error) { return p.checkBanned(c) },
		func() (string, bool, error) { return p.checkGuildMaster(c) },
		func() (string, bool, error) { return p.checkInFamily(c) },
		func() (string, bool, error) { return p.checkTradeOpen(c) },
		func() (string, bool, error) { return p.checkMerchantOpen(c) },
		func() (string, bool, error) { return p.checkMtsHolding(c) },
	}
	return runGates(gates, reject)
}

// checkWorldSame is gate 1 (destination-DEPENDENT), BUY-time only.
func (p *ProcessorImpl) checkWorldSame(c character.Model, destinationWorldId world.Id) (string, bool, error) {
	if destinationWorldId == c.WorldId() {
		return "world_same", true, nil
	}
	return "", false, nil
}

// checkIsGM is gate 2 (destination-INDEPENDENT).
func (p *ProcessorImpl) checkIsGM(c character.Model) (string, bool, error) {
	if c.GM() != 0 {
		return "is_gm", true, nil
	}
	return "", false, nil
}

// checkWorldStatus is gate 3 (destination-DEPENDENT), BUY-time only.
func (p *ProcessorImpl) checkWorldStatus(c character.Model, destinationWorldId world.Id) (string, bool, error) {
	found, full, err := p.gates.worldStatus(p.l, p.ctx, destinationWorldId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check destination world [%d] status for character [%d] transfer.", destinationWorldId, c.Id())
		return "", false, err
	}
	if !found {
		return "world_unknown", true, nil
	}
	if full {
		return "world_full", true, nil
	}
	return "", false, nil
}

// checkCharacterSlot is gate 4 (destination-DEPENDENT), BUY-time only. The
// cap is remote (atlas-account); the count already held is local.
func (p *ProcessorImpl) checkCharacterSlot(c character.Model, destinationWorldId world.Id) (string, bool, error) {
	slots, err := p.gates.accountSlots(p.l, p.ctx, c.AccountId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check character slots for account [%d] transferring character [%d].", c.AccountId(), c.Id())
		return "", false, err
	}
	existing, err := character.NewProcessor(p.l, p.ctx, p.db).GetForAccountInWorld()(c.AccountId(), destinationWorldId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to count existing characters for account [%d] in world [%d].", c.AccountId(), destinationWorldId)
		return "", false, err
	}
	if int16(len(existing)) >= slots {
		return "no_character_slot", true, nil
	}
	return "", false, nil
}

// checkNameTaken is gate 5 (destination-DEPENDENT), BUY-time only. Name
// uniqueness for a pending change is already tenant-wide (FR-3.2), so a
// tenant-scoped check subsumes the per-world one and cannot under-report.
func (p *ProcessorImpl) checkNameTaken(c character.Model) (string, bool, error) {
	cs, err := character.NewProcessor(p.l, p.ctx, p.db).GetForName()(c.Name())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check name availability for character [%d].", c.Id())
		return "", false, err
	}
	for _, other := range cs {
		if other.Id() != c.Id() {
			return "name_taken", true, nil
		}
	}
	return "", false, nil
}

// checkBanned is gate 6 (destination-INDEPENDENT).
func (p *ProcessorImpl) checkBanned(c character.Model) (string, bool, error) {
	banned, err := p.gates.banned(p.l, p.ctx, c.AccountId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check ban status for account [%d] transferring character [%d].", c.AccountId(), c.Id())
		return "", false, err
	}
	if banned {
		return "banned", true, nil
	}
	return "", false, nil
}

// checkGuildMaster is gate 7 (destination-INDEPENDENT). Guild MASTER
// specifically — a non-master member is severed by the saga, not blocked
// (design §3.6).
func (p *ProcessorImpl) checkGuildMaster(c character.Model) (string, bool, error) {
	title, inGuild, err := p.gates.guildTitle(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check guild membership for character [%d].", c.Id())
		return "", false, err
	}
	if inGuild && title == guildMasterTitle {
		return "is_guild_master", true, nil
	}
	return "", false, nil
}

// checkInFamily is gate 8 (destination-INDEPENDENT). The v83 client itself
// refuses to send the request for a character in a family (design §1.6).
func (p *ProcessorImpl) checkInFamily(c character.Model) (string, bool, error) {
	inFamily, err := p.gates.inFamily(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check family membership for character [%d].", c.Id())
		return "", false, err
	}
	if inFamily {
		return "in_family", true, nil
	}
	return "", false, nil
}

// checkTradeOpen is gate 9 (destination-INDEPENDENT). An open trade is
// player-visible and player-fixable with its own close flow — blocking beats
// an unreversible auto-cancel (design §3.6).
func (p *ProcessorImpl) checkTradeOpen(c character.Model) (string, bool, error) {
	tradeOpen, err := p.gates.tradeOpen(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check open trade rooms for character [%d].", c.Id())
		return "", false, err
	}
	if tradeOpen {
		return "trade_open", true, nil
	}
	return "", false, nil
}

// checkMerchantOpen is gate 10 (destination-INDEPENDENT), same rationale as
// gate 9.
func (p *ProcessorImpl) checkMerchantOpen(c character.Model) (string, bool, error) {
	merchantOpen, err := p.gates.merchantOpen(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check hired-merchant status for character [%d].", c.Id())
		return "", false, err
	}
	if merchantOpen {
		return "merchant_open", true, nil
	}
	return "", false, nil
}

// checkMtsHolding is gate 11 (destination-INDEPENDENT). Live MTS listings or
// bids: auto-cancelling an auction someone already bid on is not reversible
// by compensation (design §3.6).
func (p *ProcessorImpl) checkMtsHolding(c character.Model) (string, bool, error) {
	mtsOpen, err := p.gates.mtsHolding(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check MTS holdings for character [%d].", c.Id())
		return "", false, err
	}
	if mtsOpen {
		return "mts_listings_open", true, nil
	}
	return "", false, nil
}
