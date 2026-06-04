package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestBuddyInviteByteOutput verifies the byte output of BuddyInvite across all tenant variants.
// IDA evidence — CWvsContext::OnFriendResult case 9:
//   v83 @0xa3f2e8: Decode4 originatorId, DecodeStr originatorName, GW_Friend(39), Decode1 inShop — NO jobId/level.
//   v87 @0xad7ae5: ... DecodeStr originatorName, Decode4 jobId, Decode4 level, GW_Friend(39), Decode1 inShop.
//   v95 @0xa12630: same as v87.
//   JMS185 @0xb2a873: same as v87.
// Wire layout: mode(1)+originatorId(4)+name(2+len)+[jobId(4)+level(4)]+GW_Friend(39)+inShop(1).
// originatorName="TestPlayer" → 2+10=12 bytes. GW_Friend = 4+13+1+4+17 = 39 bytes.
//   no-fields:   1+4+12+39+1   = 57 bytes
//   with-fields: 1+4+12+4+4+39+1 = 65 bytes
func TestBuddyInviteByteOutput(t *testing.T) {
	const name = "TestPlayer"
	// offset of the first field after the ascii string: mode(1)+originatorId(4)+lenPrefix(2)+len(name)
	nameStart := 1 + 4 + 2
	afterName := nameStart + len(name)

	cases := []struct {
		variant     pt.TenantVariant
		wantBytes   int
		hasJobLevel bool
	}{
		{pt.Variants[0], 57, false}, // GMS v28  — no jobId/level
		{pt.Variants[1], 57, false}, // GMS v83  — no jobId/level
		{pt.Variants[2], 65, true},  // GMS v87  — with jobId+level
		{pt.Variants[3], 65, true},  // GMS v95  — with jobId+level
		{pt.Variants[4], 65, true},  // JMS v185 — with jobId+level
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := NewBuddyInvite(9, 1000, 2000, name, 510, 120)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != tc.wantBytes {
				t.Fatalf("byte count: got %d, want %d", len(got), tc.wantBytes)
			}
			if tc.hasJobLevel {
				// jobId and level are two int4 LE right after the name string.
				gotJob := binary.LittleEndian.Uint32(got[afterName : afterName+4])
				gotLvl := binary.LittleEndian.Uint32(got[afterName+4 : afterName+8])
				if gotJob != 510 {
					t.Errorf("jobId at offset %d: got %d, want 510", afterName, gotJob)
				}
				if gotLvl != 120 {
					t.Errorf("level at offset %d: got %d, want 120", afterName+4, gotLvl)
				}
				// GW_Friend buffer (FriendId == actorId) starts 8 bytes after the name.
				gotFriendId := binary.LittleEndian.Uint32(got[afterName+8 : afterName+12])
				if gotFriendId != 1000 {
					t.Errorf("GW_Friend.FriendId at offset %d: got %d, want 1000 (actorId)", afterName+8, gotFriendId)
				}
			} else {
				// No jobId/level on wire: GW_Friend buffer starts immediately after the name,
				// 8 bytes earlier than the with-fields layout.
				gotFriendId := binary.LittleEndian.Uint32(got[afterName : afterName+4])
				if gotFriendId != 1000 {
					t.Errorf("GW_Friend.FriendId at offset %d: got %d, want 1000 (actorId)", afterName, gotFriendId)
				}
			}
		})
	}
}

func TestBuddyInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			hasJobLevel := v.Region != "GMS" || v.MajorVersion >= 87
			input := NewBuddyInvite(9, 1000, 2000, "TestPlayer", 510, 120)
			output := Invite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.OriginatorId() != input.OriginatorId() {
				t.Errorf("originatorId: got %v, want %v", output.OriginatorId(), input.OriginatorId())
			}
			if output.OriginatorName() != input.OriginatorName() {
				t.Errorf("originatorName: got %v, want %v", output.OriginatorName(), input.OriginatorName())
			}
			if output.ActorId() != input.ActorId() {
				t.Errorf("actorId: got %v, want %v", output.ActorId(), input.ActorId())
			}
			if hasJobLevel {
				if output.JobId() != input.JobId() {
					t.Errorf("jobId: got %v, want %v", output.JobId(), input.JobId())
				}
				if output.Level() != input.Level() {
					t.Errorf("level: got %v, want %v", output.Level(), input.Level())
				}
			} else {
				// jobId/level are off-wire for v83/v28; they must read back as zero.
				if output.JobId() != 0 {
					t.Errorf("jobId (off-wire): got %v, want 0", output.JobId())
				}
				if output.Level() != 0 {
					t.Errorf("level (off-wire): got %v, want 0", output.Level())
				}
			}
		})
	}
}
