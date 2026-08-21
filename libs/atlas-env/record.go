package env

// Phase values published on the environment-status topic (design §4.4).
const (
	PhaseProvisioning = "PROVISIONING"
	PhaseActive       = "ACTIVE"
	PhaseDeactivating = "DEACTIVATING"
	PhaseDeleted      = "DELETED"
)

// Record is one environment as published on the environment-status topic.
// Overrides maps a service name to the NAMESPACE that serves it — not to a
// Deployment name. Deployment names are identical across namespaces in
// Atlas (atlas-character everywhere); the namespace is what varies and what
// the REST routing mechanism needs (design §4.4).
type Record struct {
	Name      Id                `json:"name"`
	Baseline  Id                `json:"baseline"`
	Namespace string            `json:"namespace"`
	Tenant    string            `json:"tenant"`
	Overrides map[string]string `json:"overrides"`
	Phase     string            `json:"phase"`
}

// Active reports whether the record's phase is PhaseActive. Any other phase
// (PROVISIONING, DEACTIVATING, DELETED) is not eligible for ownership or
// routing (FR-5.2, FR-5.7).
func (r Record) Active() bool { return r.Phase == PhaseActive }

// Provisionable reports whether the record's phase may serve requests on its
// own behalf: PhaseProvisioning (self-writes made during setup, e.g. the
// service-config rows bootstrap must write before the environment can go
// ACTIVE) or PhaseActive. PhaseDeactivating and PhaseDeleted are excluded.
// This is deliberately weaker than Active — it governs whether a request
// naming this environment is admitted at all, not whether the environment
// owns traffic (that remains Active's job via IsOwner).
func (r Record) Provisionable() bool {
	return r.Phase == PhaseProvisioning || r.Phase == PhaseActive
}
