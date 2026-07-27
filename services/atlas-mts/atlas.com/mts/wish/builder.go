package wish

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder constructs an immutable wish Model. The id and serial are assigned at
// create time in the administrator, so they are not required here.
type Builder struct {
	id            uuid.UUID
	tenantId      uuid.UUID
	worldId       world.Id
	serial        uint32
	characterId   uint32
	itemId        uint32
	listingSerial uint32
	wishType      string
	price         uint32
	count         uint32
	expiresAt     *time.Time
	createdAt     time.Time
}

func NewBuilder(tenantId uuid.UUID, characterId uint32, itemId uint32) *Builder {
	return &Builder{tenantId: tenantId, characterId: characterId, itemId: itemId, wishType: TypeCart}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetSerial(serial uint32) *Builder {
	b.serial = serial
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

// SetListingSerial records the favorited listing's ITC serial for a "cart"
// entry (0 for "wanted" entries, which reference no listing).
func (b *Builder) SetListingSerial(v uint32) *Builder {
	b.listingSerial = v
	return b
}

func (b *Builder) SetType(t string) *Builder {
	if t != "" {
		b.wishType = t
	}
	return b
}

func (b *Builder) SetPrice(v uint32) *Builder {
	b.price = v
	return b
}

func (b *Builder) SetCount(v uint32) *Builder {
	b.count = v
	return b
}

func (b *Builder) SetExpiresAt(v *time.Time) *Builder {
	b.expiresAt = v
	return b
}

func (b *Builder) SetCreatedAt(v time.Time) *Builder {
	b.createdAt = v
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		id:            b.id,
		tenantId:      b.tenantId,
		worldId:       b.worldId,
		serial:        b.serial,
		characterId:   b.characterId,
		itemId:        b.itemId,
		listingSerial: b.listingSerial,
		wishType:      b.wishType,
		price:         b.price,
		count:         b.count,
		expiresAt:     b.expiresAt,
		createdAt:     b.createdAt,
	}, nil
}
