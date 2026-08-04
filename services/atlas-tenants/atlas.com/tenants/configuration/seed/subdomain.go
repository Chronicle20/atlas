package seed

import (
	"atlas-tenants/configuration"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"gorm.io/gorm"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Entry is the decoded JSON:API data object from one catalog file:
// {id, type, attributes}. It is stored verbatim as one element of the
// configuration row's resource_data.data array, so a seeded entry is
// byte-identical to one created through the CRUD API.
type Entry = map[string]interface{}

// Subdomain adapts one tenant-configuration resource onto
// libs/atlas-seeder. All three transport resources share one storage
// shape — a single (tenant, resource_name) row whose resource_data.data
// is an array — so they share one implementation parameterised by name.
type Subdomain struct {
	// resource is simultaneously the seeder subdomain name, the catalog
	// subdirectory, the JSON:API data.type, and the configurations
	// row's resource_name. Keeping them identical is what lets the UI
	// read the status map by resource name.
	resource string
}

var _ seeder.Subdomain[Entry, Entry] = Subdomain{}

func NewSubdomain(resource string) Subdomain { return Subdomain{resource: resource} }

func (s Subdomain) Name() string { return s.resource }
func (s Subdomain) Path() string { return s.resource }
func (s Subdomain) Type() string { return s.resource }

// EntityIDPattern is nil: the catalog filename does not encode the
// entity id. flight-temple-of-time-leafre.json holds
// "temple-of-time-return-flight", so the id comes from data.id
// (libs/atlas-seeder/seed.go handles the nil-pattern case).
func (s Subdomain) EntityIDPattern() *regexp.Regexp { return nil }

func (s Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	t, err := tenantFrom(db)
	if err != nil {
		return 0, err
	}
	return configuration.DeleteConfigurationByResourceName(db, t.Id(), s.resource)
}

func (s Subdomain) Decode(payload []byte) (Entry, error) {
	env, err := seeder.ParseEnvelope(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.resource, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("%s: parse payload: %w", s.resource, err)
	}
	var entry Entry
	if err := json.Unmarshal(raw["data"], &entry); err != nil {
		return nil, fmt.Errorf("%s: parse data object: %w", s.resource, err)
	}
	if env.Data.Type != s.resource {
		return nil, fmt.Errorf("%s: data.type = %q", s.resource, env.Data.Type)
	}
	return entry, nil
}

func (s Subdomain) Build(_ tenant.Model, entityID string, entry Entry) ([]Entry, error) {
	if entry == nil {
		return nil, fmt.Errorf("%s: nil entry for %q", s.resource, entityID)
	}
	entry["id"] = entityID
	return []Entry{entry}, nil
}

// BulkCreate appends this file's single entry to the tenant's
// configuration row. seeder.Seed holds a per-(tenant, group) mutex for
// the whole run and the three resources live in different groups, so
// these read-modify-writes never contend.
func (s Subdomain) BulkCreate(db *gorm.DB, entries []Entry) error {
	t, err := tenantFrom(db)
	if err != nil {
		return err
	}
	return configuration.AppendConfigurationEntries(db, t.Id(), s.resource, entries)
}

func (s Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	t, err := tenantFrom(db)
	if err != nil {
		return 0, nil, err
	}
	return configuration.CountConfigurationEntries(db, t.Id(), s.resource)
}

// tenantFrom reads the tenant off the *gorm.DB's context. libs/atlas-seeder
// hands every Subdomain method a db already carrying the seed context
// (seed.go's db.WithContext(ctx)), and Seed itself guarantees that context
// has a tenant. Failing loudly here beats silently seeding the zero tenant.
func tenantFrom(db *gorm.DB) (tenant.Model, error) {
	t, err := tenant.FromContext(db.Statement.Context)()
	if err != nil {
		return tenant.Model{}, fmt.Errorf("seed: no tenant in context: %w", err)
	}
	return t, nil
}
