package playernpc

import "github.com/google/uuid"

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
