package settlement

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Item is one staged asset held by an in-flight settlement.
type Item struct {
	reservationId uuid.UUID
	inventoryType inventory.Type
	sourceSlot    slot.Position
	assetId       asset.Id
	templateId    item.Id
	quantity      asset.Quantity
}

// NewItem builds one recorded item. Item is a value type with no mutable state,
// so it needs no builder.
func NewItem(reservationId uuid.UUID, inventoryType inventory.Type, sourceSlot slot.Position, assetId asset.Id, templateId item.Id, quantity asset.Quantity) Item {
	return Item{
		reservationId: reservationId,
		inventoryType: inventoryType,
		sourceSlot:    sourceSlot,
		assetId:       assetId,
		templateId:    templateId,
		quantity:      quantity,
	}
}

func (i Item) ReservationId() uuid.UUID      { return i.reservationId }
func (i Item) InventoryType() inventory.Type { return i.inventoryType }
func (i Item) SourceSlot() slot.Position     { return i.sourceSlot }
func (i Item) AssetId() asset.Id             { return i.assetId }
func (i Item) TemplateId() item.Id           { return i.templateId }
func (i Item) Quantity() asset.Quantity      { return i.quantity }

// SideModel is one participant's contribution with its tax split frozen at
// submission time.
type SideModel struct {
	id            uuid.UUID
	position      byte
	characterId   character.Id
	characterName string
	mesoStaged    uint32
	mesoTax       uint32
	mesoDelivered uint32
	items         []Item
}

func (s SideModel) Id() uuid.UUID             { return s.id }
func (s SideModel) Position() byte            { return s.position }
func (s SideModel) CharacterId() character.Id { return s.characterId }
func (s SideModel) CharacterName() string     { return s.characterName }
func (s SideModel) MesoStaged() uint32        { return s.mesoStaged }
func (s SideModel) MesoTax() uint32           { return s.mesoTax }
func (s SideModel) MesoDelivered() uint32     { return s.mesoDelivered }

// Items returns a copy of the staged items, so a caller cannot write through
// the returned slice into the side's state.
func (s SideModel) Items() []Item {
	if s.items == nil {
		return nil
	}
	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out
}

// Model is one submitted-but-unresolved settlement.
type Model struct {
	id            uuid.UUID
	tenantId      uuid.UUID
	tenantRegion  string
	tenantMajor   uint16
	tenantMinor   uint16
	transactionId uuid.UUID
	roomId        uuid.UUID
	handle        uint32
	roomType      byte
	f             field.Model
	ownerId       character.Id
	visitorId     character.Id
	submittedAt   time.Time
	sides         []SideModel
}

func (m Model) Id() uuid.UUID           { return m.id }
func (m Model) TenantId() uuid.UUID     { return m.tenantId }
func (m Model) RoomId() uuid.UUID       { return m.roomId }
func (m Model) Handle() uint32          { return m.handle }
func (m Model) RoomType() byte          { return m.roomType }
func (m Model) Field() field.Model      { return m.f }
func (m Model) OwnerId() character.Id   { return m.ownerId }
func (m Model) VisitorId() character.Id { return m.visitorId }
func (m Model) SubmittedAt() time.Time  { return m.submittedAt }

// TransactionId is the settlement saga's transaction id. It is this record's
// identity, the ledger's idempotency key (FR-5.7), and the id reconciliation
// asks the orchestrator about.
func (m Model) TransactionId() uuid.UUID { return m.transactionId }

// Sides returns a copy of the two sides, ORDERED BY SEAT — owner (position 0)
// first. That is an order, not a role assignment: each side gives its own
// contribution and receives the other's.
func (m Model) Sides() []SideModel {
	if m.sides == nil {
		return nil
	}
	out := make([]SideModel, len(m.sides))
	copy(out, m.sides)
	return out
}

// Tenant rebuilds the tenant this settlement belongs to. Startup reconciliation
// runs with no tenant in context and must restore one per row before it can
// scope a ledger write, a REST read or a Kafka header set.
func (m Model) Tenant() (tenant.Model, error) {
	return tenant.Create(m.tenantId, m.tenantRegion, m.tenantMajor, m.tenantMinor)
}

// Make converts a persisted Entry (with its Sides and their Items preloaded)
// into an immutable Model. The error return is never non-nil today; it exists
// so Make satisfies model.Transformer.
func Make(e Entry) (Model, error) {
	b := NewBuilder(e.TransactionId, e.RoomId, e.Handle, e.RoomType,
		field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build(),
		e.OwnerId, e.VisitorId).
		SetId(e.Id).
		SetTenant(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor).
		SetSubmittedAt(e.SubmittedAt)

	for _, s := range e.Sides {
		items := make([]Item, 0, len(s.Items))
		for _, i := range s.Items {
			items = append(items, NewItem(i.ReservationId, i.InventoryType, i.SourceSlot, i.AssetId, i.TemplateId, i.Quantity))
		}
		b.addSideWithId(s.Id, s.Position, s.CharacterId, s.CharacterName, s.MesoStaged, s.MesoTax, s.MesoDelivered, items)
	}
	return b.Build(), nil
}
