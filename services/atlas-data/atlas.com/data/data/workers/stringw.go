package workers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/item"
	minio "atlas-data/storage/minio"
)

type String struct{}

func (String) Name() string        { return "STRING" }
func (String) ArchiveName() string { return "String.wz" }

func (String) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize String.wz: %w", err)
	}
	stringDir := filepath.Join(root, "String.wz")

	// Item string search index: flat tables (Consume/Cash/Etc/Ins/Pet) populate
	// item_string_search_index directly; Eqp.img is nested by sub-category.
	flat := []string{"Consume.img.xml", "Cash.img.xml", "Etc.img.xml", "Ins.img.xml", "Pet.img.xml"}
	for _, name := range flat {
		path := filepath.Join(stringDir, name)
		if _, statErr := os.Stat(path); statErr != nil {
			l.WithError(statErr).Debugf("skipping %s (absent)", name)
			continue
		}
		if err := item.InitStringFlat(db)(l)(ctx)(path); err != nil {
			l.WithError(err).Warnf("init item string from %s", name)
		}
	}
	eqp := filepath.Join(stringDir, "Eqp.img.xml")
	if _, statErr := os.Stat(eqp); statErr == nil {
		if err := item.InitStringNested(db)(l)(ctx)(eqp); err != nil {
			l.WithError(err).Warnf("init item string from Eqp.img.xml")
		}
	}
	return nil
}
