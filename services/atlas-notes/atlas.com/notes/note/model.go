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
	// giftNote is server-only: it records that this note originated from a
	// cash-shop gift acknowledgement, whose fame was already settled at
	// acceptance time. It is never rendered to the client.
	giftNote bool
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

// GiftNote returns whether this note originated from a cash-shop gift
// acknowledgement; its fame was settled at acceptance time.
func (n Model) GiftNote() bool {
	return n.giftNote
}
