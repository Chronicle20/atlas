package pending_change

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the read-side JSON:API projection of a pending change record.
// AssetId is deliberately absent: it is an internal detail of the refund
// mechanism (design §5.4), not something an operator-facing GET needs to
// expose.
type RestModel struct {
	Id                 string     `json:"-"`
	CharacterId        uint32     `json:"characterId"`
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	RequestedName      string     `json:"requestedName,omitempty"`
	DestinationWorldId world.Id   `json:"destinationWorldId"`
	SourceWorldId      world.Id   `json:"sourceWorldId"`
	Reason             string     `json:"reason,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	ResolvedAt         *time.Time `json:"resolvedAt,omitempty"`
}

func (r RestModel) GetName() string {
	return "pending-changes"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                 m.Id().String(),
		CharacterId:        m.CharacterId(),
		Type:               m.Type(),
		Status:             m.Status(),
		RequestedName:      m.RequestedName(),
		DestinationWorldId: m.DestinationWorldId(),
		SourceWorldId:      m.SourceWorldId(),
		Reason:             m.Reason(),
		CreatedAt:          m.CreatedAt(),
		ExpiresAt:          m.ExpiresAt(),
		ResolvedAt:         m.ResolvedAt(),
	}, nil
}

// CreateInputRestModel is the POST body:
// {data:{type:"pending-changes",attributes:{type,requestedName,destinationWorldId,assetId}}}.
//
// AssetId carries an item TEMPLATE id, not an instance id: it travels
// verbatim onto sharedsaga.DestroyAssetPayload.TemplateId at every call site
// in this package's producer (Task 6's award/destroy providers), and every
// other caller of that saga contract also treats the field as a template id.
// atlas-channel (Task 24) is the other side of this seam and must agree on
// that semantics.
type CreateInputRestModel struct {
	Id                 string   `json:"-"`
	Type               string   `json:"type"`
	RequestedName      string   `json:"requestedName,omitempty"`
	DestinationWorldId world.Id `json:"destinationWorldId,omitempty"`
	AssetId            *uint32  `json:"assetId,omitempty"`
}

func (r CreateInputRestModel) GetName() string {
	return "pending-changes"
}

func (r CreateInputRestModel) GetID() string {
	return r.Id
}

func (r *CreateInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// CancelInputRestModel is the POST body of the self-scoped cancel route:
// {data:{type:"pending-changes",attributes:{type}}}. Unlike DELETE
// .../pending-changes/{id}, the caller carries no record id -- the wire
// packet that drives this route (task-227 client-cancel addendum) has none,
// so the record is looked up by (characterId, type) instead. See
// handleCancelPendingChangeForCharacter.
type CancelInputRestModel struct {
	Id   string `json:"-"`
	Type string `json:"type"`
}

func (r CancelInputRestModel) GetName() string {
	return "pending-changes"
}

func (r CancelInputRestModel) GetID() string {
	return r.Id
}

func (r *CancelInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// EligibilityRestModel is the read-only response of GET
// .../transfer-eligibility. Id is the characterId the check was run for, so
// the response is addressable JSON:API, but the caller (atlas-channel) only
// ever reads Eligible/Reason.
type EligibilityRestModel struct {
	Id       string `json:"-"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

func (r EligibilityRestModel) GetName() string {
	return "transfer-eligibilities"
}

func (r EligibilityRestModel) GetID() string {
	return r.Id
}

func (r *EligibilityRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ResolveInputRestModel is the POST body of the generic (non-cancel) resolve
// route:
// {data:{type:"pending-changes",attributes:{status,reason}}}. It is the
// world-transfer saga's (task-227 Task 13/14) way to move a record to
// APPLIED on saga success or REJECTED on saga failure — the terminal event
// the record's own package comment describes as "what drives Resolve"
// (pending_change/processor.go's startWorldTransfer). Status is restricted
// to StatusApplied/StatusRejected at the handler: StatusCancelled already has
// its own dedicated DELETE route, and StatusExpired is only ever reached by
// the sweep ticker.
type ResolveInputRestModel struct {
	Id     string `json:"-"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (r ResolveInputRestModel) GetName() string {
	return "pending-changes"
}

func (r ResolveInputRestModel) GetID() string {
	return r.Id
}

func (r *ResolveInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// --- Remote-service projections used by requests.go's gate clients and
// task.go's tenant fetcher. Each is the minimal deserialization target for
// one cross-service GET; see requests.go/task.go for the request functions
// and task-11-report.md for the file:line citation of each owning service's
// resource.go.

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

// characterSlotRestModel is the minimal projection of atlas-account's GET
// /accounts/{accountId}/worlds/{worldId}/character-slots
// (services/atlas-account/atlas.com/account/account/rest.go,
// CharacterSlotRestModel), the world-scoped sub-resource that replaced the
// flat, always-4 `characterSlots` attribute the old accounts/{accountId}
// response carried (task-246 bug-b-type-must-add-a-slot.md). Mirrors
// atlas-login's and atlas-channel's CharacterSlotRestModel byte for byte.
type characterSlotRestModel struct {
	Id      string   `json:"-"`
	WorldId world.Id `json:"worldId"`
	Slots   int16    `json:"slots"`
}

func (r characterSlotRestModel) GetName() string        { return "character-slots" }
func (r characterSlotRestModel) GetID() string          { return r.Id }
func (r *characterSlotRestModel) SetID(id string) error { r.Id = id; return nil }

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

// parcelStatusRestModel is the minimal projection of atlas-parcel's GET
// /characters/{characterId}/parcel-status
// (services/atlas-parcel/atlas.com/parcel/parcel/resource.go:55; rest.go:120,
// 123, 138 — resource type "parcel-statuses", attribute InFlight with tag
// `json:"inFlight"`, id the characterId as a decimal string). No
// relationships block.
type parcelStatusRestModel struct {
	Id       string `json:"-"`
	InFlight bool   `json:"inFlight"`
}

func (r parcelStatusRestModel) GetName() string        { return "parcel-statuses" }
func (r parcelStatusRestModel) GetID() string          { return r.Id }
func (r *parcelStatusRestModel) SetID(id string) error { r.Id = id; return nil }

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

// familyMemberRestModel is the minimal projection of atlas-families' GET
// /families/tree/{characterId}
// (services/atlas-families/atlas.com/family/family/resource.go:28). A 404
// (ErrMemberNotFound) means the character has no family member row at all;
// any other success means it does, even a solo tree of just itself. No
// relationships block.
type familyMemberRestModel struct {
	Id string `json:"-"`
}

func (r familyMemberRestModel) GetName() string                                   { return "family-tree-members" }
func (r familyMemberRestModel) GetID() string                                     { return r.Id }
func (r *familyMemberRestModel) SetID(id string) error                            { r.Id = id; return nil }
func (r *familyMemberRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *familyMemberRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

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

// tenantRestModel is the minimal JSON:API shape needed to unmarshal
// GET /tenants/{tenantId} from atlas-tenants -- just region and version, the
// two fields tenant.Create needs beyond the id already known from the query
// above (task.go's expiredTenantIds). Downstream Kafka consumers of the
// refund/notification events Processor.Sweep emits rely on the real tenant
// region/version reaching them via the envelope headers, so a placeholder
// tenant.Model here would corrupt those headers for every other service
// reading the topic.
type tenantRestModel struct {
	Id           string `json:"-"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

func (r tenantRestModel) GetName() string { return "tenants" }

func (r tenantRestModel) GetID() string { return r.Id }

func (r *tenantRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
