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
// table, with no side effect. It backs the synchronous
// GET .../transfer-eligibility endpoint (design §3.5) atlas-channel calls
// before it even attempts a WORLD_TRANSFER pending-change request.
func (p *ProcessorImpl) CheckTransferEligibility(characterId uint32, destinationWorldId world.Id) (bool, string, error) {
	c, err := character.NewProcessor(p.l, p.ctx, p.db).GetById()(characterId)
	if err != nil {
		return false, "", err
	}
	reason, ok := p.evaluateTransferEligibility(c, destinationWorldId)
	return ok, reason, nil
}

// evaluateTransferEligibility applies the design §3.6 / §1.6 gate table in the
// documented order — cheapest and most local first, so an obviously-invalid
// request never fans out to eight services — and returns on the first failing
// gate. Every rejection (including an infrastructure failure on a remote
// gate, which fails CLOSED: an escrow gate that cannot be verified is treated
// as blocking, never as silently eligible) is logged at info with tenant,
// character, destination and reason (design §8).
func (p *ProcessorImpl) evaluateTransferEligibility(c character.Model, destinationWorldId world.Id) (string, bool) {
	reject := func(reason string) (string, bool) {
		p.l.Infof("World-transfer eligibility check for character [%d] (tenant [%s]) to world [%d] rejected: %s.", c.Id(), p.t.Id(), destinationWorldId, reason)
		return reason, false
	}

	// Gate 1: a transfer to the world you are already in is not a transfer.
	if destinationWorldId == c.WorldId() {
		return reject("world_same")
	}
	// Gate 2: the v83 client's own CCashShop::CheckTransferWorldPossible
	// refuses to send the request for a GM, so a server that permits it
	// produces a state the client considers impossible.
	if c.GM() != 0 {
		return reject("is_gm")
	}

	// Gate 3: the destination world must exist and have room.
	found, full, err := p.gates.worldStatus(p.l, p.ctx, destinationWorldId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check destination world [%d] status for character [%d] transfer.", destinationWorldId, c.Id())
		return reject("world_unknown")
	}
	if !found {
		return reject("world_unknown")
	}
	if full {
		return reject("world_full")
	}

	// Gate 4: a free character slot in the destination. The cap is remote
	// (atlas-account); the count already held is local.
	slots, err := p.gates.accountSlots(p.l, p.ctx, c.AccountId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check character slots for account [%d] transferring character [%d].", c.AccountId(), c.Id())
		return reject("no_character_slot")
	}
	existing, err := character.NewProcessor(p.l, p.ctx, p.db).GetForAccountInWorld()(c.AccountId(), destinationWorldId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to count existing characters for account [%d] in world [%d].", c.AccountId(), destinationWorldId)
		return reject("no_character_slot")
	}
	if int16(len(existing)) >= slots {
		return reject("no_character_slot")
	}

	// Gate 5: the name must be free in the destination. Name uniqueness for a
	// pending change is already tenant-wide (FR-3.2), so a tenant-scoped check
	// subsumes the per-world one and cannot under-report.
	cs, err := character.NewProcessor(p.l, p.ctx, p.db).GetForName()(c.Name())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check name availability for character [%d] transferring to world [%d].", c.Id(), destinationWorldId)
		return reject("name_taken")
	}
	for _, other := range cs {
		if other.Id() != c.Id() {
			return reject("name_taken")
		}
	}

	// Gate 6: the account must not be banned.
	banned, err := p.gates.banned(p.l, p.ctx, c.AccountId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check ban status for account [%d] transferring character [%d].", c.AccountId(), c.Id())
		return reject("banned")
	}
	if banned {
		return reject("banned")
	}

	// Gate 7: guild MASTER specifically — a non-master member is severed by
	// the saga, not blocked (design §3.6).
	title, inGuild, err := p.gates.guildTitle(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check guild membership for character [%d].", c.Id())
		return reject("is_guild_master")
	}
	if inGuild && title == guildMasterTitle {
		return reject("is_guild_master")
	}

	// Gate 8: family membership. The v83 client itself refuses to send the
	// request for a character in a family (design §1.6).
	inFamily, err := p.gates.inFamily(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check family membership for character [%d].", c.Id())
		return reject("in_family")
	}
	if inFamily {
		return reject("in_family")
	}

	// Gate 9: an open trade is player-visible and player-fixable with its own
	// close flow — blocking beats an unreversible auto-cancel (design §3.6).
	tradeOpen, err := p.gates.tradeOpen(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check open trade rooms for character [%d].", c.Id())
		return reject("trade_open")
	}
	if tradeOpen {
		return reject("trade_open")
	}

	// Gate 10: an open hired merchant, same rationale as gate 9.
	merchantOpen, err := p.gates.merchantOpen(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check hired-merchant status for character [%d].", c.Id())
		return reject("merchant_open")
	}
	if merchantOpen {
		return reject("merchant_open")
	}

	// Gate 11: live MTS listings or bids. Auto-cancelling an auction someone
	// already bid on is not reversible by compensation (design §3.6).
	mtsOpen, err := p.gates.mtsHolding(p.l, p.ctx, c.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check MTS holdings for character [%d].", c.Id())
		return reject("mts_listings_open")
	}
	if mtsOpen {
		return reject("mts_listings_open")
	}

	return "", true
}
