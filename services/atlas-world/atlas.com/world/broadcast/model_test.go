package broadcast

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

var fixedNow = time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

func newEntry(id uuid.UUID, characterId uint32, durationSeconds uint32) Entry {
	return Entry{
		Id:              id,
		CharacterId:     characterId,
		Payload:         Payload{SenderName: "Sender"},
		DurationSeconds: durationSeconds,
	}
}

func TestQueueModel_Append(t *testing.T) {
	tests := []struct {
		name    string
		initial QueueModel
		entries []Entry
	}{
		{
			name:    "append to empty queue",
			initial: QueueModel{},
			entries: []Entry{newEntry(uuid.New(), 1, 10)},
		},
		{
			name:    "append preserves existing order and appends to tail",
			initial: QueueModel{Pending: []Entry{newEntry(uuid.New(), 1, 10)}},
			entries: []Entry{newEntry(uuid.New(), 2, 20), newEntry(uuid.New(), 3, 30)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.initial
			expectedIds := make([]uuid.UUID, 0, len(q.Pending)+len(tt.entries))
			for _, e := range q.Pending {
				expectedIds = append(expectedIds, e.Id)
			}
			for _, e := range tt.entries {
				q = q.Append(e)
				expectedIds = append(expectedIds, e.Id)
			}

			if len(q.Pending) != len(expectedIds) {
				t.Fatalf("Pending length = %d, want %d", len(q.Pending), len(expectedIds))
			}
			for i, id := range expectedIds {
				if q.Pending[i].Id != id {
					t.Errorf("Pending[%d].Id = %v, want %v (order not preserved)", i, q.Pending[i].Id, id)
				}
			}
		})
	}
}

func TestQueueModel_ActivateNext(t *testing.T) {
	t.Run("pops head of Pending into Active and stamps times", func(t *testing.T) {
		head := newEntry(uuid.New(), 1, 30)
		tail := newEntry(uuid.New(), 2, 15)
		q := QueueModel{Pending: []Entry{head, tail}}

		next, activated := q.ActivateNext(fixedNow)

		if activated == nil {
			t.Fatal("ActivateNext returned nil activated entry, want non-nil")
		}
		if activated.Id != head.Id {
			t.Errorf("activated.Id = %v, want %v (head of Pending)", activated.Id, head.Id)
		}
		if !activated.ActivatedAt.Equal(fixedNow) {
			t.Errorf("activated.ActivatedAt = %v, want %v", activated.ActivatedAt, fixedNow)
		}
		wantExpires := fixedNow.Add(time.Duration(head.DurationSeconds) * time.Second)
		if !activated.ExpiresAt.Equal(wantExpires) {
			t.Errorf("activated.ExpiresAt = %v, want %v", activated.ExpiresAt, wantExpires)
		}

		if next.Active == nil {
			t.Fatal("next.Active is nil, want non-nil")
		}
		if next.Active.Id != head.Id {
			t.Errorf("next.Active.Id = %v, want %v", next.Active.Id, head.Id)
		}
		if !next.Active.ActivatedAt.Equal(fixedNow) {
			t.Errorf("next.Active.ActivatedAt = %v, want %v", next.Active.ActivatedAt, fixedNow)
		}
		if !next.Active.ExpiresAt.Equal(wantExpires) {
			t.Errorf("next.Active.ExpiresAt = %v, want %v", next.Active.ExpiresAt, wantExpires)
		}

		if len(next.Pending) != 1 {
			t.Fatalf("next.Pending length = %d, want 1 (head removed)", len(next.Pending))
		}
		if next.Pending[0].Id != tail.Id {
			t.Errorf("next.Pending[0].Id = %v, want %v", next.Pending[0].Id, tail.Id)
		}
	})

	t.Run("empty Pending returns nil activated entry and unchanged queue", func(t *testing.T) {
		q := QueueModel{}

		next, activated := q.ActivateNext(fixedNow)

		if activated != nil {
			t.Fatalf("activated = %+v, want nil", activated)
		}
		if next.Active != nil {
			t.Errorf("next.Active = %+v, want nil", next.Active)
		}
		if len(next.Pending) != 0 {
			t.Errorf("next.Pending length = %d, want 0", len(next.Pending))
		}
	})
}

