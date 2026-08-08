package workers

import (
	"atlas-data/job"
	"atlas-data/mobskill"
	"atlas-data/skill"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"

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
	// Accumulate the FR-7.3 run summary across every per-job image. Deferred
	// so the summary is still emitted on a walk-level error (corrupt job
	// image, I/O failure) — matching data/processor.go's WorkerSkill branch,
	// which logs unconditionally regardless of err.
	var skillStats skill.StatsAccumulator
	defer skillStats.Log(l)
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), skillStats.Wrap(skill.NewProcessor(l, ctx, db).RegisterSkill)); err != nil {
		return err
	}
	if err := mobskill.NewProcessor(l, ctx, db).RegisterMobSkill(filepath.Join(root, "Skill.wz", "MobSkill.img.xml")); err != nil {
		l.WithError(err).Warnf("mobskill RegisterMobSkill failed")
	}

	// FR-2.1: the JOB pass folds into the SKILL worker rather than adding a
	// data.Workers entry, exactly as mobskill does. It re-reads the same
	// serialized Skill.wz tree, so monolithic-archive tenants (GMS v12's
	// all-in-one Data.wz) are handled by the runtime's sub-view with no
	// monolith-specific code (FR-2.5).
	var jobStats jobStats
	defer jobStats.Log(l)
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), jobStats.Wrap(job.NewProcessor(l, ctx, db).RegisterJob)); err != nil {
		return err
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
		props, err := img.Properties()
		if err != nil {
			return fmt.Errorf("skill worker: parse %s: %w", img.Name(), err)
		}
		skillDir := findSub(props, "skill")
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

// jobStats accumulates the JOB pass's ingest summary across every image the
// registration walk hands it (FR-1.4, task-202). Three counters, not one:
// a numeric image that produces no document is EXPECTED at v0.84+ (the ten
// Skill.wz/Dragon/ animation images, see job.Read's FR-1.1 branch) but a
// silent "written=N" is exactly what let those images blank every Evan
// document for months. skipped = numeric - written makes a recurrence
// diagnosable from logs alone, without a WZ walk.
//
// Mirrors skill.StatsAccumulator's Wrap/Log shape so both passes in
// Skill.Run read the same way.
type jobStats struct {
	images  int // every .img.xml handed to the register
	numeric int // those whose basename parses as a job id
	written int // JOB documents actually upserted
}

// Wrap adapts job.Processor.RegisterJob -- which returns the number of
// documents it wrote -- to the RegisterFunc shape registerAllInDirectory
// expects, accumulating into s. A failing register contributes to images
// and numeric but never to written.
func (s *jobStats) Wrap(rf func(path string) (int, error)) RegisterFunc {
	return func(path string) error {
		s.images++
		if _, ok := imgID(strings.TrimSuffix(filepath.Base(path), ".xml")); ok {
			s.numeric++
		}
		n, err := rf(path)
		if err != nil {
			return err
		}
		s.written += n
		return nil
	}
}

// Log emits the JOB-document ingest summary. A Skill.wz pass that produced
// no JOB documents leaves /data/jobs empty for the tenant, so it escalates
// to warn: silent success here is the failure mode the rejected transitional
// fallback would have hidden (PRD §8 Observability).
func (s *jobStats) Log(l logrus.FieldLogger) {
	l.Infof("job documents: images=%d numeric=%d written=%d skipped=%d", s.images, s.numeric, s.written, s.numeric-s.written)
	if s.written == 0 {
		l.Warnf("Skill.wz ingest produced no JOB documents; /data/jobs will be empty for this tenant")
	}
}
