package mobskill

import (
	"fmt"
	"strconv"
)

type RestModel struct {
	SkillId     uint16  `json:"-"`
	Level       uint16  `json:"-"`
	MpCon       uint32  `json:"mp_con"`
	Duration    uint32  `json:"duration"`
	Hp          uint32  `json:"hp"`
	X           int32   `json:"x"`
	Y           int32   `json:"y"`
	Prop        uint32  `json:"prop"`
	Interval    uint32  `json:"interval"`
	Count       uint32  `json:"count"`
	Limit       uint32  `json:"limit"`
	LtX         int32   `json:"lt_x"`
	LtY         int32   `json:"lt_y"`
	RbX         int32   `json:"rb_x"`
	RbY         int32   `json:"rb_y"`
	SummonEffect uint32 `json:"summon_effect"`
	Summons     []uint32 `json:"summons"`
}

func (r RestModel) GetName() string {
	return "mob-skills"
}

// CompositeId encodes a (skillId, level) pair into a single uint32 for document storage.
func CompositeId(skillId uint16, level uint16) string {
	return strconv.Itoa(int(skillId)*10000 + int(level))
}

func (r RestModel) GetID() string {
	return CompositeId(r.SkillId, r.Level)
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return fmt.Errorf("invalid mob skill id: %s", strId)
	}
	r.SkillId = uint16(id / 10000)
	r.Level = uint16(id % 10000)
	return nil
}
