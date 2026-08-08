package rest

import (
	"atlas-data/ingestrun"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

// IngestRunWorkerRestModel is one worker's slot in the response.
type IngestRunWorkerRestModel struct {
	Name       string  `json:"name"`
	State      string  `json:"state"`
	StartedAt  *string `json:"startedAt"`
	FinishedAt *string `json:"finishedAt"`
	Error      *string `json:"error"`
}

// IngestRunRestModel is the JSON:API projection of an ingestrun.Record.
//
// workersTotal / workersComplete are derived here rather than stored: a value
// computable from the worker list has no business being persisted twice and
// drifting.
type IngestRunRestModel struct {
	Id              string                     `json:"-"`
	RunId           string                     `json:"runId"`
	JobName         string                     `json:"jobName"`
	Scope           string                     `json:"scope"`
	Region          string                     `json:"region"`
	Version         string                     `json:"version"`
	Tenant          string                     `json:"tenant,omitempty"`
	Phase           string                     `json:"phase"`
	StartedAt       *string                    `json:"startedAt"`
	FinishedAt      *string                    `json:"finishedAt"`
	Reason          *string                    `json:"reason"`
	WorkersTotal    int                        `json:"workersTotal"`
	WorkersComplete int                        `json:"workersComplete"`
	Workers         []IngestRunWorkerRestModel `json:"workers"`
}

func (r IngestRunRestModel) GetName() string { return "ingestRun" }

func (r IngestRunRestModel) GetID() string { return r.Id }

func (r *IngestRunRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r IngestRunRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r IngestRunRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r IngestRunRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *IngestRunRestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *IngestRunRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r *IngestRunRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// toIngestRunRestModel projects a stored record onto the wire. phase is passed
// separately because `unknown` is computed at read time and deliberately never
// stored — a record that later proves alive recovers on the next poll with no
// repair path.
func toIngestRunRestModel(rec ingestrun.Record, phase ingestrun.Phase, id string) IngestRunRestModel {
	ws := make([]IngestRunWorkerRestModel, 0, len(rec.Workers))
	for _, w := range rec.Workers {
		ws = append(ws, IngestRunWorkerRestModel{
			Name:       w.Name,
			State:      string(w.State),
			StartedAt:  rfc3339Ptr(w.StartedAt),
			FinishedAt: rfc3339Ptr(w.FinishedAt),
			Error:      strPtr(w.Error),
		})
	}
	m := IngestRunRestModel{
		Id:              id,
		RunId:           rec.RunId,
		JobName:         rec.JobName,
		Scope:           rec.Scope,
		Region:          rec.Region,
		Version:         rec.Version,
		Tenant:          rec.Tenant,
		Phase:           string(phase),
		FinishedAt:      rfc3339Ptr(rec.FinishedAt),
		Reason:          strPtr(rec.Reason),
		WorkersTotal:    len(rec.Workers),
		WorkersComplete: rec.CompleteCount(),
		Workers:         ws,
	}
	if !rec.StartedAt.IsZero() {
		m.StartedAt = rfc3339Ptr(&rec.StartedAt)
	}
	return m
}
