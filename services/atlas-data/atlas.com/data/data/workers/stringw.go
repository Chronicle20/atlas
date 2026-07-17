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
	legacyItem string   // legacy single Item.img.xml; set only when NO modern image exists
}

// resolveStringSources decides which sources feed the item-string
// registries. Modern images always win; the legacy pre-v83 layout (one
// String/Item.img with Con/Cash/Etc/Ins/Pet/Eqp children) engages only
// when no modern image is present, so v83+/JMS behavior is unchanged.
func resolveStringSources(stringDir string) stringSources {
	var src stringSources
	for _, name := range []string{"Consume.img.xml", "Cash.img.xml", "Etc.img.xml", "Ins.img.xml", "Pet.img.xml"} {
		p := filepath.Join(stringDir, name)
		if _, err := os.Stat(p); err == nil {
			src.flat = append(src.flat, p)
		}
	}
	if p := filepath.Join(stringDir, "Eqp.img.xml"); fileExists(p) {
		src.eqp = p
	}
	if len(src.flat) == 0 && src.eqp == "" {
		if p := filepath.Join(stringDir, "Item.img.xml"); fileExists(p) {
			src.legacyItem = p
		}
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
