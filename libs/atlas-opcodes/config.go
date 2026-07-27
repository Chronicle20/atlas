package opcodes

// Canonical socket-service names used to scope a shared tenant socket config to
// the service that owns each handler/writer entry. See HandlerConfig.Services.
const (
	ServiceLogin   = "login"
	ServiceChannel = "channel"
)

// HandlerConfig represents a client-to-server handler opcode mapping.
type HandlerConfig struct {
	OpCode    string                 `json:"opCode"`
	Validator string                 `json:"validator"`
	Handler   string                 `json:"handler"`
	Options   map[string]interface{} `json:"options"`
	// Services scopes this entry to specific socket services (e.g. ServiceLogin,
	// ServiceChannel). A tenant's socket config is shared by every service that
	// speaks the tenant's protocol, so entries carry which service(s) own them.
	// Empty means the entry applies to every service (legacy/untagged configs).
	Services []string `json:"services,omitempty"`
}

// WriterConfig represents a server-to-client writer opcode mapping.
type WriterConfig struct {
	OpCode  string                 `json:"opCode"`
	Writer  string                 `json:"writer"`
	Options map[string]interface{} `json:"options"`
	// Services scopes this entry to specific socket services; see
	// HandlerConfig.Services. Empty applies to every service.
	Services []string `json:"services,omitempty"`
}

// appliesToService reports whether a config entry tagged with the given service
// list is in scope for service. An empty list applies to every service so that
// untagged (legacy) configs behave exactly as before.
func appliesToService(services []string, service string) bool {
	if len(services) == 0 {
		return true
	}
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}
