package diagnostics

// RestModel mirrors the diagnostics sub-object of the tenant configuration
// document owned by atlas-configurations. Zero value is "off", so a tenant
// document written before the object existed traces nothing (FR-1.2).
type RestModel struct {
	TracePackets bool `json:"tracePackets"`
}
