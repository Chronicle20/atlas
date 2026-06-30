package matrix

import "testing"

func TestEveryVersionKeyHasShortLabel(t *testing.T) {
	for _, k := range VersionKeys {
		if _, ok := shortLabels[k]; !ok {
			t.Errorf("VersionKeys entry %q has no short label in shortLabels", k)
		}
	}
}

func TestEveryVersionKeyHasTemplateFile(t *testing.T) {
	for _, k := range VersionKeys {
		if _, ok := templateFiles[k]; !ok {
			t.Errorf("VersionKeys entry %q has no template filename in templateFiles", k)
		}
	}
}
