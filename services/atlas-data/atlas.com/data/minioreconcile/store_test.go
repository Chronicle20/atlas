package minioreconcile

import "testing"

func TestParseTenantID(t *testing.T) {
	cases := map[string]string{
		"tenants/1cccd449-6751-4cdd-9b1a-2c33f4b6834d/": "1cccd449-6751-4cdd-9b1a-2c33f4b6834d",
		"tenants/abc/": "abc",
		"tenants/":     "",  // no id
		"shared/x/":    "",  // wrong prefix
		"tenants/a/b/": "a", // only first segment
	}
	for in, want := range cases {
		if got := parseTenantID(in); got != want {
			t.Errorf("parseTenantID(%q)=%q want %q", in, got, want)
		}
	}
}
