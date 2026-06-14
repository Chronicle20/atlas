package summon

import "strconv"

// RestModel mirrors the atlas-summons JSON:API `summons` resource (the subset the
// channel needs to replay an existing summon to an entering player).
type RestModel struct {
	Id               string `json:"-"`
	OwnerCharacterId uint32 `json:"ownerCharacterId"`
	SkillId          uint32 `json:"skillId"`
	SkillLevel       byte   `json:"skillLevel"`
	SummonType       string `json:"summonType"`
	MovementType     byte   `json:"movementType"`
	X                int16  `json:"x"`
	Y                int16  `json:"y"`
}

func (r RestModel) GetID() string          { return r.Id }
func (r *RestModel) SetID(id string) error { r.Id = id; return nil }
func (r RestModel) GetName() string        { return "summons" }

// Model is the channel-side view of an existing summon, used to build a
// SummonSpawn packet for a character entering the map.
type Model struct {
	id               uint32
	ownerCharacterId uint32
	skillId          uint32
	skillLevel       byte
	summonType       string
	movementType     byte
	x                int16
	y                int16
}

func (m Model) Id() uint32               { return m.id }
func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) SkillId() uint32          { return m.skillId }
func (m Model) SkillLevel() byte         { return m.skillLevel }
func (m Model) MovementType() byte       { return m.movementType }
func (m Model) X() int16                 { return m.x }
func (m Model) Y() int16                 { return m.y }
func (m Model) IsPuppet() bool           { return m.summonType == "PUPPET" }

func Extract(r RestModel) (Model, error) {
	id, err := strconv.ParseUint(r.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:               uint32(id),
		ownerCharacterId: r.OwnerCharacterId,
		skillId:          r.SkillId,
		skillLevel:       r.SkillLevel,
		summonType:       r.SummonType,
		movementType:     r.MovementType,
		x:                r.X,
		y:                r.Y,
	}, nil
}
