package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// RestModel mirrors the subset of atlas-character's "characters" resource
// that a deploy snapshot needs (design.md §6.1): appearance and identity.
// Equipment is not served by atlas-character — it is read separately from
// atlas-inventory (see the inventory/ package).
type RestModel struct {
	Id        uint32 `json:"-"`
	Name      string `json:"name"`
	Gender    byte   `json:"gender"`
	SkinColor byte   `json:"skinColor"`
	Face      uint32 `json:"face"`
	Hair      uint32 `json:"hair"`
	JobId     job.Id `json:"jobId"`
	Level     byte   `json:"level"`
	Gm        int    `json:"gm"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required even though
// this client no longer requests any relationship — see
// libs/atlas-rest/CLAUDE.md: api2go errors decoding any resource whose
// response carries a relationships block unless the target struct
// implements these, whether or not the caller cares about the data.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:        rm.Id,
		name:      rm.Name,
		gender:    rm.Gender,
		skinColor: rm.SkinColor,
		face:      rm.Face,
		hair:      rm.Hair,
		jobId:     rm.JobId,
		level:     rm.Level,
		gm:        rm.Gm == 1,
	}, nil
}
