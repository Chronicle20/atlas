package minioreconcile

import "time"

const defaultMinAgeHours = 48

// ReconcileInputModel is the JSON:API input for POST /api/data/minio/reconcile.
type ReconcileInputModel struct {
	Id            string   `json:"-"`
	KeepTenantIDs []string `json:"keepTenantIds"`
	MinAgeHours   int      `json:"minAgeHours"`
	DryRun        bool     `json:"dryRun"`
}

func (ReconcileInputModel) GetName() string                                     { return "minioReconciles" }
func (m ReconcileInputModel) GetID() string                                     { return m.Id }
func (m *ReconcileInputModel) SetID(id string) error                            { m.Id = id; return nil }
func (m *ReconcileInputModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (m *ReconcileInputModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// ToRequest maps the JSON:API input to a domain Request, defaulting
// MinAgeHours to defaultMinAgeHours when the input omits it (<= 0).
func (m ReconcileInputModel) ToRequest() Request {
	minAge := m.MinAgeHours
	if minAge <= 0 {
		minAge = defaultMinAgeHours
	}
	return Request{
		KeepTenantIDs: m.KeepTenantIDs,
		MinAgeHours:   minAge,
		DryRun:        m.DryRun,
	}
}

// ReconcileOutputModel is the JSON:API report.
type ReconcileOutputModel struct {
	Id            string      `json:"-"`
	DryRun        bool        `json:"dryRun"`
	MinAgeHours   int         `json:"minAgeHours"`
	TotalPrefixes int         `json:"totalPrefixes"`
	TotalBytes    int64       `json:"totalBytes"`
	Rows          []OutputRow `json:"rows"`
}

type OutputRow struct {
	Bucket   string `json:"bucket"`
	TenantID string `json:"tenantId"`
	Action   string `json:"action"`
	Count    int    `json:"count"`
	Bytes    int64  `json:"bytes"`
	Newest   string `json:"newest"`
}

func (ReconcileOutputModel) GetName() string                                     { return "minioReconciles" }
func (m ReconcileOutputModel) GetID() string                                     { return m.Id }
func (m *ReconcileOutputModel) SetID(id string) error                            { m.Id = id; return nil }
func (m *ReconcileOutputModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (m *ReconcileOutputModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// toOutput maps a domain Report to the JSON:API ReconcileOutputModel.
func toOutput(rep Report) ReconcileOutputModel {
	rows := make([]OutputRow, 0, len(rep.Rows))
	for _, row := range rep.Rows {
		rows = append(rows, OutputRow{
			Bucket: row.Bucket, TenantID: row.TenantID, Action: row.Action,
			Count: row.Count, Bytes: row.Bytes, Newest: row.Newest.UTC().Format(time.RFC3339),
		})
	}
	return ReconcileOutputModel{Id: "current", DryRun: rep.DryRun, MinAgeHours: rep.MinAgeHours, TotalPrefixes: rep.TotalPrefixes, TotalBytes: rep.TotalBytes, Rows: rows}
}
