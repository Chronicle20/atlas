package workers

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/atlas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/atlas/pngenc"
	"github.com/Chronicle20/atlas/libs/atlas-wz/charparts"
	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
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
// trees. After registering the per-domain documents, the worker walks the WZ
// in memory via libs/atlas-wz/charparts to emit per-(partClass, id) atlas
// sprite sheets + manifests under
// <scope>/regions/<region>/versions/<major>.<minor>/atlases/<partClass>/<id>.{png,json}.
// atlas-renders' composite handler consumes those keys via Storage.GetAtlas.
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

	// Atlas emission. atlas-renders' composite handler fetches each part atlas
	// via Storage.GetAtlas(scope, region, version, partClass, id); without the
	// PNG+JSON pair the handler returns 500. Per-(partClass, id) failures are
	// logged and skipped so one bad template can't poison the entire ingest.
	if err := emitCharacterAtlases(ctx, l, mc, file, p); err != nil {
		return fmt.Errorf("emit character atlases: %w", err)
	}

	// Cross-archive sidecar: Base.wz/smap.img drives the equipment-vs-hair
	// occlusion filter (full helmets hiding bangs). The fetch is best-effort
	// because Base.wz is not always available in test fixtures — but without
	// smap.json downstream, atlas-renders disables occlusion and bangs paint
	// over helmets. The warning logged below makes that consequence visible.
	if err := emitSmapSidecar(ctx, l, mc, p); err != nil {
		l.WithError(err).Warn("smap sidecar emit failed; vslot-based occlusion will be disabled in atlas-renders")
	}
	return nil
}

// emitSmapSidecar fetches Base.wz, calls charparts.ExtractSmap, and PUTs the
// resulting layer-name → slot-codes map as character-meta/smap.json under the
// worker's scope/region/version prefix. atlas-renders reads this sidecar to
// drive the vslot/smap occlusion filter that hides hair bangs behind a full
// helmet.
//
// Errors from the cross-archive fetch and from a missing smap.img are wrapped
// and returned so the caller logs them as warnings — none of them are fatal
// to the Character ingest (the atlas PNG+JSON pairs were already emitted).
func emitSmapSidecar(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params) error {
	base, cleanup, err := fetchArchive(ctx, l, mc, p, "Base.wz")
	if err != nil {
		return fmt.Errorf("fetch Base.wz: %w", err)
	}
	defer cleanup()

	smap, err := charparts.ExtractSmap(base)
	if err != nil {
		return fmt.Errorf("extract smap: %w", err)
	}

	data, err := charparts.MarshalSmap(smap)
	if err != nil {
		return fmt.Errorf("marshal smap: %w", err)
	}

	key := fmt.Sprintf("%s/character-meta/smap.json", minioAssetPrefix(p))
	if err := putJSON(ctx, mc, key, data); err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	l.Infof("Character smap sidecar emitted: key=%s entries=%d", key, len(smap))
	return nil
}

// emitCharacterAtlases walks Character.wz via charparts.WalkCharacter, packs
// each PartSet into a deterministic atlas sheet + manifest, and uploads the
// (PNG, JSON) pair to MinIO under the canonical
// <prefix>/atlases/<partClass>/<id>.{png,json} keyspace. Per-template failures
// are logged and skipped; the only fatal errors are upload failures (so
// callers can rerun the worker without partial bucket state for the working
// templates).
func emitCharacterAtlases(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, file *wz.File, p Params) error {
	sets, err := charparts.WalkCharacter(file, nil)
	if err != nil {
		return fmt.Errorf("character walk: %w", err)
	}
	prefix := minioAssetPrefix(p)
	var emitted, skipped int
	for _, set := range sets {
		inputs := charparts.ToAtlasInputs(set)
		if len(inputs) == 0 {
			skipped++
			continue
		}
		sheet, m, err := atlas.Pack(inputs)
		if err != nil {
			l.WithError(err).Warnf("atlas.Pack partClass=%s id=%d", set.PartClass, set.ID)
			skipped++
			continue
		}
		// Pack copies Input.Name into Sprite.Part verbatim. The dotted form
		// "stance.frame.part" lets us split each Sprite back into the donor's
		// (stance, frame, part) tags so the on-disk manifest matches the
		// composite handler's lookup expectations.
		for i := range m.Sprites {
			stance, frame, part, ok := charparts.DecodePartName(m.Sprites[i].Part)
			if !ok {
				// Defensive: keep the dotted form as Part so the entry isn't
				// silently lost. atlas.Pack only inserts Names we wrote, so
				// this branch indicates a future schema drift.
				continue
			}
			m.Sprites[i].Stance = stance
			m.Sprites[i].Frame = frame
			m.Sprites[i].Part = part
		}
		m.ID = set.ID
		m.PartClass = set.PartClass
		m.Vslot = set.Info.Vslot

		pngKey := fmt.Sprintf("%s/atlases/%s/%d.png", prefix, set.PartClass, set.ID)
		var pngBuf bytes.Buffer
		if err := pngenc.Encode(&pngBuf, sheet); err != nil {
			return fmt.Errorf("pngenc.Encode %s: %w", pngKey, err)
		}
		if err := putBytes(ctx, mc, pngKey, pngBuf.Bytes(), "image/png"); err != nil {
			return fmt.Errorf("put atlas png %s: %w", pngKey, err)
		}

		manKey := fmt.Sprintf("%s/atlases/%s/%d.json", prefix, set.PartClass, set.ID)
		manBytes, err := manifest.Marshal(m)
		if err != nil {
			return fmt.Errorf("manifest.Marshal %s: %w", manKey, err)
		}
		if err := putJSON(ctx, mc, manKey, manBytes); err != nil {
			return fmt.Errorf("put atlas json %s: %w", manKey, err)
		}
		emitted++
	}
	l.Infof("Character atlas emit: emitted=%d skipped=%d total=%d", emitted, skipped, len(sets))
	return nil
}
