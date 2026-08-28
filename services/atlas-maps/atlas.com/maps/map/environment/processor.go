package environment

import (
	"context"
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrBlankName is returned by Set for an empty or whitespace-only object name.
// Object names are otherwise opaque: no obstacle/environment name index exists
// in libs/atlas-wz or atlas-data, so a name the client does not know is a
// silent client-side no-op rather than a server error.
var ErrBlankName = errors.New("environment object name must not be blank")

type Processor interface {
	Set(f field.Model, kind field.ObjectKind, name string, state uint32) (ObjectEntry, error)
	Reset(f field.Model) []ObjectEntry
	GetAll(f field.Model) []ObjectEntry
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Set(f field.Model, kind field.ObjectKind, name string, state uint32) (ObjectEntry, error) {
	if strings.TrimSpace(name) == "" {
		return ObjectEntry{}, ErrBlankName
	}
	t := tenant.MustFromContext(p.ctx)
	entry := ObjectEntry{Kind: kind, Name: name, State: state}
	getRegistry().Set(FieldKey{Tenant: t, Field: f}, entry)
	p.l.Debugf("Environment object [%s] kind [%s] set to state [%d] in map [%d] instance [%s].", name, kind, state, f.MapId(), f.Instance())
	return entry, nil
}

// Reset clears the field's tracked entries and returns what was cleared, so the
// caller can build the per-object reset sweep. FieldObstacleAllReset restores
// only the client's obstacle list, so non-obstacle named objects must be zeroed
// explicitly -- see design.md section 1.2.
func (p *ProcessorImpl) Reset(f field.Model) []ObjectEntry {
	t := tenant.MustFromContext(p.ctx)
	cleared := getRegistry().Clear(FieldKey{Tenant: t, Field: f})
	p.l.Debugf("Environment reset in map [%d] instance [%s]; cleared [%d] object(s).", f.MapId(), f.Instance(), len(cleared))
	return cleared
}

func (p *ProcessorImpl) GetAll(f field.Model) []ObjectEntry {
	t := tenant.MustFromContext(p.ctx)
	return getRegistry().Get(FieldKey{Tenant: t, Field: f})
}
