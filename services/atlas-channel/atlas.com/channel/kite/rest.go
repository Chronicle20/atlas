package kite

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel mirrors atlas-kites' kite.RestModel (Task 10) field-for-field.
// Id holds the kite wire id -- NOT the character id.
type RestModel struct {
	Id          uint32     `json:"-"`
	CharacterId uint32     `json:"characterId"`
	Name        string     `json:"name"`
	TemplateId  uint32     `json:"templateId"`
	Message     string     `json:"message"`
	X           int16      `json:"x"`
	Y           int16      `json:"y"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	InstanceId  uuid.UUID  `json:"instanceId"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "kites"
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

// SetToOneReferenceID and SetToManyReferenceIDs are required by
// api2go/jsonapi's UnmarshalToOneRelations/UnmarshalToManyRelations
// interfaces even though kites has no relationships of its own -- without
// them, any response body carrying a `relationships` block fails decode
// with "does not implement UnmarshalToManyRelations" instead of a clean
// fetch error (libs/atlas-rest/CLAUDE.md; the task-037 bug class).
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		characterId: rm.CharacterId,
		name:        rm.Name,
		templateId:  rm.TemplateId,
		message:     rm.Message,
		x:           rm.X,
		y:           rm.Y,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.id,
		CharacterId: m.characterId,
		Name:        m.name,
		TemplateId:  m.templateId,
		Message:     m.message,
		X:           m.x,
		Y:           m.y,
	}, nil
}
