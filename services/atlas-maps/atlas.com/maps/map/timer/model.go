package timer

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

type Entry struct {
	tenant            tenant.Model
	characterId       uint32
	field             field.Model
	forcedReturnMapId _map.Id
	seconds           uint32
	token             uuid.UUID
	expiresAt         time.Time
	timer             *time.Timer
}

func (e Entry) Tenant() tenant.Model       { return e.tenant }
func (e Entry) CharacterId() uint32        { return e.characterId }
func (e Entry) Field() field.Model         { return e.field }
func (e Entry) ForcedReturnMapId() _map.Id { return e.forcedReturnMapId }
func (e Entry) Seconds() uint32            { return e.seconds }
func (e Entry) Token() uuid.UUID           { return e.token }
func (e Entry) ExpiresAt() time.Time       { return e.expiresAt }
func (e Entry) Timer() *time.Timer         { return e.timer }

type EntryBuilder struct {
	e Entry
}

func NewEntryBuilder() *EntryBuilder { return &EntryBuilder{} }

func (b *EntryBuilder) SetTenant(t tenant.Model) *EntryBuilder        { b.e.tenant = t; return b }
func (b *EntryBuilder) SetCharacterId(id uint32) *EntryBuilder        { b.e.characterId = id; return b }
func (b *EntryBuilder) SetField(f field.Model) *EntryBuilder          { b.e.field = f; return b }
func (b *EntryBuilder) SetForcedReturnMapId(id _map.Id) *EntryBuilder { b.e.forcedReturnMapId = id; return b }
func (b *EntryBuilder) SetSeconds(s uint32) *EntryBuilder             { b.e.seconds = s; return b }
func (b *EntryBuilder) SetToken(t uuid.UUID) *EntryBuilder            { b.e.token = t; return b }
func (b *EntryBuilder) SetExpiresAt(t time.Time) *EntryBuilder        { b.e.expiresAt = t; return b }
func (b *EntryBuilder) SetTimer(t *time.Timer) *EntryBuilder          { b.e.timer = t; return b }
func (b *EntryBuilder) Build() Entry                                  { return b.e }
