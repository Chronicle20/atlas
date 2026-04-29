package character

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RestModel is the minimal projection of the atlas-character JSON:API
// resource needed by atlas-maps. atlas-character exposes many more
// attributes; only the fields we consume (position + map id) are
// declared here.
type RestModel struct {
	Id    uint32  `json:"-"`
	MapId _map.Id `json:"mapId"`
	X     int16   `json:"x"`
	Y     int16   `json:"y"`
}

// GetName returns the JSON:API resource type. Must match atlas-character.
func (r RestModel) GetName() string {
	return "characters"
}

// GetID returns the JSON:API resource id.
func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID parses the JSON:API resource id back into the model.
func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToManyReferenceIDs is a no-op required by api2go's interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
