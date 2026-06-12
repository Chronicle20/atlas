package petdata

// Model represents pet evolution data sourced from atlas-data
type Model struct {
	id          uint32
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  int
}

// NewModel creates a new pet evolution data model.
func NewModel(id uint32, reqPetLevel uint32, reqItemId uint32, evolutions int) Model {
	return Model{
		id:          id,
		reqPetLevel: reqPetLevel,
		reqItemId:   reqItemId,
		evolutions:  evolutions,
	}
}

// Id returns the pet's template identifier
func (m Model) Id() uint32 { return m.id }

// ReqPetLevel returns the pet level required to evolve
func (m Model) ReqPetLevel() uint32 { return m.reqPetLevel }

// ReqItemId returns the item id required to evolve
func (m Model) ReqItemId() uint32 { return m.reqItemId }

// IsEvolvable reports an NPC-evolvable pet (gated by a required item).
func (m Model) IsEvolvable() bool { return m.evolutions > 0 && m.reqItemId != 0 }
