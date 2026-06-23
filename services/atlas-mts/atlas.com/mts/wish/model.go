package wish

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Model is the immutable wish-list entry: a character's standing wish for an
// item template. "Criteria" is just the item template id for now. Construct it
// via the Builder.
//
// Serial is the per-(tenant, world) ITC serial (the client's nITCSN) assigned to
// the wish entry at create time, drawn from the SAME shared counter as listings
// and holdings (the `serial` package) so a serial maps to exactly one row across
// all three tables within a world. The wish view (LoadWishSaleListDone) renders
// each entry's Serial into the ITCITEM's itcSn field; the client echoes it back
// verbatim on CANCEL_WISH (IDA: CITC::OnCancelWish Encode4 of the item's nITCSN),
// so the channel resolves a CANCEL_WISH serial straight back to the wish entry.
type Model struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	worldId     world.Id
	serial      uint32
	characterId uint32
	itemId      uint32
	createdAt   time.Time
}

func (m Model) Id() uuid.UUID        { return m.id }
func (m Model) TenantId() uuid.UUID  { return m.tenantId }
func (m Model) WorldId() world.Id    { return m.worldId }
func (m Model) Serial() uint32       { return m.serial }
func (m Model) CharacterId() uint32  { return m.characterId }
func (m Model) ItemId() uint32       { return m.itemId }
func (m Model) CreatedAt() time.Time { return m.createdAt }
