package playernpc

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// TransformSlice converts a slice of Model into their wire representation
// in parallel -- the list endpoint's (resource.go's handleGetPlayerNpcs)
// shared entry point, so no handler needs its own inline model.SliceMap
// call.
func TransformSlice(ms []Model) ([]RestModel, error) {
	return model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
}

// PositionRestModel is PRD §5's optional deploy `position: {x, y}` -- the
// GM path that bypasses the positioner entirely (processor.go's Position,
// Deploy's `explicit` parameter). It has exactly Position's two fields.
type PositionRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// DeployRestModel is the POST /player-npcs body (PRD §5). Position is
// optional; when absent, Deploy resolves it itself.
type DeployRestModel struct {
	Id          uuid.UUID          `json:"-"`
	CharacterId uint32             `json:"characterId"`
	WorldId     byte               `json:"worldId"`
	MapId       uint32             `json:"mapId"`
	Position    *PositionRestModel `json:"position,omitempty"`
}

func (r DeployRestModel) GetName() string {
	return "player-npcs"
}

func (r DeployRestModel) GetID() string {
	return r.Id.String()
}

// SetID tolerates an empty id -- a deploy request supplies no id of its
// own; the created resource's id is minted by the insert.
func (r *DeployRestModel) SetID(strId string) error {
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

// RedeployRestModel is the (structurally empty) PATCH /player-npcs/{id}
// body. Redeploy takes no attributes -- script id, object id and position
// are immutable through this endpoint (design §6.2) -- but the route still
// registers through the input-handler variant (DOM-08), so a caller sends
// a bare JSON:API document (`{"data":{"type":"player-npcs"}}`) and every
// field here is ignored.
type RedeployRestModel struct {
	Id uuid.UUID `json:"-"`
}

func (r RedeployRestModel) GetName() string {
	return "player-npcs"
}

func (r RedeployRestModel) GetID() string {
	return r.Id.String()
}

// SetID tolerates an empty id -- the id that matters comes from the URL
// path, not the body.
func (r *RedeployRestModel) SetID(strId string) error {
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
