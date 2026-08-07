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
