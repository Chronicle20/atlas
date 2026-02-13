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
	version := fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())
	xmlOutPath := filepath.Join(p.outputXmlDir, t.Id().String(), t.Region(), version)
	imgOutPath := filepath.Join(p.outputImgDir, t.Id().String(), t.Region(), version)
	return p.runExtraction(l, xmlOutPath, imgOutPath, xmlOnly, imagesOnly)
}

func (p *processorImpl) runExtraction(l logrus.FieldLogger, xmlOutPath, imgOutPath string, xmlOnly, imagesOnly bool) error {
	wzFiles, err := filepath.Glob(filepath.Join(p.inputDir, "*.wz"))
	if err != nil {
		return fmt.Errorf("unable to list WZ files: %w", err)
	}
	if len(wzFiles) == 0 {
		return fmt.Errorf("no WZ files found in [%s]", p.inputDir)
	}

	l.Infof("Found [%d] WZ files in [%s].", len(wzFiles), p.inputDir)

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
