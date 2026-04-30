package character

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestRequestCreateCharacterProvider_GmMeso(t *testing.T) {
	p := RequestCreateCharacterProvider(uuid.New(), 1, world.Id(0), "Hero", 200, 0, 0, 0, 0, 0, 0, job.Id(112), 0, 0, 0, 0, _map.Id(0), 2, 12345)
	msgs, err := p()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	body := string(msgs[0].Value)
	if !strings.Contains(body, `"gm":2`) || !strings.Contains(body, `"meso":12345`) {
		t.Fatalf("expected gm/meso in body, got %s", body)
	}
}
