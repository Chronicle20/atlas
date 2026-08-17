package environments

// Builder assembles a RestModel for the environment create/update path. It
// is production code (not a *_testhelpers.go constructor) so that both
// tests and any future callers share the same construction rules.
type Builder struct {
	name      string
	baseline  string
	namespace string
	tenant    string
	overrides map[string]string
	phase     string
}

func NewBuilder() *Builder {
	return &Builder{overrides: make(map[string]string)}
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetBaseline(baseline string) *Builder {
	b.baseline = baseline
	return b
}

func (b *Builder) SetNamespace(namespace string) *Builder {
	b.namespace = namespace
	return b
}

func (b *Builder) SetTenant(tenant string) *Builder {
	b.tenant = tenant
	return b
}

// SetOverride records that the named service is served out of namespace
// (not a Deployment name - see RestModel/env.Record doc) for this
// environment. May be called multiple times to add further overrides.
func (b *Builder) SetOverride(service string, namespace string) *Builder {
	b.overrides[service] = namespace
	return b
}

func (b *Builder) SetPhase(phase string) *Builder {
	b.phase = phase
	return b
}

func (b *Builder) Build() RestModel {
	return RestModel{
		Name:      b.name,
		Baseline:  b.baseline,
		Namespace: b.namespace,
		Tenant:    b.tenant,
		Overrides: b.overrides,
		Phase:     b.phase,
	}
}
