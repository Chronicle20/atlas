package playernpc

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EquipmentRestModel is one frozen equipment slot on the wire, mirroring
// atlas-player-npcs' playernpc.EquipmentRestModel (Task 16) field-for-field.
type EquipmentRestModel struct {
	Slot   int16  `json:"slot"`
	ItemId uint32 `json:"itemId"`
}

// RestModel mirrors atlas-player-npcs' playernpc.RestModel (Task 16)
// field-for-field, including overallJobRank -- PRD §5 lists it; PRD §6's
// table omitted it, and design §3.1 resolves the discrepancy in PRD §5's
// favor (services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource.go).
type RestModel struct {
	Id             uuid.UUID            `json:"-"`
	CharacterId    uint32               `json:"characterId"`
	Name           string               `json:"name"`
	WorldId        world.Id             `json:"worldId"`
	MapId          _map.Id              `json:"mapId"`
	ScriptId       uint32               `json:"scriptId"`
	ObjectId       uint32               `json:"objectId"`
	Gender         byte                 `json:"gender"`
	Skin           byte                 `json:"skin"`
	Face           uint32               `json:"face"`
	Hair           uint32               `json:"hair"`
	JobId          job.Id               `json:"jobId"`
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
// unmarshalling, matching atlas-player-npcs' own RestModel.SetID.
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

// SetToOneReferenceID and SetToManyReferenceIDs are required by
// api2go/jsonapi's UnmarshalToOneRelations/UnmarshalToManyRelations
// interfaces even though player-npcs has no relationships of its own --
// without them, any response body carrying a `relationships` block fails
// decode with "does not implement UnmarshalToManyRelations" instead of a
// clean fetch error (libs/atlas-rest/CLAUDE.md; the task-037 bug class).
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Extract converts a wire RestModel into the domain Model.
func Extract(rm RestModel) (Model, error) {
	equipment := make([]EquipmentModel, 0, len(rm.Equipment))
	for _, e := range rm.Equipment {
		equipment = append(equipment, EquipmentModel{slot: e.Slot, itemId: e.ItemId})
	}
	return Model{
		id:             rm.Id,
		characterId:    rm.CharacterId,
		name:           rm.Name,
		worldId:        rm.WorldId,
		mapId:          rm.MapId,
		scriptId:       rm.ScriptId,
		objectId:       rm.ObjectId,
		gender:         rm.Gender,
		skin:           rm.Skin,
		face:           rm.Face,
		hair:           rm.Hair,
		jobId:          rm.JobId,
		x:              rm.X,
		cy:             rm.Cy,
		fh:             rm.Fh,
		rx0:            rm.Rx0,
		rx1:            rm.Rx1,
		dir:            rm.Dir,
		worldRank:      rm.WorldRank,
		overallRank:    rm.OverallRank,
		worldJobRank:   rm.WorldJobRank,
		overallJobRank: rm.OverallJobRank,
		equipment:      equipment,
		deployedAt:     rm.DeployedAt,
	}, nil
}
