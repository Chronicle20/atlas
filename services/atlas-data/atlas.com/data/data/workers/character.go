package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/characters/templates"
	"atlas-data/cosmetic/face"
	"atlas-data/cosmetic/hair"
	"atlas-data/equipment"
	"atlas-data/item"
	minio "atlas-data/storage/minio"
)

type Character struct{}

func (Character) Name() string        { return "CHARACTER" }
func (Character) ArchiveName() string { return "Character.wz" }

// Character.wz holds equipment (top-level subdirs) plus cosmetic Face and Hair
// trees. Atlas-style sprite packing (per partClass sprite sheets and
// manifests) is a STATED LIMITATION: the per-stance/per-frame sprite assembly
// in the deleted atlas-wz-extractor image/character_parts.go is ~500 LOC of
// link-resolution and joint logic that does not have a current
// libs/atlas-wz/*-shaped wrapper. Cleanly re-implementing it here would
// double the size of this worker, so it is deferred. Equipment, Face, and
// Hair documents are still fully populated below; only the atlases/<part>
// MinIO PNG+JSON outputs are missing.
func (Character) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Character.wz: %w", err)
	}
	base := filepath.Join(root, "Character.wz")
	// Equipment names live in String.wz/Eqp.img (nested by sub-category).
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "String.wz"); err != nil {
		l.WithError(err).Warnf("String.wz unavailable; equipment names will be empty")
	} else {
		if err := item.InitStringNested(db)(l)(ctx)(filepath.Join(root, "String.wz", "Eqp.img.xml")); err != nil {
			l.WithError(err).Warnf("item.InitStringNested(Eqp) failed")
		}
	}

	// Face and Hair live under Character.wz/Face and Character.wz/Hair.
	faceDir := filepath.Join(base, "Face")
	if err := registerAllInDirectory(l, ctx, faceDir, face.RegisterFace(db)); err != nil {
		l.WithError(err).Warnf("walk %s", faceDir)
	}
	hairDir := filepath.Join(base, "Hair")
	if err := registerAllInDirectory(l, ctx, hairDir, hair.RegisterHair(db)); err != nil {
		l.WithError(err).Warnf("walk %s", hairDir)
	}

	// Equipment registration walks the rest of Character.wz subdirs. The
	// equipment Read tolerates non-equipment .img.xml entries because Face/Hair
	// dirs are already registered above and equipment.Read will fail benignly
	// for them; the walker logs and continues.
	if err := registerAllInDirectory(l, ctx, base, equipment.RegisterEquipment(db)); err != nil {
		return err
	}

	// Character creation templates come from Etc.wz/MakeCharInfo.img.xml,
	// which is not the Character.wz archive. We fetch Etc.wz so the templates
	// register populates alongside Character data; without this no character
	// creation flow would have starter template data.
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "Etc.wz"); err == nil {
		mkChar := filepath.Join(root, "Etc.wz", "MakeCharInfo.img.xml")
		if err := templates.RegisterCharacterTemplate(db)(l)(ctx)(mkChar); err != nil {
			l.WithError(err).Warnf("templates.RegisterCharacterTemplate failed")
		}
	}
	return nil
}
