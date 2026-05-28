package seeder

// Group declares one (POST /<prefix>/seed, GET /<prefix>/seed/status) pair.
type Group struct {
	Name       string         // stored as seed_state.group_name; e.g. "drops"
	URLPrefix  string         // e.g. "/drops" → routes POST /drops/seed
	Subdomains []SubdomainAny
}
