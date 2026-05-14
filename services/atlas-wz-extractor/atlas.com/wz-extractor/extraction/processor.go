package extraction

import (
	wzimage "atlas-wz-extractor/image"
	"atlas-wz-extractor/wz"
	wzxml "atlas-wz-extractor/xml"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const envParallelism = "WZ_EXTRACT_PARALLELISM"

type Processor interface {
	// Extract preserves today's entry-point: list every WZ file under the
	// tenant's input dir, wipe the character cache (unless xmlOnly), and
	// process them in parallel via a bounded worker pool. Used by the
	// in-process tests; the cross-pod path uses ExtractUnit through Kafka.
	Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error

	// ExtractUnit processes one WZ file. Returns non-nil error only when
	// wz.Open fails ("couldn't even open the file"); per-stage errors are
	// logged but do not flip the unit to failed (continue-on-error semantics
	// from the original whole-list loop).
	ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error
}

type processorImpl struct {
	inputDir     string
	outputXmlDir string
	outputImgDir string
}

func NewProcessor(inputDir, outputXmlDir, outputImgDir string) Processor {
	return &processorImpl{
		inputDir:     inputDir,
		outputXmlDir: outputXmlDir,
		outputImgDir: outputImgDir,
	}
}

func (p *processorImpl) Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error {
	t := tenant.MustFromContext(ctx)
	tenantPath := TenantPath(t)
	inputPath := filepath.Join(p.inputDir, tenantPath)

	wzFiles, err := filepath.Glob(filepath.Join(inputPath, "*.wz"))
	if err != nil {
		return fmt.Errorf("unable to list WZ files: %w", err)
	}
	if len(wzFiles) == 0 {
		return fmt.Errorf("no WZ files found in [%s]", inputPath)
	}
	l.Infof("Found [%d] WZ files in [%s].", len(wzFiles), inputPath)

	if !xmlOnly {
		imgOutPath := filepath.Join(p.outputImgDir, tenantPath)
		if err := wipeCharacterCache(imgOutPath); err != nil {
			l.WithError(err).Warnf("Unable to wipe character cache.")
		}
	}

	workers := ParallelismFromEnv(l)
	files := make([]string, 0, len(wzFiles))
	for _, full := range wzFiles {
		files = append(files, filepath.Base(full))
	}

	runPool(ctx, l, files, workers, func(c context.Context, wzName string) error {
		return p.ExtractUnit(l, c, wzName, xmlOnly, imagesOnly)
	})
	return nil
}

func (p *processorImpl) ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error {
	t := tenant.MustFromContext(ctx)
	tenantPath := TenantPath(t)
	inputPath := filepath.Join(p.inputDir, tenantPath)
	xmlOutPath := filepath.Join(p.outputXmlDir, tenantPath)
	imgOutPath := filepath.Join(p.outputImgDir, tenantPath)
	wzPath := filepath.Join(inputPath, wzFile)

	l = l.WithField("wzFile", wzFile)
	l.Info("processing wz unit")

	f, err := wz.Open(l, wzPath)
	if err != nil {
		return fmt.Errorf("unable to open WZ file [%s]: %w", wzFile, err)
	}
	defer f.Close()

	if !imagesOnly {
		if err := wzxml.SerializeToDirectory(l, f, xmlOutPath); err != nil {
			l.WithError(err).Errorf("Unable to serialize [%s] to XML.", wzFile)
		}
	}

	if !xmlOnly {
		if err := wzimage.ExtractIcons(l, f, imgOutPath); err != nil {
			l.WithError(err).Errorf("Unable to extract icons from [%s].", wzFile)
		}
		if err := wzimage.ExtractMinimaps(l, f, imgOutPath); err != nil {
			l.WithError(err).Errorf("Unable to extract minimaps from [%s].", wzFile)
		}
		if err := RenderMaps(ctx, l, f, imgOutPath); err != nil {
			l.WithError(err).Errorf("Unable to render maps from [%s].", wzFile)
		}
	}

	return nil
}

// ParallelismFromEnv reads WZ_EXTRACT_PARALLELISM with a runtime.NumCPU()
// fallback. Invalid/zero values fall back to default and log a warning.
func ParallelismFromEnv(l logrus.FieldLogger) int {
	v := os.Getenv(envParallelism)
	if v == "" {
		return runtime.NumCPU()
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		l.WithField("value", v).Warnf("invalid %s; using runtime.NumCPU()", envParallelism)
		return runtime.NumCPU()
	}
	return n
}

// wipeCharacterCache removes the {imgOut}/character directory so a fresh
// extraction does not serve stale renders against newly extracted assets.
// Per the design, character-parts/ and character-meta/ are kept and
// overwritten in place by the extraction itself.
func wipeCharacterCache(imgOut string) error {
	target := filepath.Join(imgOut, "character")
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove %s: %w", target, err)
	}
	return nil
}
