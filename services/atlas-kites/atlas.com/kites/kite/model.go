package kite

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Model is one kite (cash category 508 message box) hanging in one field.
// Immutable: construct via Builder.
type Model struct {
	id          uint32
	f           field.Model
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
	createdAt   time.Time
}

func (m Model) Id() uint32           { return m.id }
func (m Model) Field() field.Model   { return m.f }
func (m Model) CharacterId() uint32  { return m.characterId }
func (m Model) Name() string         { return m.name }
func (m Model) TemplateId() uint32   { return m.templateId }
func (m Model) Message() string      { return m.message }
func (m Model) X() int16             { return m.x }
func (m Model) Y() int16             { return m.y }
func (m Model) CreatedAt() time.Time { return m.createdAt }
