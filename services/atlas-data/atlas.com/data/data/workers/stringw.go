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

// stringSources is the outcome of layout detection over the serialized
// String.wz tree (task-172 C-4).
type stringSources struct {
	flat       []string // present modern flat images (Consume/Cash/Etc/Ins/Pet)
	eqp        string   // modern Eqp.img.xml, "" when absent
	legacyItem string   // legacy single Item.img.xml; when set it is the sole item-string source
}

// resolveStringSources decides which sources feed the item-string registries.
//
// The legacy pre-v83 layout ships a single String/Item.img holding every
// category (Con/Ins/Etc/Eqp/Pet). When present it is the authoritative,
// complete item-string source and takes precedence — the modern per-category
// images are absent in that layout anyway. Keying the decision off the
// per-category images (an earlier heuristic) breaks GMS v12/v48: those ship
// Item.img AND a standalone Pet.img, so the presence of Pet.img.xml made the
// resolver treat the set as modern and skip Item.img entirely, dropping every
// non-pet item name (task-172 C-4, caught in E2E on real v12/v48 data).
//
// Modern layouts (v83+/JMS) have no Item.img, so they fall through to the
// per-category branch and behave exactly as before.
func resolveStringSources(stringDir string) stringSources {
	var src stringSources
	if p := filepath.Join(stringDir, "Item.img.xml"); fileExists(p) {
		src.legacyItem = p
		return src
	}
	for _, name := range []string{"Consume.img.xml", "Cash.img.xml", "Etc.img.xml", "Ins.img.xml", "Pet.img.xml"} {
		p := filepath.Join(stringDir, name)
		if fileExists(p) {
			src.flat = append(src.flat, p)
		}
	}
	if p := filepath.Join(stringDir, "Eqp.img.xml"); fileExists(p) {
		src.eqp = p
	}
	return src
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

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

	src := resolveStringSources(stringDir)

	// Item string search index: flat tables (Consume/Cash/Etc/Ins/Pet)
	// populate item_string_search_index directly; Eqp.img is nested by
	// sub-category.
	for _, path := range src.flat {
		if err := item.InitStringFlat(db)(l)(ctx)(path); err != nil {
			l.WithError(err).Warnf("init item string from %s", filepath.Base(path))
		}
	}
	if src.eqp != "" {
		if err := item.InitStringNested(db)(l)(ctx)(src.eqp); err != nil {
			l.WithError(err).Warnf("init item string from Eqp.img.xml")
		}
	}
	// Legacy pre-v83 layout: one Item.img whose Con/Cash/Etc/Ins/Pet/Eqp
	// children carry the same name/desc leaves. InitStringFlat's walker
	// recurses through non-numeric nodes, so a single pass harvests all
	// six subtrees including Eqp's sub-category nesting (task-172 C-4).
	if src.legacyItem != "" {
		l.Infof("legacy String layout detected (single Item.img) — initializing item strings from it")
		if err := item.InitStringFlat(db)(l)(ctx)(src.legacyItem); err != nil {
			l.WithError(err).Warnf("init item string from legacy Item.img.xml")
		}
	}
	return nil
}
