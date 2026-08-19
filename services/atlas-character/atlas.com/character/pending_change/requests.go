package pending_change

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// This file is the production implementation of gateDeps
// (processor_eligibility.go): one narrow REST client per remote gate. Every
// route below was read from the owning service's own resource.go
// registration — see task-11-report.md for the file:line citation of each.
// The RestModel projections these clients unmarshal into are defined in
// rest.go, per this package's file-responsibility convention.

// --- Gate 3: atlas-world -----------------------------------------------

// worldStatusFull mirrors atlas-world's world.StatusFull
// (services/atlas-world/atlas.com/world/world/model.go): capacityStatus == 2.
const worldStatusFull = 2

func worldBaseUrl() string { return requests.RootUrl("WORLDS") }

func requestWorld(worldId world.Id) requests.Request[worldRestModel] {
	return requests.GetRequest[worldRestModel](fmt.Sprintf(worldBaseUrl()+"worlds/%d", worldId))
}

// worldStatus implements gateDeps.worldStatus.
func worldStatus(l logrus.FieldLogger, ctx context.Context, worldId world.Id) (bool, bool, error) {
	rm, err := requestWorld(worldId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, rm.CapacityStatus == worldStatusFull, nil
}

// --- Gate 4: atlas-account -----------------------------------------------

func accountBaseUrl() string { return requests.RootUrl("ACCOUNTS") }

func requestAccount(accountId uint32) requests.Request[accountRestModel] {
	return requests.GetRequest[accountRestModel](fmt.Sprintf(accountBaseUrl()+"accounts/%d", accountId))
}

// accountSlots implements gateDeps.accountSlots.
func accountSlots(l logrus.FieldLogger, ctx context.Context, accountId uint32) (int16, error) {
	rm, err := requestAccount(accountId)(l, ctx)
	if err != nil {
		return 0, err
	}
	return rm.CharacterSlots, nil
}

// --- Gate 6: atlas-ban -----------------------------------------------

func banBaseUrl() string { return requests.RootUrl("BANS") }

func requestBanCheck(accountId uint32) requests.Request[banCheckRestModel] {
	return requests.GetRequest[banCheckRestModel](fmt.Sprintf(banBaseUrl()+"bans/check?accountId=%d", accountId))
}

// banned implements gateDeps.banned.
func banned(l logrus.FieldLogger, ctx context.Context, accountId uint32) (bool, error) {
	rm, err := requestBanCheck(accountId)(l, ctx)
	if err != nil {
		return false, err
	}
	return rm.Banned, nil
}

// --- Gate 7: atlas-guilds -----------------------------------------------

func guildBaseUrl() string { return requests.RootUrl("GUILDS") }

func requestGuildsByMember(characterId uint32) requests.Request[[]guildRestModel] {
	return requests.GetRequest[[]guildRestModel](fmt.Sprintf(guildBaseUrl()+"guilds?filter[members.id]=%d", characterId))
}

// guildTitle implements gateDeps.guildTitle.
func guildTitle(l logrus.FieldLogger, ctx context.Context, characterId uint32) (byte, bool, error) {
	gs, err := requestGuildsByMember(characterId)(l, ctx)
	if err != nil {
		return 0, false, err
	}
	for _, g := range gs {
		for _, m := range g.Members {
			if m.CharacterId == characterId {
				return m.Title, true, nil
			}
		}
	}
	return 0, false, nil
}

// --- Gate 8: atlas-families -----------------------------------------------

func familyBaseUrl() string { return requests.RootUrl("FAMILIES") }

func requestFamilyTree(characterId uint32) requests.Request[[]familyMemberRestModel] {
	return requests.GetRequest[[]familyMemberRestModel](fmt.Sprintf(familyBaseUrl()+"families/tree/%d", characterId))
}

// inFamily implements gateDeps.inFamily. A 404 means the character has no
// family member row at all: not in a family. Any other error (transport,
// decode, non-2xx) is propagated rather than swallowed — a failed check must
// never be reported to the caller as an affirmative family membership; gate 8
// relies on this to decide whether to fail open or closed.
//
// The tree returned on success is bounded (self + senior + juniors +
// siblings — familyMemberRestModel doc above, and getFamilyTreeHandler in
// atlas-families). A character with no relatives still gets a 200 containing
// only itself, so success alone is not "in a family": len(members) > 1 is
// required to have an actual relative present.
func inFamily(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	ms, err := requestFamilyTree(characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return len(ms) > 1, nil
}

// --- Gate 9: atlas-trades -----------------------------------------------

func tradeBaseUrl() string { return requests.RootUrl("TRADES") }

func requestTradeRooms(characterId uint32) requests.Request[[]tradeRoomRestModel] {
	return requests.GetRequest[[]tradeRoomRestModel](fmt.Sprintf(tradeBaseUrl()+"trades/rooms?filter[characterId]=%d", characterId))
}

// tradeOpen implements gateDeps.tradeOpen.
func tradeOpen(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	rs, err := requestTradeRooms(characterId)(l, ctx)
	if err != nil {
		return false, err
	}
	return len(rs) > 0, nil
}

// --- Gate 10: atlas-merchant -----------------------------------------------

func merchantBaseUrl() string { return requests.RootUrl("MERCHANT") }

func requestCharacterMerchants(characterId uint32) requests.Request[[]merchantShopRestModel] {
	return requests.GetRequest[[]merchantShopRestModel](fmt.Sprintf(merchantBaseUrl()+"characters/%d/merchants", characterId))
}

// merchantOpen implements gateDeps.merchantOpen.
func merchantOpen(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	rs, err := requestCharacterMerchants(characterId)(l, ctx)
	if err != nil {
		return false, err
	}
	return len(rs) > 0, nil
}

// --- Gate 11: atlas-mts -----------------------------------------------

func mtsBaseUrl() string { return requests.RootUrl("MTS") }

func requestCharacterHoldings(characterId uint32) requests.Request[[]mtsHoldingRestModel] {
	return requests.GetRequest[[]mtsHoldingRestModel](fmt.Sprintf(mtsBaseUrl()+"characters/%d/mts/holding", characterId))
}

func requestCharacterActiveListings(characterId uint32) requests.Request[[]mtsListingRestModel] {
	return requests.GetRequest[[]mtsListingRestModel](fmt.Sprintf(mtsBaseUrl()+"characters/%d/mts/listings", characterId))
}

// mtsHoldingOpen implements gateDeps.mtsHolding. It blocks on EITHER a live
// holding OR an active listing — the two are mutually exclusive states of the
// same escrowed item, and either one is exactly the un-reversible-by-
// compensation situation gate 11 exists to catch (design §3.6).
func mtsHoldingOpen(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	holdings, err := requestCharacterHoldings(characterId)(l, ctx)
	if err != nil {
		return false, err
	}
	if len(holdings) > 0 {
		return true, nil
	}
	listings, err := requestCharacterActiveListings(characterId)(l, ctx)
	if err != nil {
		return false, err
	}
	return len(listings) > 0, nil
}

// --- Gate 12: atlas-parcel -----------------------------------------------

func parcelBaseUrl() string { return requests.RootUrl("PARCEL") }

func requestParcelStatus(characterId uint32) requests.Request[parcelStatusRestModel] {
	return requests.GetRequest[parcelStatusRestModel](fmt.Sprintf(parcelBaseUrl()+"characters/%d/parcel-status", characterId))
}

// parcelPending implements gateDeps.parcelPending.
func parcelPending(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	rm, err := requestParcelStatus(characterId)(l, ctx)
	if err != nil {
		return false, err
	}
	return rm.InFlight, nil
}

// --- World-transfer severance snapshot ------------------------------------
//
// The five-step WorldTransfer saga (design §3.11) needs the character's guild,
// party and buddy state captured BEFORE any severance runs, because the
// compensations are built from the saga payloads and nothing else: the guild
// re-add needs the exact prior Title, and the buddy restore needs the ids that
// are about to be deleted. Reading them after the fact is impossible.
//
// Every helper below distinguishes "no membership" from "lookup failed". A
// zero GuildId/PartyId is a LEGITIMATE skip signal the handlers act on
// (handleLeaveGuildForTransfer self-completes on GuildId == 0), so a failed
// lookup must never degrade into a zero — it would silently skip a real
// severance and leave the character in a guild in the world they just left.
// Errors propagate; the saga is not dispatched at all.

// guildMembership resolves the character's guild id and the rank they hold.
// found == false means the character is genuinely in no guild.
func guildMembership(l logrus.FieldLogger, ctx context.Context, characterId uint32) (uint32, byte, bool, error) {
	gs, err := requestGuildsByMember(characterId)(l, ctx)
	if err != nil {
		return 0, 0, false, err
	}
	for _, g := range gs {
		for _, m := range g.Members {
			if m.CharacterId != characterId {
				continue
			}
			id, convErr := strconv.ParseUint(g.Id, 10, 32)
			if convErr != nil {
				// A membership we cannot address is worse than none: the saga
				// would emit a LEAVE against guild 0. Fail the dispatch.
				return 0, 0, false, fmt.Errorf("guild id [%s] for character [%d] is not numeric: %w", g.Id, characterId, convErr)
			}
			return uint32(id), m.Title, true, nil
		}
	}
	return 0, 0, false, nil
}

func partyBaseUrl() string { return requests.RootUrl("PARTIES") }

func requestPartiesByMember(characterId uint32) requests.Request[[]partyRestModel] {
	return requests.GetRequest[[]partyRestModel](fmt.Sprintf(partyBaseUrl()+"parties?filter[members.id]=%d", characterId))
}

// partyMembership resolves the character's party id. found == false means the
// character is genuinely in no party.
func partyMembership(l logrus.FieldLogger, ctx context.Context, characterId uint32) (uint32, bool, error) {
	ps, err := requestPartiesByMember(characterId)(l, ctx)
	if err != nil {
		return 0, false, err
	}
	if len(ps) == 0 {
		return 0, false, nil
	}
	p := ps[0]
	id, convErr := strconv.ParseUint(p.Id, 10, 32)
	if convErr != nil {
		return 0, false, fmt.Errorf("party id [%s] for character [%d] is not numeric: %w", p.Id, characterId, convErr)
	}
	return uint32(id), true, nil
}

func buddyBaseUrl() string { return requests.RootUrl("BUDDIES") }

func requestBuddyList(characterId uint32) requests.Request[buddyListRestModel] {
	return requests.GetRequest[buddyListRestModel](fmt.Sprintf(buddyBaseUrl()+"characters/%d/buddy-list", characterId))
}

// buddyIds captures every id the sever step must remove, in both directions.
// A character with no buddy list at all (404) genuinely has no buddies; any
// other error propagates. Pending entries are excluded: they are unaccepted
// invites, not mutual relationships, so REQUEST_DELETE has nothing symmetric
// to remove and the compensation would restore a relationship that never
// existed.
func buddyIds(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]uint32, error) {
	bl, err := requestBuddyList(characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]uint32, 0, len(bl.Buddies))
	for _, b := range bl.Buddies {
		if b.Pending {
			continue
		}
		ids = append(ids, b.CharacterId)
	}
	return ids, nil
}
