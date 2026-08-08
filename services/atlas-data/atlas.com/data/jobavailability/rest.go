package jobavailability

import "strconv"

// RestModel is one job identity available (released/playable) at the
// requesting tenant's client version. Id is the version-appropriate wire
// id (job.Id is uint16) -- NOT the version-blind job.Identity token -- so
// the frontend preset selector can round-trip it straight back into
// whatever job-id field the tenant's version expects.
type RestModel struct {
	Id   uint16 `json:"-"`
	Name string `json:"name"`
	// Parent is this version's advancement parent, as a WIRE id, or nil for
	// a branch root. A POINTER, not a plain uint16, because Beginner is a
	// legitimate wire id 0: "parent": 0 and "parent": null must not collide
	// (task-202 design D8, FR-3.3). It also never points at a job absent
	// from this version's response (FR-3.4) -- job.Set.ParentWire filters on
	// availability.
	Parent *uint16 `json:"parent"`
	// Identity is the version-blind canonical job token. Exposed so a client
	// can key version-stable curation (atlas-ui's rail grouping and accent
	// colours) on a job CONCEPT rather than on a wire id that means
	// different things per version -- wire id 500 is Gm at gms 48.1 and
	// Pirate at gms 72.1 (task-202 design D9).
	Identity uint16 `json:"identity"`
}

func (r RestModel) GetName() string { return "job-availability" }
func (r RestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint16(id)
	return nil
}
