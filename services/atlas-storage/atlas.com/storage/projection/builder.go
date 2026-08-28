package projection

import (
	"atlas-storage/asset"
	"errors"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder for constructing Model instances
type Builder struct {
	characterId  uint32
	accountId    uint32
	worldId      world.Id
	storageId    uuid.UUID
	capacity     uint32
	mesos        uint32
	npcId        uint32
	compartments map[inventory.Type][]asset.Model
}

func NewBuilder() *Builder {
	return &Builder{
		compartments: make(map[inventory.Type][]asset.Model),
	}
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetAccountId(accountId uint32) *Builder {
	b.accountId = accountId
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetStorageId(storageId uuid.UUID) *Builder {
	b.storageId = storageId
	return b
}

func (b *Builder) SetCapacity(capacity uint32) *Builder {
	b.capacity = capacity
	return b
}

func (b *Builder) SetMesos(mesos uint32) *Builder {
	b.mesos = mesos
	return b
}

func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

func (b *Builder) SetCompartments(compartments map[inventory.Type][]asset.Model) *Builder {
	b.compartments = compartments
	return b
}

func (b *Builder) validate() error {
	if b.characterId == 0 {
		return errors.New("character id is required")
	}
	if b.accountId == 0 {
		return errors.New("account id is required")
	}
	if b.storageId == uuid.Nil {
		return errors.New("storage id is required")
	}
	return nil
}

func (b *Builder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		characterId:  b.characterId,
		accountId:    b.accountId,
		worldId:      b.worldId,
		storageId:    b.storageId,
		capacity:     b.capacity,
		mesos:        b.mesos,
		npcId:        b.npcId,
		compartments: b.compartments,
	}, nil
}

func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// Clone creates a builder from an existing model for modifications
func Clone(m Model) *Builder {
	// Deep copy compartments
	compartments := make(map[inventory.Type][]asset.Model)
	for k, v := range m.compartments {
		copied := make([]asset.Model, len(v))
		copy(copied, v)
		compartments[k] = copied
	}

	return &Builder{
		characterId:  m.characterId,
		accountId:    m.accountId,
		worldId:      m.worldId,
		storageId:    m.storageId,
		capacity:     m.capacity,
		mesos:        m.mesos,
		npcId:        m.npcId,
		compartments: compartments,
	}
}
