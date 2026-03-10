package guild

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestBBSThreadListEmpty(t *testing.T) {
	input := NewBBSThreadList(nil, nil, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestBBSThreadListWithThreads(t *testing.T) {
	threads := []BBSThreadSummary{
		{Id: 1, PosterId: 100, Title: "Hello", CreatedAt: 116444736000000000, EmoticonId: 0, ReplyCount: 3},
		{Id: 2, PosterId: 200, Title: "Test", CreatedAt: 116444736100000000, EmoticonId: 1, ReplyCount: 0},
	}
	input := NewBBSThreadList(nil, threads, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestBBSThread(t *testing.T) {
	replies := []BBSReply{
		{Id: 1, PosterId: 200, CreatedAt: 116444736000000000, Message: "Nice post!"},
	}
	input := NewBBSThread(1, 100, 116444736000000000, "Hello", "World", 0, replies)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
