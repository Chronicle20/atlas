package cmd

import "testing"

// TestFnameTokenMatchesBareSubSymbol is a regression test for task-229 Ruling 1:
// the roster scraper's fnameToken regex required a literal "::", so a bare IDA
// sub_XXXXXX fname (no class qualifier) could never be scraped out of a pending
// roster. Three ops in task-229 carry a bare sub_ form as their registry
// PRIMARY fname (sub_841AA5, sub_955499), so the scraper must recognize it.
func TestFnameTokenMatchesBareSubSymbol(t *testing.T) {
	got := fnameToken.FindAllString("candidate: sub_841AA5 seen in roster", -1)
	if len(got) != 1 || got[0] != "sub_841AA5" {
		t.Fatalf("expected [sub_841AA5], got %v", got)
	}
}

// TestFnameTokenStillMatchesClassMethod is a regression guard for the existing
// Class::Method token form — the fix must not narrow or break it.
func TestFnameTokenStillMatchesClassMethod(t *testing.T) {
	got := fnameToken.FindAllString("candidate: CWvsContext::SendMobSummonItemUseRequest seen", -1)
	if len(got) != 1 || got[0] != "CWvsContext::SendMobSummonItemUseRequest" {
		t.Fatalf("expected [CWvsContext::SendMobSummonItemUseRequest], got %v", got)
	}
}

// TestFnameTokenRejectsBareEnglishWords proves the sub_ extension is not
// over-broad: ordinary English words that happen to start with "sub" (subject,
// submit) must NOT be scraped as fnames.
func TestFnameTokenRejectsBareEnglishWords(t *testing.T) {
	for _, s := range []string{"subject", "submit", "subroutine call", "sub-total"} {
		if got := fnameToken.FindAllString(s, -1); len(got) != 0 {
			t.Fatalf("input %q: expected no matches, got %v", s, got)
		}
	}
}

// TestFnameTokenRejectsEmbeddedSubSymbol proves the sub_ alternative is
// word-boundary anchored: a sub_XXXX sequence embedded in a larger identifier is
// not a bare IDA symbol and must not be scraped into the roster, in either
// direction (leading qualifier or trailing suffix).
func TestFnameTokenRejectsEmbeddedSubSymbol(t *testing.T) {
	for _, s := range []string{"prefix_sub_841AA5", "sub_841AA5_thunk", "xsub_841AA5", "sub_841AA5z"} {
		if got := fnameToken.FindAllString(s, -1); len(got) != 0 {
			t.Fatalf("input %q: expected no matches, got %v", s, got)
		}
	}
}
