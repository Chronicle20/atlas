package anniversary

import (
	"atlas-events/event/definition"
	"atlas-events/event/scheduling"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Scheduler schedules the durable work an ANNIVERSARY definition needs, both
// when it is newly enabled and when Handler.Evaluate finds the window has
// not opened yet.
type Scheduler struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewScheduler constructs a Scheduler.
func NewScheduler(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Scheduler {
	return &Scheduler{l: l, ctx: ctx, db: db}
}

// OnDefinitionEnabled schedules the row that will start the ANNIVERSARY
// window (FR-A2), applying FR-A4/FR-A6 at enable time too:
//   - a definition whose whole window has already elapsed schedules nothing
//     — a fresh enable of a long-expired definition must not spontaneously
//     start it;
//   - otherwise, the row is scheduled for scheduledStart (or now, if
//     scheduledStart is already in the past — the window is open right now).
func (s *Scheduler) OnDefinitionEnabled(d definition.Model) error {
	c, err := DecodeConfig(d.Configuration())
	if err != nil {
		return err
	}
	return s.scheduleStart(d.Id(), c)
}

// scheduleStart is the shared timing decision behind OnDefinitionEnabled and
// Handler.Evaluate's "not open yet" branch (FR-A6) — both need the same
// executeAt computation, so it lives here once rather than being duplicated.
func (s *Scheduler) scheduleStart(definitionId uuid.UUID, c Config) error {
	now := time.Now()
	if !c.ScheduledEnd.After(now) {
		// FR-A4: no retroactive occurrence. A definition whose window has
		// fully elapsed schedules nothing.
		return nil
	}

	executeAt := c.ScheduledStart
	if !executeAt.After(now) {
		executeAt = now
	}

	m, err := scheduling.NewBuilder(definitionId, scheduling.WorkTypeTriggerEvaluation).
		SetContext(json.RawMessage("{}")).
		SetExecuteAt(executeAt).
		SetDedupeKey(fmt.Sprintf("enable:%s", definitionId)).
		Build()
	if err != nil {
		return err
	}

	_, _, err = scheduling.NewAdministrator(s.l, s.ctx, s.db).Schedule(m)
	return err
}
