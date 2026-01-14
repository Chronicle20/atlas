package note

import (
	"errors"
	"time"
)

var (
	ErrInvalidId = errors.New("note id must be greater than 0")
)

type modelBuilder struct {
	id          uint32
	characterId uint32
	senderId    uint32
	message     string
	timestamp   time.Time
	flag        byte
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		timestamp: time.Now(),
	}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder() *modelBuilder {
	return NewModelBuilder()
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:          m.id,
		characterId: m.characterId,
		senderId:    m.senderId,
		message:     m.message,
		timestamp:   m.timestamp,
		flag:        m.flag,
	}
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetSenderId(senderId uint32) *modelBuilder {
	b.senderId = senderId
	return b
}

func (b *modelBuilder) SetMessage(message string) *modelBuilder {
	b.message = message
	return b
}

func (b *modelBuilder) SetTimestamp(timestamp time.Time) *modelBuilder {
	b.timestamp = timestamp
	return b
}

func (b *modelBuilder) SetFlag(flag byte) *modelBuilder {
	b.flag = flag
	return b
}

func (b *modelBuilder) Build() (Model, error) {
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

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
