package diagnostics

// RestModel carries per-tenant operational diagnostics switches. Every
// field must be zero-value safe: a tenant document written before this
// object existed unmarshals to the zero value, which is "off" (FR-1.2),
// so no backfill and no migration is required.
//
// TracePackets is deliberately dangerous -- with it on, and the serving
// pod at LOG_LEVEL=Debug, login-family packets put account passwords,
// PICs/PINs and HWIDs into the log stream in plaintext. Logs captured
// while it is on are credential-bearing material.
type RestModel struct {
	TracePackets bool `json:"tracePackets"`
}
