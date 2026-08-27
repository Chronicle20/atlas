package wz

// TraceEvent records what the property decoder actually consumed while
// walking one node of the property tree: where its decode started and
// ended, and enough context (Kind, Type, Detail) to tell a healed drift
// from a correct read.
//
// TraceEvent is a diagnostic value, not a public parse contract — its
// shape may grow as later divergence-diagnosis tasks need more detail.
type TraceEvent struct {
	// Path is the property path from the image root, e.g. "/info/state".
	Path string
	// Kind discriminates what the event describes: "list", "stringblock",
	// "prop", "sub", "extended", "canvas", or "uol".
	Kind string
	// Name is the property or tag name this event concerns.
	Name string
	// Type is the raw property-type byte for Kind == "prop" events; zero
	// for events that have no single type byte (e.g. "list").
	Type byte
	// StartOff and EndOff are reader positions (Reader.Pos) bracketing the
	// bytes this event's decode consumed.
	StartOff int64
	EndOff   int64
	// Detail carries kind-specific context, e.g. "count=3" for a list or
	// "declaredSize=12 endPos=340 actualEnd=338" for a type-9 sub-object —
	// the actualEnd/endPos mismatch is what surfaces an under- or over-read
	// that the type-9 recovery reseek would otherwise silently heal.
	Detail string
	// DeclaredEnd is the reader position a type-9 sub-object's own declared
	// size says its decode should end at, and ActualEnd is where the decode
	// actually ended, captured before the recovery reseek heals any drift.
	// Both are zero for events where Kind != "sub".
	DeclaredEnd int64
	ActualEnd   int64
}

// SetTrace installs fn as the parse trace hook for wz and every sub-file
// view of it (NewSubFile delegates through parent, mirroring LockParse).
//
// Concurrency contract: SetTrace must be called before any Properties()
// call on wz or any of its sub-files — there is no lock protecting the
// trace field itself, matching how encryptionKey/versionHash are set once
// during Open and never mutated after publication. The hook fires
// synchronously on the goroutine holding File.parseMu; fn must not re-enter
// the reader (no Seek, no Read) or call back into any wz parse function.
//
// fn is nil by default (production). Every emit site guards on a nil
// check before constructing a TraceEvent, so the hook costs nothing when
// unset.
func (wz *File) SetTrace(fn func(TraceEvent)) {
	if wz.parent != nil {
		wz.parent.SetTrace(fn)
		return
	}
	wz.trace = fn
}

// traceHook resolves the trace hook through parent (so a hook set on a
// parent File fires exactly once per node reached through any sub-file
// view of it, matching LockParse's delegation shape). Call sites guard on
// this before constructing a TraceEvent, so the nil-hook path costs one
// pointer check and no allocation.
func (wz *File) traceHook() func(TraceEvent) {
	if wz.parent != nil {
		return wz.parent.traceHook()
	}
	return wz.trace
}

// emitPropTrace emits a Kind: "prop" TraceEvent for a scalar property once
// its value has been read (StartOff/EndOff bracket the value bytes, not the
// name/type tag already consumed by the caller). A plain function rather
// than a closure so the nil-hook path costs one comparison and no
// allocation.
func emitPropTrace(hook func(TraceEvent), r *Reader, path, name string, propType byte, start int64) {
	if hook == nil {
		return
	}
	end, err := r.Pos()
	if err != nil {
		return
	}
	hook(TraceEvent{Path: path, Kind: "prop", Name: name, Type: propType, StartOff: start, EndOff: end})
}
