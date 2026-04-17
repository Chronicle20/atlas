package extraction

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const multipartPartName = "zip_file"

func streamZipToTempFile(r *http.Request) (*os.File, int64, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, 0, fmt.Errorf("request must be multipart/form-data: %w", err)
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return nil, 0, fmt.Errorf("missing %q multipart field", multipartPartName)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("unable to read multipart part: %w", err)
		}
		if part.FormName() != multipartPartName {
			_ = part.Close()
			continue
		}
		tmp, err := os.CreateTemp("", "wz-upload-*.zip")
		if err != nil {
			_ = part.Close()
			return nil, 0, fmt.Errorf("unable to create temp file: %w", err)
		}
		n, copyErr := io.Copy(tmp, part)
		_ = part.Close()
		if copyErr != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, 0, fmt.Errorf("unable to spool upload: %w", copyErr)
		}
		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, 0, fmt.Errorf("unable to rewind temp file: %w", err)
		}
		return tmp, n, nil
	}
}

func validateZip(f *os.File) (*zip.Reader, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to stat temp file: %w", err)
	}
	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		return nil, fmt.Errorf("not a valid zip: %w", err)
	}
	for _, e := range zr.File {
		if err := validateEntry(e); err != nil {
			return nil, err
		}
	}
	return zr, nil
}

func validateEntry(e *zip.File) error {
	name := e.Name
	if name == "" {
		return fmt.Errorf("zip entry has empty name")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("zip entry %q contains a path separator", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("zip entry %q contains '..' segment", name)
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("zip entry %q is an absolute path", name)
	}
	if e.FileInfo().IsDir() {
		return fmt.Errorf("zip entry %q is a directory", name)
	}
	mode := e.FileInfo().Mode()
	if !mode.IsRegular() {
		return fmt.Errorf("zip entry %q is not a regular file", name)
	}
	if !strings.EqualFold(filepath.Ext(name), ".wz") {
		return fmt.Errorf("zip entry %q is not a .wz file", name)
	}
	return nil
}

func extractFlat(zr *zip.Reader, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("unable to remove destination: %w", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("unable to create destination: %w", err)
	}
	for _, e := range zr.File {
		if err := writeEntry(e, dst); err != nil {
			return err
		}
	}
	return nil
}

func writeEntry(e *zip.File, dst string) error {
	base := filepath.Base(e.Name)
	target := filepath.Join(dst, base)
	src, err := e.Open()
	if err != nil {
		return fmt.Errorf("unable to open entry %q: %w", e.Name, err)
	}
	defer src.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("unable to create %q: %w", target, err)
	}
	if _, err := io.Copy(out, src); err != nil {
		_ = out.Close()
		return fmt.Errorf("unable to write %q: %w", target, err)
	}
	return out.Close()
}

type uploadDeps struct {
	inputDir string
}

func (u *uploadDeps) handleUpload(l logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(ctx)
		key := TenantKey(t)

		m, ok := TryAcquire(key)
		if !ok {
			writeJSONError(w, http.StatusConflict, "tenant busy: another upload or extraction is in progress")
			return
		}
		defer Release(m)

		started := time.Now()

		tmp, size, err := streamZipToTempFile(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		defer func() {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}()

		zr, err := validateZip(tmp)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		dst := ResolveTenantInputDir(u.inputDir, t)
		if err := extractFlat(zr, dst); err != nil {
			l.WithError(err).Errorf("Upload extract failed for tenant [%s].", t.Id().String())
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		l.Infof("Upload complete: tenant=%s bytes=%d entries=%d duration=%s",
			t.Id().String(), size, len(zr.File), time.Since(started))

		w.WriteHeader(http.StatusAccepted)
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
