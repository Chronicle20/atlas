package extraction

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type statusDeps struct {
	inputDir     string
	outputXmlDir string
}

type statusAttrs struct {
	FileCount  int     `json:"fileCount"`
	TotalBytes int64   `json:"totalBytes"`
	UpdatedAt  *string `json:"updatedAt"`
}

type statusResource struct {
	Type       string      `json:"type"`
	Id         string      `json:"id"`
	Attributes statusAttrs `json:"attributes"`
}

type statusEnvelope struct {
	Data statusResource `json:"data"`
}

func (s *statusDeps) renderInputStatus(l logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(ctx)
		dir := ResolveTenantInputDir(s.inputDir, t)
		attrs := topLevelStatus(dir, ".wz")
		writeStatus(w, "wzInputStatus", TenantPath(t), attrs)
		_ = l
	}
}

func (s *statusDeps) renderExtractionStatus(l logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(ctx)
		dir := ResolveTenantOutputDir(s.outputXmlDir, t)
		attrs := recursiveStatus(dir, ".xml")
		writeStatus(w, "wzExtractionStatus", TenantPath(t), attrs)
		_ = l
	}
}

func topLevelStatus(dir, wantExt string) statusAttrs {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return statusAttrs{}
	}
	var attrs statusAttrs
	var maxModTime time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), wantExt) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		attrs.FileCount++
		attrs.TotalBytes += info.Size()
		if info.ModTime().After(maxModTime) {
			maxModTime = info.ModTime()
		}
	}
	if attrs.FileCount > 0 && !maxModTime.IsZero() {
		s := maxModTime.UTC().Format(time.RFC3339)
		attrs.UpdatedAt = &s
	}
	return attrs
}

func recursiveStatus(dir, wantExt string) statusAttrs {
	var attrs statusAttrs
	var maxModTime time.Time
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), wantExt) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		attrs.FileCount++
		attrs.TotalBytes += info.Size()
		if info.ModTime().After(maxModTime) {
			maxModTime = info.ModTime()
		}
		return nil
	})
	if err != nil || attrs.FileCount == 0 {
		return statusAttrs{}
	}
	if !maxModTime.IsZero() {
		s := maxModTime.UTC().Format(time.RFC3339)
		attrs.UpdatedAt = &s
	}
	return attrs
}

func writeStatus(w http.ResponseWriter, resourceType, id string, attrs statusAttrs) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	env := statusEnvelope{
		Data: statusResource{
			Type:       resourceType,
			Id:         id,
			Attributes: attrs,
		},
	}
	_ = json.NewEncoder(w).Encode(env)
}
