package reactor

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Register(s *Storage, r model.Provider[RestModel]) error
	RegisterReactor(path string) error
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

func (p *ProcessorImpl) Register(s *Storage, r model.Provider[RestModel]) error {
	m, err := r()
	if err != nil {
		return err
	}
	_, err = s.Add(p.ctx)(m)()
	if err != nil {
		return err
	}
	return nil
}

func extractPathAndID(path string) (string, uint32, error) {
	// Extract the base filename
	base := filepath.Base(path)

	// Trim the ".img.xml" extension
	if !strings.HasSuffix(base, ".img.xml") {
		return "", 0, fmt.Errorf("invalid file format: %s", base)
	}
	idStr := strings.TrimSuffix(base, ".img.xml")

	// Convert to uint32
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to convert ID to uint32: %w", err)
	}

	// Extract the directory
	dir := filepath.Dir(path) + "/"

	return dir, uint32(id), nil
}

func (p *ProcessorImpl) RegisterReactor(path string) error {
	parentPath, reactorId, err := extractPathAndID(path)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return p.Register(NewStorage(p.l, tx), Read(p.l)(parentPath, reactorId, xml.FromParentPathProvider(7)))
	})
}