func TestQueueModel_ClearActive(t *testing.T) {
	active := newEntry(uuid.New(), 1, 10)
	pending := []Entry{newEntry(uuid.New(), 2, 20)}
	q := QueueModel{Active: &active, Pending: pending}

	cleared := q.ClearActive()

	if cleared.Active != nil {
		t.Errorf("cleared.Active = %+v, want nil", cleared.Active)
	}
	if len(cleared.Pending) != 1 || cleared.Pending[0].Id != pending[0].Id {
		t.Errorf("cleared.Pending = %+v, want unchanged %+v", cleared.Pending, pending)
	}
}

func TestQueueModel_ActiveExpired(t *testing.T) {
	tests := []struct {
		name string
		q    QueueModel
		now  time.Time
		want bool
	}{
		{
			name: "no active entry is never expired",
			q:    QueueModel{},
			now:  fixedNow,
			want: false,
		},
		{
			name: "now before ExpiresAt is not expired",
			q: QueueModel{Active: &Entry{
				ExpiresAt: fixedNow.Add(1 * time.Second),
			}},
			now:  fixedNow,
			want: false,
		},
		{
			name: "now exactly equal to ExpiresAt is expired (boundary)",
			q: QueueModel{Active: &Entry{
				ExpiresAt: fixedNow,
			}},
			now:  fixedNow,
			want: true,
		},
		{
			name: "now after ExpiresAt is expired",
			q: QueueModel{Active: &Entry{
				ExpiresAt: fixedNow.Add(-1 * time.Second),
			}},
			now:  fixedNow,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.ActiveExpired(tt.now); got != tt.want {
				t.Errorf("ActiveExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueueModel_WaitSeconds(t *testing.T) {
	tests := []struct {
		name string
		q    QueueModel
		now  time.Time
		want uint32
	}{
		{
			name: "idle queue (no active, no pending) is 0",
			q:    QueueModel{},
			now:  fixedNow,
			want: 0,
		},
		{
			name: "active only: remaining time rounded up",
			q: QueueModel{Active: &Entry{
				ExpiresAt: fixedNow.Add(5*time.Second + 200*time.Millisecond),
			}},
			now:  fixedNow,
			want: 6, // ceil(5.2) = 6
		},
		{
			name: "active exactly on a whole second boundary needs no rounding",
			q: QueueModel{Active: &Entry{
				ExpiresAt: fixedNow.Add(5 * time.Second),
			}},
			now:  fixedNow,
			want: 5,
		},
		{
			name: "pending only: sum of durations, no active",
			q: QueueModel{Pending: []Entry{
				{DurationSeconds: 10},
				{DurationSeconds: 15},
			}},
			now:  fixedNow,
			want: 25,
		},
		{
			name: "active remainder plus pending durations",
			q: QueueModel{
				Active: &Entry{ExpiresAt: fixedNow.Add(3*time.Second + 100*time.Millisecond)},
				Pending: []Entry{
					{DurationSeconds: 10},
					{DurationSeconds: 5},
				},
			},
			now:  fixedNow,
			want: 19, // ceil(3.1) + 10 + 5 = 4 + 15
		},
		{
			name: "active already expired contributes 0, not negative",
			q: QueueModel{
				Active:  &Entry{ExpiresAt: fixedNow.Add(-10 * time.Second)},
				Pending: []Entry{{DurationSeconds: 7}},
			},
			now:  fixedNow,
			want: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.WaitSeconds(tt.now); got != tt.want {
				t.Errorf("WaitSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}
