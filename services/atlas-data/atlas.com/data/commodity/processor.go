package commodity

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "COMMODITY")
}

// Register adds each item via the storage's per-call commit. No outer
// transaction wraps the loop — a connection drop or single-row failure now
// preserves successfully-committed rows so a retry can converge.
//
// See task-076 F2: the prior outer ExecuteTransaction wrapped the entire
// Etc.wz import in one long-lived transaction and was fatal to any conn
// blip across a multi-thousand-row register.
func Register(s *document.Storage[string, RestModel]) func(ctx context.Context) func(r model.Provider[[]RestModel]) error {
	return func(ctx context.Context) func(r model.Provider[[]RestModel]) error {
		return func(r model.Provider[[]RestModel]) error {
			ms, err := r()
			if err != nil {
				return err
			}
			for _, m := range ms {
				if _, err := s.Add(ctx)(m)(); err != nil {
					return err
				}
			}
			return nil
		}
	}
}

func RegisterCommodity(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
		return func(ctx context.Context) func(path string) error {
			return func(path string) error {
				return Register(NewStorage(l, db))(ctx)(Read(l)(xml.FromPathProvider(path)))
			}
		}
	}
}
