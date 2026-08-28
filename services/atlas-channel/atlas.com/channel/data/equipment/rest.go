package equipment

import "strconv"

type RestModel struct {
	Id           uint32   `json:"-"`
	PetAbilities []string `json:"petAbilities,omitempty"`
	// NotExtend is info/notExtend — when set, an item-expiration extender
	// (Magical Sandglass) may not be applied to this equip. The client
	// enforces it via CItemInfo::IsNotExtendItem; the server re-checks so a
	// crafted request cannot bypass it.
	NotExtend bool `json:"notExtend"`
}

func (r RestModel) GetName() string {
	return "statistics"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's unmarshal
// even though this client doesn't care about the equipment resource's
// relationships (see libs/atlas-rest gotcha): a target struct must implement
// them or unmarshal errors whenever the upstream response includes a
// relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Transform(m Model) (RestModel, error) {
	petAbilities := make([]string, len(m.petAbilities))
	copy(petAbilities, m.petAbilities)

	return RestModel{
		Id:           m.id,
		PetAbilities: petAbilities,
		NotExtend:    m.notExtend,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		petAbilities: rm.PetAbilities,
		notExtend:    rm.NotExtend,
	}, nil
}
