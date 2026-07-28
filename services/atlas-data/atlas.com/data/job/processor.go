package job

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"strconv"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	GetSkillsForJob(jobId uint32) (RestModel, bool)
	Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error
	// RegisterJob reads one Skill.wz per-job image and returns the number of
	// JOB documents written (0 for a non-numeric image such as MobSkill.img).
	// The count feeds the SKILL worker's ingest observability (design D12).
	RegisterJob(path string) (int, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "JOB")
}

// GetSkillsForJob resolves the tenant's JOB document. Storage.ByIdProvider
// supplies the registry cache, the tenant rows, and the canonical-tenant
// fallback keyed on canonical.TenantId(region, major, minor) (FR-1.3). Any
// error — including gorm.ErrRecordNotFound — collapses to ok=false, so
// "unknown job id" and "job absent from this tenant's version" are the same
// 404 (FR-3.2).
func (p *ProcessorImpl) GetSkillsForJob(jobId uint32) (RestModel, bool) {
	m, err := NewStorage(p.l, p.db).GetById(p.ctx)(strconv.Itoa(int(jobId)))
	if err != nil {
		p.l.WithError(err).Debugf("Unable to locate JOB document [%d].", jobId)
		return RestModel{Id: jobId, Skills: []uint32{}}, false
	}
	return m, true
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error {
	ms, err := r()
	if err != nil {
		return err
	}
	for _, m := range ms {
		if _, err = s.Add(p.ctx)(m)(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) RegisterJob(path string) (int, error) {
	written := 0
	err := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		ms, err := Read(p.l)(p.ctx)(xml.FromPathProvider(path))()
		if err != nil {
			return err
		}
		if err = p.Register(NewStorage(p.l, tx), model.FixedProvider(ms)); err != nil {
			return err
		}
		written = len(ms)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return written, nil
}
