package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// RestModel mirrors the subset of atlas-character's wire format the trade
// service needs (id, name, hp, level, meso).
type RestModel struct {
	Id    character.Id `json:"-"`
	Name  string       `json:"name"`
	Hp    uint16       `json:"hp"`
	Level byte         `json:"level"`
	Meso  uint32       `json:"meso"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.Id = character.Id(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are defensive no-op stubs so a
// future relationships block on atlas-character's resource cannot break the
// decode (task-037 failure class, see libs/atlas-rest/CLAUDE.md). The trade
// service reads only id/name/hp/level/meso.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:    rm.Id,
		name:  rm.Name,
		hp:    rm.Hp,
		level: rm.Level,
		meso:  rm.Meso,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:    m.id,
		Name:  m.name,
		Hp:    m.hp,
		Level: m.level,
		Meso:  m.meso,
	}, nil
}
