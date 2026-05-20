package wzinput

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const maxUploadBytes = 512 << 20 // 512 MiB

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
			if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
				http.Error(w, "parse multipart: "+err.Error(), http.StatusBadRequest)
				return
			}
			file, _, err := r.FormFile("zip_file")
			if err != nil {
				http.Error(w, "missing zip_file", http.StatusBadRequest)
				return
			}
			defer file.Close()
			buf := &bytes.Buffer{}
			if _, err := io.Copy(buf, file); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			if err != nil {
				http.Error(w, "bad zip", http.StatusBadRequest)
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
				data, err := io.ReadAll(rc)
				_ = rc.Close()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				key := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s",
					scope.Key, t.Region(), t.MajorVersion(), t.MinorVersion(), entry.Name)
				if err := mc.Put(r.Context(), mc.Cfg().BucketWZ, key, bytes.NewReader(data), int64(len(data)), "application/octet-stream"); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
