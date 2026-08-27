package consumable

import "strconv"

type SpecType string

const (
	SpecTypeHP         = SpecType("hp")
	SpecTypeMP         = SpecType("mp")
	SpecTypeHPRecovery = SpecType("hpR")
	SpecTypeMPRecovery = SpecType("mpR")
)

type RestModel struct {
	Id   uint32             `json:"-"`
	Spec map[SpecType]int32 `json:"spec"`
	// Npc is the NPC template a scripted item's dialogue renders with (the
	// 243xxxx family, WZ spec/npc) or the NPC a remote-NPC item summons (the
	// 239xxxx family, WZ info/npc). Tags must match atlas-data's
	// consumable/rest.go exactly — a mismatch decodes to zero silently and is
	// indistinguishable from a content gap.
	Npc uint32 `json:"npc"`
	// Script is the WZ spec/script value. Recorded for authoring traceability
	// only; conversations are keyed by item id, never by script name.
	Script      string `json:"script"`
	RunOnPickup bool   `json:"runOnPickup"`
}

func (r RestModel) GetName() string {
	return "consumables"
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
// even though the consumable resource carries no relationships this client
// cares about (see libs/atlas-rest gotcha): a target struct must implement them
// or unmarshal errors whenever the upstream response includes a relationships
// block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Transform(m Model) (RestModel, error) {
	spec := make(map[SpecType]int32, len(m.spec))
	for k, v := range m.spec {
		spec[k] = v
	}

	return RestModel{
		Id:          m.id,
		Spec:        spec,
		Npc:         m.npc,
		Script:      m.script,
		RunOnPickup: m.runOnPickup,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		spec:        rm.Spec,
		npc:         rm.Npc,
		script:      rm.Script,
		runOnPickup: rm.RunOnPickup,
	}, nil
}
