// Package character is the atlas-character REST client atlas-trades validates
// through. It reads only what the trade rules need: hp (FR-4.7 alive check),
// level (FR-4.5 minimum trade level), name (the room participant label the
// client renders) and meso (FR-4.8 meso staging bound).
package character

import "github.com/Chronicle20/atlas/libs/atlas-constants/character"

// Model is the minimal character view the trade validation ladder needs.
type Model struct {
	id    character.Id
	name  string
	hp    uint16
	level byte
	meso  uint32
}

func (m Model) Id() character.Id { return m.id }

func (m Model) Name() string { return m.name }

func (m Model) Hp() uint16 { return m.hp }

// Level is a byte because atlas-character's wire field is a byte
// (services/atlas-character/atlas.com/character/character/rest.go:20); widening
// it here would misrepresent the boundary.
func (m Model) Level() byte { return m.level }

func (m Model) Meso() uint32 { return m.meso }
