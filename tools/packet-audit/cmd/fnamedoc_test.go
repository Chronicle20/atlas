package cmd

import "testing"

func TestApplyCommentInsertsAboveStruct(t *testing.T) {
	src := "package x\n\ntype Foo struct {\n\ta byte\n}\n"
	out, state := applyComment(src, "Foo", "// packet-audit:fname C::OnFoo")
	if state != commentMissing {
		t.Fatalf("state = %v, want missing", state)
	}
	want := "package x\n\n// packet-audit:fname C::OnFoo\ntype Foo struct {\n\ta byte\n}\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestApplyCommentDriftRewrites(t *testing.T) {
	src := "// packet-audit:fname C::Old\ntype Foo struct {\n}\n"
	out, state := applyComment(src, "Foo", "// packet-audit:fname C::New")
	if state != commentDrift {
		t.Fatalf("state = %v, want drift", state)
	}
	if out != "// packet-audit:fname C::New\ntype Foo struct {\n}\n" {
		t.Errorf("drift not rewritten: %q", out)
	}
}

func TestApplyCommentOKWhenCurrent(t *testing.T) {
	src := "// packet-audit:fname C::Foo\ntype Foo struct {\n}\n"
	_, state := applyComment(src, "Foo", "// packet-audit:fname C::Foo")
	if state != commentOK {
		t.Errorf("state = %v, want ok", state)
	}
}

func TestApplyCommentListBlockGetsSeparator(t *testing.T) {
	// A preceding doc block containing a list item requires a blank `//`
	// separator before the appended fname line to stay gofmt-clean.
	src := "// Foo does things:\n//   - one\n//   - two\ntype Foo struct {\n}\n"
	out, _ := applyComment(src, "Foo", "// packet-audit:fname C::Foo")
	want := "// Foo does things:\n//   - one\n//   - two\n//\n// packet-audit:fname C::Foo\ntype Foo struct {\n}\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestApplyCommentPlainDocNoSeparator(t *testing.T) {
	src := "// Foo is a packet.\ntype Foo struct {\n}\n"
	out, _ := applyComment(src, "Foo", "// packet-audit:fname C::Foo")
	want := "// Foo is a packet.\n// packet-audit:fname C::Foo\ntype Foo struct {\n}\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestCodecStructsFindsOperationReceivers(t *testing.T) {
	src := `package x
type Foo struct{}
func (f Foo) Operation() string { return "X" }
type Bar struct{}
func (b *Bar) Operation() string { return "Y" }
type NotACodec struct{}
`
	got := codecStructs(src)
	if len(got) != 2 {
		t.Fatalf("got %v, want [Foo Bar]", got)
	}
}
