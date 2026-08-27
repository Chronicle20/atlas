package note

import (
	"errors"
	"time"
)

var ErrInvalidId = errors.New("note id must be greater than 0")

type builder struct {
	id          uint32
	characterId uint32
	senderId    uint32
	message     string
	timestamp   time.Time
	flag        byte
}

// NewBuilder creates a new builder instance
func NewBuilder() *builder {
	return &builder{
		timestamp: time.Now(),
	}
}

func CloneModel(m Model) *builder {
	return &builder{
		id:          m.id,
		characterId: m.characterId,
		senderId:    m.senderId,
		message:     m.message,
		timestamp:   m.timestamp,
		flag:        m.flag,
	}
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetSenderId(senderId uint32) *builder {
	b.senderId = senderId
	return b
}

func (b *builder) SetMessage(message string) *builder {
	b.message = message
	return b
}

func (b *builder) SetTimestamp(timestamp time.Time) *builder {
	b.timestamp = timestamp
	return b
}

func (b *builder) SetFlag(flag byte) *builder {
	b.flag = flag
	return b
}

func (b *builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:          b.id,
		characterId: b.characterId,
		senderId:    b.senderId,
		message:     b.message,
		timestamp:   b.timestamp,
		flag:        b.flag,
	}, nil
}

func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
