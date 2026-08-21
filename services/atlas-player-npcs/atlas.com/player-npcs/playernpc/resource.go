package playernpc

import (
	"time"

	"github.com/google/uuid"
)

// EquipmentRestModel is one frozen equipment slot on the wire (PRD §5).
type EquipmentRestModel struct {
	Slot   int16  `json:"slot"`
	ItemId uint32 `json:"itemId"`
}

// RestModel is the JSON:API resource for a deployed Player NPC -- PRD §5's
// attribute list verbatim, plus overallJobRank (PRD §5 lists it; PRD §6's
// table omitted it -- design §3.1 resolves the discrepancy in PRD §5's
// favor).
type RestModel struct {
	Id             uuid.UUID            `json:"-"`
	CharacterId    uint32               `json:"characterId"`
	Name           string               `json:"name"`
	WorldId        byte                 `json:"worldId"`
	MapId          uint32               `json:"mapId"`
	ScriptId       uint32               `json:"scriptId"`
	ObjectId       uint32               `json:"objectId"`
	Gender         byte                 `json:"gender"`
	Skin           byte                 `json:"skin"`
	Face           uint32               `json:"face"`
	Hair           uint32               `json:"hair"`
	JobId          uint16               `json:"jobId"`
	X              int16                `json:"x"`
	Cy             int16                `json:"cy"`
	Fh             uint16               `json:"fh"`
	Rx0            int16                `json:"rx0"`
	Rx1            int16                `json:"rx1"`
	Dir            byte                 `json:"dir"`
	WorldRank      uint32               `json:"worldRank"`
	OverallRank    uint32               `json:"overallRank"`
	WorldJobRank   uint32               `json:"worldJobRank"`
	OverallJobRank uint32               `json:"overallJobRank"`
	Equipment      []EquipmentRestModel `json:"equipment"`
	DeployedAt     time.Time            `json:"deployedAt"`
}

func (r RestModel) GetName() string {
	return "player-npcs"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID tolerates an empty id -- api2go calls it unconditionally while
// unmarshalling, and none of the read paths that produce a RestModel ever
// need to parse one back.
func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = uuid.Nil
		return nil
	}
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

// Transform converts a domain Model to its wire representation.
// DeployedAt is the row's CreatedAt -- a re-deploy (Redeploy) never
// recreates the row, so this stays the original deploy time (design §6.2).
func Transform(m Model) (RestModel, error) {
	equipment := make([]EquipmentRestModel, 0, len(m.Equipment()))
	for _, e := range m.Equipment() {
		equipment = append(equipment, EquipmentRestModel{Slot: e.Slot(), ItemId: e.ItemId()})
	}
	return RestModel{
		Id:             m.Id(),
		CharacterId:    m.CharacterId(),
		Name:           m.Name(),
		WorldId:        m.WorldId(),
		MapId:          m.MapId(),
		ScriptId:       m.ScriptId(),
		ObjectId:       m.ObjectId(),
		Gender:         m.Gender(),
		Skin:           m.Skin(),
		Face:           m.Face(),
		Hair:           m.Hair(),
		JobId:          m.JobId(),
		X:              m.X(),
		Cy:             m.Cy(),
		Fh:             m.Fh(),
		Rx0:            m.RX0(),
		Rx1:            m.RX1(),
		Dir:            m.Dir(),
		WorldRank:      m.WorldRank(),
		OverallRank:    m.OverallRank(),
		WorldJobRank:   m.WorldJobRank(),
		OverallJobRank: m.OverallJobRank(),
		Equipment:      equipment,
		DeployedAt:     m.CreatedAt(),
	}, nil
}
