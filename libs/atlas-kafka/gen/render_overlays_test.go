package main

import "testing"

func TestEmitOverlayBlock(t *testing.T) {
	m := Manifest{Topics: []Entry{
		{Token: "COMMAND_TOPIC_A", Cleanup: "delete"},
		{Token: "EVENT_TOPIC_B", Cleanup: "delete"},
	}}

	tests := []struct {
		overlay string
		suffix  string
		want    string
	}{
		{
			overlay: "main",
			suffix:  "-main",
			want:    "      - COMMAND_TOPIC_A=COMMAND_TOPIC_A-main\n      - EVENT_TOPIC_B=EVENT_TOPIC_B-main\n",
		},
		{
			overlay: "pr",
			suffix:  "-PLACEHOLDER_ATLAS_ENV",
			want:    "      - COMMAND_TOPIC_A=COMMAND_TOPIC_A-PLACEHOLDER_ATLAS_ENV\n      - EVENT_TOPIC_B=EVENT_TOPIC_B-PLACEHOLDER_ATLAS_ENV\n",
		},
		{
			overlay: "pr-sparse",
			suffix:  "-PLACEHOLDER_BASELINE_ENVIRONMENT",
			want:    "      - COMMAND_TOPIC_A=COMMAND_TOPIC_A-PLACEHOLDER_BASELINE_ENVIRONMENT\n      - EVENT_TOPIC_B=EVENT_TOPIC_B-PLACEHOLDER_BASELINE_ENVIRONMENT\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.overlay, func(t *testing.T) {
			got := string(m.EmitOverlayBlock(tt.suffix))
			if got != tt.want {
				t.Fatalf("EmitOverlayBlock(%q) = %q, want %q", tt.suffix, got, tt.want)
			}
		})
	}
}

func TestEmitOverlayBlockUsesOverlaySuffixes(t *testing.T) {
	m := Manifest{Topics: []Entry{
		{Token: "COMMAND_TOPIC_A", Cleanup: "delete"},
		{Token: "EVENT_TOPIC_B", Cleanup: "delete"},
	}}

	for overlay, suffix := range overlaySuffixes {
		got := string(m.EmitOverlayBlock(suffix))
		want := "      - COMMAND_TOPIC_A=COMMAND_TOPIC_A" + suffix + "\n      - EVENT_TOPIC_B=EVENT_TOPIC_B" + suffix + "\n"
		if got != want {
			t.Fatalf("EmitOverlayBlock(overlaySuffixes[%q]) = %q, want %q", overlay, got, want)
		}
	}
}

func TestEmitComposeBlock(t *testing.T) {
	m := Manifest{Topics: []Entry{
		{Token: "COMMAND_TOPIC_A", Cleanup: "delete"},
		{Token: "EVENT_TOPIC_B", Cleanup: "delete"},
	}}

	got := string(m.EmitComposeBlock())
	want := "COMMAND_TOPIC_A=COMMAND_TOPIC_A\nEVENT_TOPIC_B=EVENT_TOPIC_B\n"
	if got != want {
		t.Fatalf("EmitComposeBlock() = %q, want %q", got, want)
	}
}
