package extraction

import (
	wzimage "atlas-wz-extractor/image"
	"atlas-wz-extractor/wz"
	wzxml "atlas-wz-extractor/xml"
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error
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
	xmlOutPath := filepath.Join(p.outputXmlDir, tenantPath)
	imgOutPath := filepath.Join(p.outputImgDir, tenantPath)
	return p.runExtraction(l, inputPath, xmlOutPath, imgOutPath, xmlOnly, imagesOnly)
}

func (p *processorImpl) runExtraction(l logrus.FieldLogger, inputPath, xmlOutPath, imgOutPath string, xmlOnly, imagesOnly bool) error {
	wzFiles, err := filepath.Glob(filepath.Join(inputPath, "*.wz"))
	if err != nil {
		return fmt.Errorf("unable to list WZ files: %w", err)
	}
	if len(wzFiles) == 0 {
		return fmt.Errorf("no WZ files found in [%s]", inputPath)
	}

	l.Infof("Found [%d] WZ files in [%s].", len(wzFiles), inputPath)

	for _, wzPath := range wzFiles {
		wzName := filepath.Base(wzPath)
		l.Infof("Processing [%s].", wzName)

		f, err := wz.Open(l, wzPath)
		if err != nil {
			l.WithError(err).Errorf("Unable to open WZ file [%s].", wzName)
			continue
		}

		if !imagesOnly {
			if err := wzxml.SerializeToDirectory(l, f, xmlOutPath); err != nil {
				l.WithError(err).Errorf("Unable to serialize [%s] to XML.", wzName)
			}
		}

		if !xmlOnly {
			if err := wzimage.ExtractIcons(l, f, imgOutPath); err != nil {
				l.WithError(err).Errorf("Unable to extract icons from [%s].", wzName)
			}
		}

		f.Close()
	}
	return nil
}
