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

// This file is the production implementation of gateDeps (eligibility.go):
// one narrow REST client per remote gate. Every route below was read from the
// owning service's own resource.go registration — see task-11-report.md for
// the file:line citation of each.

// --- Gate 3: atlas-world -----------------------------------------------

// worldStatusFull mirrors atlas-world's world.StatusFull
// (services/atlas-world/atlas.com/world/world/model.go): capacityStatus == 2.
const worldStatusFull = 2

// worldRestModel is the minimal projection of atlas-world's GET
// /worlds/{worldId} (services/atlas-world/atlas.com/world/world/resource.go:306,
// rest.go:13). The server's RestModel also carries a "channels"
// relationship, so the no-op stubs are required even though this client
// never reads it (libs/atlas-rest CLAUDE.md).
type worldRestModel struct {
	Id             string `json:"-"`
	CapacityStatus uint16 `json:"capacityStatus"`
}

func (r worldRestModel) GetName() string                                   { return "worlds" }
func (r worldRestModel) GetID() string                                     { return r.Id }
func (r *worldRestModel) SetID(id string) error                            { r.Id = id; return nil }
func (r *worldRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *worldRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

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

// accountRestModel is the minimal projection of atlas-account's GET
// /accounts/{accountId}
// (services/atlas-account/atlas.com/account/account/resource.go:35,
// rest.go:23). No relationships block.
type accountRestModel struct {
	Id             string `json:"-"`
	CharacterSlots int16  `json:"characterSlots"`
}

func (r accountRestModel) GetName() string        { return "accounts" }
func (r accountRestModel) GetID() string          { return r.Id }
func (r *accountRestModel) SetID(id string) error { r.Id = id; return nil }

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

// banCheckRestModel is the minimal projection of atlas-ban's GET
// /bans/check?accountId={id}
// (services/atlas-ban/atlas.com/ban/ban/resource.go:28,186; rest.go:64 —
// the exact query shape atlas-account's own ban client uses,
// services/atlas-account/atlas.com/account/ban/requests.go). No
// relationships block.
type banCheckRestModel struct {
	Id     string `json:"-"`
	Banned bool   `json:"banned"`
}

func (r banCheckRestModel) GetName() string        { return "ban-checks" }
func (r banCheckRestModel) GetID() string          { return r.Id }
func (r *banCheckRestModel) SetID(id string) error { r.Id = id; return nil }

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

// guildMemberRestModel mirrors atlas-guilds' member.RestModel
// (services/atlas-guilds/atlas.com/guilds/guild/member — same shape as the
// atlas-channel guild/member client). Title == 1 is the guild master.
type guildMemberRestModel struct {
	CharacterId uint32 `json:"characterId"`
	Title       byte   `json:"title"`
}

// guildRestModel is the minimal projection of atlas-guilds' GET
// /guilds?filter[members.id]={characterId}
// (services/atlas-guilds/atlas.com/guilds/guild/resource.go:23). Members are
// a plain JSON attribute array in this response, not a JSON:API
// relationship, so no relationship stubs are needed (matches
// services/atlas-channel/atlas.com/channel/guild/rest.go).
type guildRestModel struct {
	Id      string                 `json:"-"`
	Members []guildMemberRestModel `json:"members"`
}

func (r guildRestModel) GetName() string        { return "guilds" }
func (r guildRestModel) GetID() string          { return r.Id }
func (r *guildRestModel) SetID(id string) error { r.Id = id; return nil }

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

// familyMemberRestModel is the minimal projection of atlas-families' GET
// /families/tree/{characterId}
// (services/atlas-families/atlas.com/family/family/resource.go:28). A 404
// (ErrMemberNotFound) means the character has no family member row at all;
// any other success means it does, even a solo tree of just itself. No
// relationships block.
type familyMemberRestModel struct {
	Id string `json:"-"`
}

func (r familyMemberRestModel) GetName() string        { return "family-tree-members" }
func (r familyMemberRestModel) GetID() string          { return r.Id }
func (r *familyMemberRestModel) SetID(id string) error { r.Id = id; return nil }

func familyBaseUrl() string { return requests.RootUrl("FAMILIES") }

func requestFamilyTree(characterId uint32) requests.Request[[]familyMemberRestModel] {
	return requests.GetRequest[[]familyMemberRestModel](fmt.Sprintf(familyBaseUrl()+"families/tree/%d", characterId))
}

// inFamily implements gateDeps.inFamily.
func inFamily(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	_, err := requestFamilyTree(characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// --- Gate 9: atlas-trades -----------------------------------------------

// tradeRoomRestModel is the minimal projection of atlas-trades' GET
// /trades/rooms?filter[characterId]={id}
// (services/atlas-trades/atlas.com/trades/trade/resource.go:51 — the exact
// query shape services/atlas-channel/atlas.com/channel/trade/requests.go
// already uses; the filter matches either side of the room). No
// relationships block.
type tradeRoomRestModel struct {
	Id string `json:"-"`
}

func (r tradeRoomRestModel) GetName() string        { return "rooms" }
func (r tradeRoomRestModel) GetID() string          { return r.Id }
func (r *tradeRoomRestModel) SetID(id string) error { r.Id = id; return nil }

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

// merchantShopRestModel is the minimal projection of atlas-merchant's GET
// /characters/{characterId}/merchants
// (services/atlas-merchant/atlas.com/merchant/shop/resource.go:40). The
// server's shop RestModel carries a "listings" relationship, so the no-op
// stubs are required even though this client never reads it.
type merchantShopRestModel struct {
	Id string `json:"-"`
}

func (r merchantShopRestModel) GetName() string                                   { return "merchants" }
func (r merchantShopRestModel) GetID() string                                     { return r.Id }
func (r *merchantShopRestModel) SetID(id string) error                            { r.Id = id; return nil }
func (r *merchantShopRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *merchantShopRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

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

// mtsHoldingRestModel is the minimal projection of atlas-mts's GET
// /characters/{characterId}/mts/holding
// (services/atlas-mts/atlas.com/mts/holding/resource.go:36 — the exact
// route services/atlas-channel/atlas.com/channel/mts/holding/requests.go
// already uses). No relationships block.
type mtsHoldingRestModel struct {
	Id string `json:"-"`
}

func (r mtsHoldingRestModel) GetName() string        { return "holdings" }
func (r mtsHoldingRestModel) GetID() string          { return r.Id }
func (r *mtsHoldingRestModel) SetID(id string) error { r.Id = id; return nil }

func mtsBaseUrl() string { return requests.RootUrl("MTS") }

func requestCharacterHoldings(characterId uint32) requests.Request[[]mtsHoldingRestModel] {
	return requests.GetRequest[[]mtsHoldingRestModel](fmt.Sprintf(mtsBaseUrl()+"characters/%d/mts/holding", characterId))
}

// mtsListingRestModel is the minimal projection of atlas-mts's GET
// /characters/{characterId}/mts/listings — a seller's ACTIVE listings across
// worlds (services/atlas-mts/atlas.com/mts/listing/resource.go, route
// "get_character_active_listings"). No relationships block.
//
// This is a SEPARATE endpoint from the holding one above and deliberately so:
// a listing becomes a holding only on cancel/expiry (Cancel/Expire in
// services/atlas-mts/atlas.com/mts/listing/processor.go), so an ACTIVE
// listing never shows up as a holding. Checking holdings alone let a
// character with a live, un-settled auction pass this gate and transfer,
// stranding the listing in the world they just left — the fix-round-1
// finding this type exists to close.
type mtsListingRestModel struct {
	Id string `json:"-"`
}

func (r mtsListingRestModel) GetName() string        { return "listings" }
func (r mtsListingRestModel) GetID() string          { return r.Id }
func (r *mtsListingRestModel) SetID(id string) error { r.Id = id; return nil }

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

// partyRestModel is the minimal projection of atlas-parties' GET
// /parties?filter[members.id]={characterId}
// (services/atlas-parties/atlas.com/parties/party/resource.go:36 — the exact
// query shape services/atlas-channel/atlas.com/channel/party/requests.go
// already uses). Members are a JSON:API relationship on this resource, not a
// plain attribute, so the reference stubs are required for unmarshalling; only
// the id is read.
type partyRestModel struct {
	Id string `json:"-"`
}

func (r partyRestModel) GetName() string                                   { return "parties" }
func (r partyRestModel) GetID() string                                     { return r.Id }
func (r *partyRestModel) SetID(id string) error                            { r.Id = id; return nil }
func (r *partyRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *partyRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

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
	for _, p := range ps {
		id, convErr := strconv.ParseUint(p.Id, 10, 32)
		if convErr != nil {
			return 0, false, fmt.Errorf("party id [%s] for character [%d] is not numeric: %w", p.Id, characterId, convErr)
		}
		return uint32(id), true, nil
	}
	return 0, false, nil
}

// buddyEntryRestModel mirrors atlas-buddies' buddy.RestModel
// (services/atlas-buddies/atlas.com/buddies/buddy/rest.go:7). Only the id is
// read; Pending distinguishes a live buddy from an unaccepted invite.
type buddyEntryRestModel struct {
	CharacterId uint32 `json:"characterId"`
	Pending     bool   `json:"pending"`
}

// buddyListRestModel is the minimal projection of atlas-buddies' GET
// /characters/{characterId}/buddy-list
// (services/atlas-buddies/atlas.com/buddies/list/resource.go:36). Buddies are
// a plain JSON attribute array on this resource, not a relationship.
type buddyListRestModel struct {
	Id      string                `json:"-"`
	Buddies []buddyEntryRestModel `json:"buddies"`
}

func (r buddyListRestModel) GetName() string        { return "buddy-list" }
func (r buddyListRestModel) GetID() string          { return r.Id }
func (r *buddyListRestModel) SetID(id string) error { r.Id = id; return nil }

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
