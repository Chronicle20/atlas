package workers

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/mobskill"
	"atlas-data/skill"
	minio "atlas-data/storage/minio"
)

type Skill struct{}

func (Skill) Name() string        { return "SKILL" }
func (Skill) ArchiveName() string { return "Skill.wz" }

func (Skill) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, t, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Skill.wz: %w", err)
	}
	// String.wz Skill.img + MobSkill.img drive skill / mobskill names.
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "String.wz"); err != nil {
		l.WithError(err).Warnf("String.wz unavailable; skill names will be empty")
	} else {
		if err := skill.InitString(t, filepath.Join(root, "String.wz", "Skill.img.xml")); err != nil {
			l.WithError(err).Warnf("skill.InitString failed")
		}
		defer func() { _ = skill.GetSkillStringRegistry().Clear(t) }()
		if err := mobskill.InitString(t, filepath.Join(root, "String.wz", "MobSkill.img.xml")); err != nil {
			l.WithError(err).Warnf("mobskill.InitString failed")
		}
		defer func() { _ = mobskill.GetMobSkillStringRegistry().Clear(t) }()
	}
	// Register skills (per-job images) and the single MobSkill.img.
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), skill.RegisterSkill(db)); err != nil {
		return err
	}
	if err := mobskill.RegisterMobSkill(db)(l)(ctx)(filepath.Join(root, "Skill.wz", "MobSkill.img.xml")); err != nil {
		l.WithError(err).Warnf("mobskill RegisterMobSkill failed")
	}

	// Emit per-skill icons. Skill IDs live as SubProperty children of the
	// "skill" SubProperty in each per-job .img.
	prefix := minioAssetPrefix(p)
	var scanned, extracted, uploaded int
	for _, img := range file.Root().Images() {
		// MobSkill.img and others don't have job ids; skip them.
		if _, ok := imgID(img.Name()); !ok {
			continue
		}
		skillDir := findSub(img.Properties(), "skill")
		if skillDir == nil {
			continue
		}
		for _, child := range skillDir.Children() {
			sub, ok := child.(*property.SubProperty)
			if !ok {
				continue
			}
			skillId, err := strconv.ParseUint(sub.Name(), 10, 32)
			if err != nil {
				continue
			}
			scanned++
			icon, err := icons.ExtractSkillIcon(file, uint32(skillId))
			if err != nil || icon == nil {
				continue
			}
			extracted++
			key := fmt.Sprintf("%s/skill/%d/icon.png", prefix, skillId)
			if err := putPNG(ctx, mc, key, icon); err != nil {
				l.WithError(err).Warnf("upload skill icon %d", skillId)
				continue
			}
			uploaded++
		}
	}
	l.Infof("skill icons: scanned=%d extracted=%d uploaded=%d", scanned, extracted, uploaded)
	return nil
}

