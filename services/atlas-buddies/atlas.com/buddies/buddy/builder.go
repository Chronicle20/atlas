package buddy

import (
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	listId        uuid.UUID
	characterId   uint32
	group         string
	characterName string
	channelId     int8
	inShop        bool
	pending       bool
}

func NewBuilder(listId uuid.UUID, characterId uint32) *Builder {
	return &Builder{
		listId:      listId,
		characterId: characterId,
		group:       "Default Group",
		channelId:   -1,
		inShop:      false,
		pending:     false,
	}
}

func (b *Builder) SetGroup(group string) *Builder {
	b.group = group
	return b
}

func (b *Builder) SetCharacterName(name string) *Builder {
	b.characterName = name
	return b
}

func (b *Builder) SetChannelId(channelId int8) *Builder {
	b.channelId = channelId
	return b
}

func (b *Builder) SetInShop(inShop bool) *Builder {
	b.inShop = inShop
	return b
}

func (b *Builder) SetPending(pending bool) *Builder {
	b.pending = pending
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.listId == uuid.Nil {
		return Model{}, errors.New("listId is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}

	return Model{
		listId:        b.listId,
		characterId:   b.characterId,
		group:         b.group,
		characterName: b.characterName,
		channelId:     b.channelId,
		inShop:        b.inShop,
		pending:       b.pending,
	}, nil
}
