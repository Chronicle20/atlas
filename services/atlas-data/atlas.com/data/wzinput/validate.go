package wzinput

import (
	"archive/zip"
	"errors"
	"strings"
)

// ValidateZipEntry rejects zip-slip, symlinks, and non-.wz entries.
func ValidateZipEntry(e *zip.File) error {
	name := e.Name
	if strings.Contains(name, "..") || strings.HasPrefix(name, "/") || strings.Contains(name, "\x00") {
		return errors.New("invalid entry path")
	}
	if e.Mode()&0o170000 == 0o120000 {
		return errors.New("symlink entries forbidden")
	}
	if !strings.HasSuffix(strings.ToLower(name), ".wz") {
		return errors.New("only .wz entries allowed")
	}
	return nil
}
