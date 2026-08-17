package environments

// RestModel is the JSON:API resource shape for one environment. Its field
// names mirror libs/atlas-env.Record exactly (task-232 Task 18) because the
// outbox envelope's config is built directly from these values — a
// mismatch here would be invisible at compile time (Record lives in a
// different module) and would surface only as a silently empty registry at
// runtime downstream.
type RestModel struct {
	Id        string            `json:"-"`
	Name      string            `json:"name"`
	Baseline  string            `json:"baseline"`
	Namespace string            `json:"namespace"`
	Tenant    string            `json:"tenant"`
	Overrides map[string]string `json:"overrides"`
	Phase     string            `json:"phase"`
}

func (r RestModel) GetName() string {
	return "environments"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}
