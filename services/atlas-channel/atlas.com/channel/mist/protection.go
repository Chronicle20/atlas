package mist

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Protection is a live protection (Smokescreen) mist as this channel knows
// it: enough to answer "is this character standing in one, and does it belong
// to them or their party?" on the damage path.
//
// The channel keeps its own copy rather than querying atlas-maps because the
// alternative is a synchronous REST round trip on the most latency-sensitive
// path in the service. The cost is a restart gap: the mist consumer starts at
// kafka.LastOffset, so a channel that restarts mid-mist never learns about
// mists created before it came up. That is not a regression -- the same
// restart already loses the AffectedAreaCreated broadcast, so those mists are
// invisible to every client on that channel too -- and it is bounded by the
// longest Smokescreen lifetime (60s at level 30).
type Protection struct {
	id        uuid.UUID
	f         field.Model
	ownerId   uint32
	minX      int16
	minY      int16
	maxX      int16
	maxY      int16
	expiresAt time.Time
}

// Id returns the mist id this protection was created from.
func (p Protection) Id() uuid.UUID { return p.id }

// Field returns the field the protection covers.
func (p Protection) Field() field.Model { return p.f }

// OwnerId returns the casting character. The client's smoke lookup accepts an
// area only if its dwOwnerID is the local character or one of their online
// party members (CAffectedAreaPool::IsSmokeAreaByPoint, v95 @0x434f40), so
// the owner is what the party check is evaluated against.
func (p Protection) OwnerId() uint32 { return p.ownerId }

// ExpiresAt returns the absolute expiry.
func (p Protection) ExpiresAt() time.Time { return p.expiresAt }

// Expired reports whether the protection is past its lifetime as of now.
func (p Protection) Expired(now time.Time) bool { return now.After(p.expiresAt) }

// Contains reports whether the world coordinates fall inside the protection's
// axis-aligned bounding box. Edges are INCLUSIVE, matching atlas-maps'
// Mist.Contains and the atlas-monsters in-rect endpoint -- the rect test
// exists on both sides and the two conventions must not drift.
func (p Protection) Contains(x, y int16) bool {
	return x >= p.minX && x <= p.maxX && y >= p.minY && y <= p.maxY
}
