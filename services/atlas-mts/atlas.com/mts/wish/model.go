package wish

import (
	"time"

	"github.com/google/uuid"
)

// Model is the immutable wish-list entry: a character's standing wish for an
// item template. "Criteria" is just the item template id for now. Construct it
// via the Builder.
type Model struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	characterId uint32
	itemId      uint32
	createdAt   time.Time
}

func (m Model) Id() uuid.UUID        { return m.id }
func (m Model) TenantId() uuid.UUID  { return m.tenantId }
func (m Model) CharacterId() uint32  { return m.characterId }
func (m Model) ItemId() uint32       { return m.itemId }
func (m Model) CreatedAt() time.Time { return m.createdAt }
