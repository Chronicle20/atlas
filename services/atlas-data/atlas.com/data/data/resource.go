package data

import (
	"archive/zip"
	"atlas-data/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/data").Subrouter()
		r.HandleFunc("", rest.RegisterHandler(l)(si)("upload", uploadData)).Methods(http.MethodPatch)
	}
}

func uploadData(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(d.Context())
		d.Logger().Debugf("Processing .zip for tenant [%s], region [%s], version [%d.%d].", t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())
		r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

		err := r.ParseMultipartForm(1 << 30) // 1GB max file size
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to process zip.")
			w.WriteHeader(http.StatusRequestHeaderFieldsTooLarge)
			return
		}

		// Get file from request
		file, handler, err := r.FormFile("zip_file")
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to process zip.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		uploadDir := os.Getenv("ZIP_DIR")

		// Save ZIP file to disk
		tenantDir := filepath.Join(uploadDir, t.Id().String(), t.Region())
		zipPath := filepath.Join(tenantDir, handler.Filename)

		if err := os.MkdirAll(tenantDir, os.ModePerm); err != nil {
			d.Logger().WithError(err).Errorf("Unable to process zip.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		outFile, err := os.Create(zipPath)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to process zip.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer outFile.Close()

		// Stream file contents to disk
		_, err = io.Copy(outFile, file)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to process zip.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = unzip(zipPath, tenantDir)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to process zip.")
			http.Error(w, "Failed to extract ZIP file", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func unzip(zipPath, dest string) error {
	// Open the ZIP file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}

	// Extract each file
	for _, file := range r.File {
		filePath := filepath.Join(dest, file.Name)

		// Ensure parent directories exist
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		} else {
			os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		}

		// Extract file contents
		destFile, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		srcFile, err := file.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return err
		}
	}

	fmt.Println("ZIP extracted to:", dest)
	return nil
}
