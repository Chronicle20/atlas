package timer

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
