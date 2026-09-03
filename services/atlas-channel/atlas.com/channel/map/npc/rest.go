package npc

import "strconv"

// RestModel mirrors atlas-maps' map/npc.RestModel field-for-field
// (services/atlas-maps/atlas.com/maps/map/npc/rest.go). Id carries the
// scripted NPC's uniqueId as a string -- atlas-maps' own RestModel.Id is a
// string, not a uuid.UUID.
type RestModel struct {
	Id    string `json:"-"`
	NpcId uint32 `json:"npcId"`
	X     int16  `json:"x"`
	Y     int16  `json:"y"`
	Fh    int16  `json:"fh"`
}

func (r RestModel) GetName() string {
	return "npcs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by
// api2go/jsonapi's UnmarshalToOneRelations/UnmarshalToManyRelations
// interfaces even though this resource has no relationships of its own --
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
	uniqueId, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}
	return Model{
		uniqueId: uint32(uniqueId),
		npcId:    rm.NpcId,
		x:        rm.X,
		y:        rm.Y,
		fh:       rm.Fh,
	}, nil
}
