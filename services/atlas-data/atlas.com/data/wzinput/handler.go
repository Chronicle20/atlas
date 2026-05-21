package wzinput

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const multipartPartName = "zip_file"

// streamZipToTempFile reads the request's multipart body part-by-part until it
// finds the `zip_file` field, then streams its bytes into a freshly-created
// temp file on disk. Returns the open file (rewound to offset 0) and its size.
//
// Why a temp file instead of bytes.Buffer / ParseMultipartForm: production
// atlas.zip is ~1.6 GB. Loading that into memory via ParseMultipartForm +
// io.Copy(buf, file) repeatedly truncated the resulting buffer on the
// PR-544 env (atlas-data returned "bad zip" because the buffer was missing
// the central directory). Streaming to disk matches the old extractor's
// upload.go pattern, keeps memory usage flat, and lets zip.NewReader read
// the full file via random access.
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
			return nil, 0, fmt.Errorf("read multipart part: %w", err)
		}
		if part.FormName() != multipartPartName {
			_ = part.Close()
			continue
		}
		tmp, err := os.CreateTemp("", "wz-upload-*.zip")
		if err != nil {
			_ = part.Close()
			return nil, 0, fmt.Errorf("create temp file: %w", err)
		}
		n, copyErr := io.Copy(tmp, part)
		_ = part.Close()
		if copyErr != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, 0, fmt.Errorf("spool upload: %w", copyErr)
		}
		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, 0, fmt.Errorf("rewind temp file: %w", err)
		}
		return tmp, n, nil
	}
}

// uploadHandler is the inner handler used by the resource initializer.
func uploadHandler(mc *minio.Client) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			t := tenant.MustFromContext(d.Context())
			scope, err := ResolveScope(r, t)
			if err != nil {
				if errors.Is(err, ErrSharedRequiresOperator) {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			tmp, size, err := streamZipToTempFile(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer func() {
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
			}()
			zr, err := zip.NewReader(tmp, size)
			if err != nil {
				http.Error(w, fmt.Sprintf("bad zip: %v", err), http.StatusBadRequest)
				return
			}
			for _, entry := range zr.File {
				if err := ValidateZipEntry(entry); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				rc, err := entry.Open()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				key := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s",
					scope.Key, t.Region(), t.MajorVersion(), t.MinorVersion(), entry.Name)
				putErr := mc.Put(r.Context(), mc.Cfg().BucketWZ, key, rc, int64(entry.UncompressedSize64), "application/octet-stream")
				_ = rc.Close()
				if putErr != nil {
					http.Error(w, putErr.Error(), http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
