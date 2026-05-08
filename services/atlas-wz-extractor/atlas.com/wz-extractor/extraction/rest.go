package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"time"
)

// UnitRest is the per-WZ-file view embedded in JobRestModel.
type UnitRest struct {
	WzFile      string  `json:"wzFile"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"startedAt"`
	CompletedAt *string `json:"completedAt"`
	Error       *string `json:"error"`
}

// JobRestModel is the JSON:API resource representation of an extraction job.
// Id is handled through GetID/SetID; json:"-" keeps it out of the attributes block.
type JobRestModel struct {
	Id             string     `json:"-"`
	TenantId       string     `json:"tenantId"`
	Region         string     `json:"region"`
	MajorVersion   uint16     `json:"majorVersion"`
	MinorVersion   uint16     `json:"minorVersion"`
	Status         string     `json:"status"`
	XmlOnly        bool       `json:"xmlOnly"`
	ImagesOnly     bool       `json:"imagesOnly"`
	UnitsTotal     int        `json:"unitsTotal"`
	UnitsCompleted int        `json:"unitsCompleted"`
	UnitsFailed    int        `json:"unitsFailed"`
	CreatedAt      string     `json:"createdAt"`
	UpdatedAt      string     `json:"updatedAt"`
	CompletedAt    *string    `json:"completedAt"`
	Units          []UnitRest `json:"units"`
}

func (j JobRestModel) GetName() string { return "wzExtractionJob" }
func (j JobRestModel) GetID() string   { return j.Id }

func (j *JobRestModel) SetID(id string) error {
	j.Id = id
	return nil
}

// SetToOneReferenceID satisfies api2go UnmarshalToOneRelations.
func (j *JobRestModel) SetToOneReferenceID(_, _ string) error { return nil }

// SetToManyReferenceIDs satisfies api2go UnmarshalToManyRelations.
func (j *JobRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// TransformJob converts domain Job + Units into a JobRestModel.
func TransformJob(j job.Job, units []job.Unit) JobRestModel {
	fmtTime := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	optTime := func(t time.Time) *string {
		if t.IsZero() {
			return nil
		}
		s := t.UTC().Format(time.RFC3339)
		return &s
	}

	ujs := make([]UnitRest, 0, len(units))
	for _, u := range units {
		var errPtr *string
		if u.ErrorMessage() != "" {
			e := u.ErrorMessage()
			errPtr = &e
		}
		ujs = append(ujs, UnitRest{
			WzFile:      u.WzFile(),
			Status:      string(u.Status()),
			StartedAt:   optTime(u.StartedAt()),
			CompletedAt: optTime(u.CompletedAt()),
			Error:       errPtr,
		})
	}

	return JobRestModel{
		Id:             j.Id(),
		TenantId:       j.TenantId(),
		Region:         j.Region(),
		MajorVersion:   j.MajorVersion(),
		MinorVersion:   j.MinorVersion(),
		Status:         string(j.Status()),
		XmlOnly:        j.XmlOnly(),
		ImagesOnly:     j.ImagesOnly(),
		UnitsTotal:     j.UnitsTotal(),
		UnitsCompleted: j.UnitsCompleted(),
		UnitsFailed:    j.UnitsFailed(),
		CreatedAt:      fmtTime(j.CreatedAt()),
		UpdatedAt:      fmtTime(j.UpdatedAt()),
		CompletedAt:    optTime(j.CompletedAt()),
		Units:          ujs,
	}
}
