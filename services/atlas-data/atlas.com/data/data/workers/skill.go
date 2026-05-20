package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Skill struct{}

func (Skill) Name() string        { return "SKILL" }
func (Skill) ArchiveName() string { return "Skill.wz" }

func (Skill) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	return nil
}
