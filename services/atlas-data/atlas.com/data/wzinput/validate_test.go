package wzinput

import (
	"archive/zip"
	"bytes"
	"testing"
)

func makeZipEntry(t *testing.T, name string) *zip.File {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("data"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return zr.File[0]
}

func TestValidateZipSlip(t *testing.T) {
	if err := ValidateZipEntry(makeZipEntry(t, "../escape.wz")); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAbsolutePath(t *testing.T) {
	if err := ValidateZipEntry(makeZipEntry(t, "/etc/passwd.wz")); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNonWzExtension(t *testing.T) {
	if err := ValidateZipEntry(makeZipEntry(t, "Item.exe")); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateOkay(t *testing.T) {
	if err := ValidateZipEntry(makeZipEntry(t, "Item.wz")); err != nil {
		t.Fatal(err)
	}
}
