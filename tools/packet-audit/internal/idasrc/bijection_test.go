package idasrc

import "testing"

func TestBijection_MissingAndExtra(t *testing.T) {
	cs := &CaseSet{}
	cs.add(1)
	cs.add(2)
	cs.add(9) // client has cases 1,2,9
	modes := []ModeBinding{
		{FName: "X#A", Case: 1},
		{FName: "X#B", Case: 9},
		{FName: "X#Ghost", Case: 7}, // case 7 not in client -> extra
	}
	res := Bijection(cs, modes)
	if len(res.Missing) != 1 || res.Missing[0] != 2 {
		t.Fatalf("missing=%v want [2]", res.Missing)
	}
	if len(res.Extra) != 1 || res.Extra[0].FName != "X#Ghost" {
		t.Fatalf("extra=%v want [X#Ghost]", res.Extra)
	}
}

func TestBijection_NilClient(t *testing.T) {
	// No case-label info: nothing missing (we can't claim a case is unhandled),
	// and every Atlas mode is "extra" only if we KNOW the client lacks it — with
	// nil client we cannot, so Extra is empty too.
	res := Bijection(nil, []ModeBinding{{FName: "X#A", Case: 1}})
	if len(res.Missing) != 0 || len(res.Extra) != 0 {
		t.Fatalf("nil client should yield empty result, got %+v", res)
	}
}
