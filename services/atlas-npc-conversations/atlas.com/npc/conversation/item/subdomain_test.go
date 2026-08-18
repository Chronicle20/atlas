package item

import "testing"

// The subdomain's four values are mirrored by hand in
// tools/catalog-lint/subdomains.go. If this test changes, that file must too.
func TestSubdomainIdentity(t *testing.T) {
	s := ItemConversationSubdomain{}
	if s.Name() != "item.conversation" {
		t.Errorf("Name: got %q", s.Name())
	}
	if s.Path() != "npc-conversations/items" {
		t.Errorf("Path: got %q", s.Path())
	}
	if s.Type() != "item-conversation" {
		t.Errorf("Type: got %q", s.Type())
	}
	for _, ok := range []string{"item-2430008.json", "item-2430013.json"} {
		if !s.EntityIDPattern().MatchString(ok) {
			t.Errorf("pattern should match %q", ok)
		}
	}
	for _, bad := range []string{"quest-1021.json", "item-abc.json", "2430008.json"} {
		if s.EntityIDPattern().MatchString(bad) {
			t.Errorf("pattern should not match %q", bad)
		}
	}
}
