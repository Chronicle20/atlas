package diagnostics

// RestModel is the projection's deserialization mirror of the
// diagnostics.RestModel produced by atlas-configurations' tenant document
// (services/atlas-configurations/atlas.com/configurations/tenants/diagnostics/rest.go).
// Same package name, same struct, same json tag: the projection decodes
// this straight off the tenant envelope's config bytes.
//
// Every field must be zero-value safe: a tenant document written before
// this object existed unmarshals to the zero value, which is "off"
// (FR-1.2), so no backfill and no migration is required.
type RestModel struct {
	TracePackets bool `json:"tracePackets"`
}
