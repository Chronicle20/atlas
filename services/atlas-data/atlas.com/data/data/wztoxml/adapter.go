// Package wztoxml converts an in-memory WZ tree to HaRepacker-compatible XML
// files on disk. Output mirrors what atlas-wz-extractor's deleted
// xml/serializer.go used to produce, so the existing atlas-data domain readers
// (which still consume `.img.xml` files via xml.FromPathProvider) work
// unchanged.
//
// Layout (rooted at outputDir, mirroring the WZ directory tree):
//
//	{outputDir}/{wzName}.wz/{dirPath}/{imageName}.img.xml
package wztoxml

import (
	stdxml "encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/wzxml"
)

// SerializeToDirectory serializes a parsed WZ file to HaRepacker-compatible XML
// files. Output layout:
//
//	{outputDir}/{wzName}.wz/{dirPath}/{imageName}.img.xml
func SerializeToDirectory(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return fmt.Errorf("wz file [%s] has no root directory", f.Name())
	}
	archiveName := f.Name() + ".wz"
	wzDir := filepath.Join(outputDir, archiveName)
	if err := os.MkdirAll(wzDir, 0o755); err != nil {
		return fmt.Errorf("create output directory [%s]: %w", wzDir, err)
	}
	failed, total, err := serializeDirectory(l, archiveName, root, wzDir)
	if err != nil {
		return err
	}
	if failed > 0 {
		l.Warnf("wz archive [%s]: %d of %d images failed to serialize", archiveName, failed, total)
	} else {
		l.Infof("wz archive [%s]: %d of %d images failed to serialize", archiveName, failed, total)
	}
	return nil
}

// serializeDirectory recursively serializes dir's images (and its
// sub-directories') to outputPath, logging one per-image warning for each
// failure and returning the failure/total image counts so the caller can
// report one per-archive summary line. It never aggregates per-property
// detail — that granularity is forbidden by the Observability NFR.
func serializeDirectory(l logrus.FieldLogger, archiveName string, dir *wz.Directory, outputPath string) (failed int, total int, err error) {
	for _, img := range dir.Images() {
		total++
		if serr := SerializeImage(img, outputPath); serr != nil {
			failed++
			l.WithError(serr).Warnf("wz archive [%s]: unable to serialize image [%s]", archiveName, img.Name())
		}
	}
	for _, sub := range dir.Directories() {
		subPath := filepath.Join(outputPath, sub.Name())
		if err := os.MkdirAll(subPath, 0o755); err != nil {
			return failed, total, fmt.Errorf("create directory [%s]: %w", subPath, err)
		}
		subFailed, subTotal, err := serializeDirectory(l, archiveName, sub, subPath)
		failed += subFailed
		total += subTotal
		if err != nil {
			return failed, total, err
		}
	}
	return failed, total, nil
}

// SerializeImage writes a single WZ image to {outputPath}/{imageName}.img.xml.
func SerializeImage(img *wz.Image, outputPath string) error {
	xmlPath := filepath.Join(outputPath, img.Name()+".img.xml")
	f, err := os.Create(xmlPath)
	if err != nil {
		return fmt.Errorf("create xml file [%s]: %w", xmlPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(stdxml.Header); err != nil {
		return err
	}
	e := stdxml.NewEncoder(f)
	e.Indent("", "  ")
	root := wzxml.Element{
		XMLName: stdxml.Name{Local: "imgdir"},
		Name:    img.Name() + ".img",
	}
	props, err := img.Properties()
	if err != nil {
		return fmt.Errorf("wztoxml adapter: %s: %w", img.Name(), err)
	}
	root.Children = wzxml.PropertiesToElements(props)
	if err := e.Encode(root); err != nil {
		return fmt.Errorf("encode xml for [%s]: %w", img.Name(), err)
	}
	return nil
}
