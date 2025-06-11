package note

import (
	"time"
)

// Model represents a note for a character
type Model struct {
	id          uint32
	characterId uint32
	senderId    uint32
	message     string
	timestamp   time.Time
	flag        byte
}

// Id returns the note's ID
func (n Model) Id() uint32 {
	return n.id
}

// CharacterId returns the ID of the character the note belongs to
func (n Model) CharacterId() uint32 {
	return n.characterId
}

// SenderId returns the ID of the character who sent the note
func (n Model) SenderId() uint32 {
	return n.senderId
}

// Message returns the note's message
func (n Model) Message() string {
	return n.message
}

// Timestamp returns when the note was created
func (n Model) Timestamp() time.Time {
	return n.timestamp
}

// Flag returns the note's flag
func (n Model) Flag() byte {
	return n.flag
}

// Builder is a builder for creating Model instances
type Builder struct {
	id          uint32
	characterId uint32
	senderId    uint32
	message     string
	timestamp   time.Time
	flag        byte
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		timestamp: time.Now(),
	}
}

// SetId sets the note's ID
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetCharacterId sets the ID of the character the note belongs to
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetSenderId sets the ID of the character who sent the note
func (b *Builder) SetSenderId(senderId uint32) *Builder {
	b.senderId = senderId
	return b
}

// SetMessage sets the note's message
func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

// SetTimestamp sets when the note was created
func (b *Builder) SetTimestamp(timestamp time.Time) *Builder {
	b.timestamp = timestamp
	return b
}

// SetFlag sets the note's flag
func (b *Builder) SetFlag(flag byte) *Builder {
	b.flag = flag
	return b
}

// Build creates a new Model with the builder's values
func (b *Builder) Build() Model {
	return Model{
		id:          b.id,
		characterId: b.characterId,
		senderId:    b.senderId,
		message:     b.message,
		timestamp:   b.timestamp,
		flag:        b.flag,
	}
}
